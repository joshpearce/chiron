import PDFKit
import PencilKit
import SwiftUI

/// A PDF open in the reader: which page the reader is on, kept on the
/// server so the other device opens there too.
@MainActor
final class DocumentSession: ObservableObject, Identifiable {
    let id: String
    let title: String
    let pages: Int
    let fileURL: URL
    @Published private(set) var page: Int
    private(set) var position: Double
    /// A page the app asked the view to show (the harness, a link).
    @Published var requestedPage: Int?
    @Published var captureRequested: String?
    /// The harness's way of dragging the Pencil under the select tool:
    /// a selection from one point to another, in page points.
    @Published var selectRequested: SelectRequest?
    struct SelectRequest {
        let id = UUID()
        let page: Int
        let from, to: CGPoint
    }
    var onTurn: ((Int, Double) -> Void)?
    /// Ink per page, in the page's own points, with the version the server
    /// last gave for each and the pages drawn on since.
    @Published private(set) var ink: [Int: PKDrawing] = [:]
    var inkVersions: [Int: Int] = [:]
    private var dirtyInk: Set<Int> = []
    var onInk: ((Int) -> Void)?
    /// Select gives the page to PDFKit (text selection, links); pen and
    /// eraser give it to the ink layer over the page.
    enum Tool: String { case select, pen, eraser }
    @Published var tool: Tool = .pen
    /// The text selected on the page, as the harness sees it.
    @Published var selection = ""
    #if DEBUG
    var highlightProbe = ""
    #endif

    /// The Pencil Pro's double-tap: pen to eraser and back; from select,
    /// to the pen.
    func flipEraser() {
        tool = tool == .pen ? .eraser : .pen
    }

    /// The Pencil Pro's squeeze: round the three tools.
    func nextTool() {
        switch tool {
        case .select: tool = .pen
        case .pen: tool = .eraser
        case .eraser: tool = .select
        }
    }
    /// Pages with a canvas laid over them, for the harness to see the
    /// overlays came up.
    var overlaidPages: Set<Int> = []
    /// A passage selected on a page, sent on: a summary, a primer or a
    /// book of its own, with the document and page as where it came from.
    var onCapture: ((String, Int) -> Void)?

    func captured(_ text: String, page: Int) {
        let passage = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !passage.isEmpty else { return }
        onCapture?(passage, page)
    }

    init(id: String, title: String, pages: Int, page: Int, position: Double, fileURL: URL) {
        self.id = id; self.title = title; self.pages = pages; self.fileURL = fileURL
        self.page = page; self.position = position
    }

    /// The view reports where the reader is.
    func turned(to page: Int, position: Double) {
        guard page != self.page || position != self.position else { return }
        self.page = page
        self.position = position
        onTurn?(page, position)
    }

    /// Show a page, and record it as read.
    func go(to page: Int) {
        requestedPage = page
        turned(to: page, position: 0)
    }

    /// The server's ink, page by page, when the document opens.
    func adopt(ink pages: [Int: PageInk]) {
        for (page, p) in pages {
            inkVersions[page] = p.version
            if let data = Data(base64Encoded: p.inkB64), let d = try? PKDrawing(data: data) {
                ink[page] = d
            }
        }
    }

    /// The reader drew on a page.
    func drew(on page: Int, _ drawing: PKDrawing) {
        ink[page] = drawing
        dirtyInk.insert(page)
        onInk?(page)
    }

    /// The other device's ink on this page lies over ours.
    func merge(_ theirs: PKDrawing, on page: Int) {
        ink[page] = (ink[page] ?? PKDrawing()).appending(theirs)
    }

    func takeDirtyInk() -> [Int] {
        let pages = dirtyInk.sorted()
        dirtyInk = []
        return pages
    }

    func markInkDirty(_ page: Int) { dirtyInk.insert(page) }
}

struct DocumentReaderView: View {
    @EnvironmentObject var library: Library
    @ObservedObject var doc: DocumentSession

    var body: some View {
        NavigationStack {
            PDFKitView(doc: doc)
                .ignoresSafeArea(edges: .bottom)
                .onPencilDoubleTap { _ in doc.flipEraser() }
                .onPencilSqueeze { phase in
                    if case .ended = phase { doc.nextTool() }
                }
                .navigationTitle(doc.title)
                .toolbarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Button { library.closeBook() } label: {
                            Label("Bookshelf", systemImage: "books.vertical")
                        }
                        .accessibilityLabel("Bookshelf")
                    }
                    ToolbarItemGroup(placement: .topBarTrailing) {
                        Button { doc.tool = .select } label: { Label("Select", systemImage: "character.cursor.ibeam") }
                            .tint(doc.tool == .select ? .accentColor : .secondary)
                            .accessibilityLabel("Select")
                            .accessibilityAddTraits(doc.tool == .select ? .isSelected : [])
                        Button { doc.tool = .pen } label: { Label("Pen", systemImage: "pencil.tip") }
                            .tint(doc.tool == .pen ? .accentColor : .secondary)
                            .accessibilityLabel("Pen")
                            .accessibilityAddTraits(doc.tool == .pen ? .isSelected : [])
                        Button { doc.tool = .eraser } label: { Label("Eraser", systemImage: "eraser") }
                            .tint(doc.tool == .eraser ? .accentColor : .secondary)
                            .accessibilityLabel("Eraser")
                            .accessibilityAddTraits(doc.tool == .eraser ? .isSelected : [])
                    }
                    ToolbarSpacer(.fixed, placement: .topBarTrailing)
                    ToolbarItem(placement: .topBarTrailing) {
                        Text(doc.pages > 0 ? "\(doc.page + 1) of \(doc.pages)" : "")
                            .font(Typography.sans(14, relativeTo: .caption))
                            .foregroundStyle(.secondary)
                            .accessibilityLabel("Page \(doc.page + 1) of \(doc.pages)")
                    }
                }
        }
    }
}

/// PDFKit's view, scrolling page after page, reporting the page in view.
struct PDFKitView: UIViewRepresentable {
    @ObservedObject var doc: DocumentSession

    func makeUIView(context: Context) -> PDFView {
        let view = DocumentPDFView()
        view.onCapture = { [weak doc] text, page in doc?.captured(text, page: page) }
        view.autoScales = true
        view.displayMode = .singlePageContinuous
        view.displayDirection = .vertical
        view.backgroundColor = .systemBackground
        context.coordinator.view = view
        // The provider must be in place before the document: PDFKit asks
        // for a page's overlay as it lays the page out the first time.
        view.pageOverlayViewProvider = context.coordinator
        view.document = PDFDocument(url: doc.fileURL)
        if let page = view.document?.page(at: doc.page) {
            // The document lays out after it is in a window; going to the
            // page before that lands on the first page.
            DispatchQueue.main.async { view.go(to: page) }
        }
        NotificationCenter.default.addObserver(context.coordinator, selector: #selector(Coordinator.pageChanged),
                                               name: .PDFViewPageChanged, object: view)
        NotificationCenter.default.addObserver(context.coordinator, selector: #selector(Coordinator.selectionChanged),
                                               name: .PDFViewSelectionChanged, object: view)
        view.isInMarkupMode = doc.tool != .select
        context.coordinator.installSelector(on: view)
        return view
    }

    func updateUIView(_ view: PDFView, context: Context) {
        // Markup mode hands touches to the overlays; off, PDFKit keeps them
        // for text selection. Without it the ink layer is never hit-tested
        // and a Pencil stroke scrolls the page.
        let markup = doc.tool != .select
        if view.isInMarkupMode != markup { view.isInMarkupMode = markup }
        context.coordinator.selector.isEnabled = !markup
        if let text = doc.captureRequested {
            // The harness's way of choosing the menu item: the selection
            // when there is one, else the text it gave.
            let selected = view.currentSelection?.string ?? text
            let page = view.currentSelection?.pages.first.flatMap { view.document?.index(for: $0) } ?? doc.page
            DispatchQueue.main.async {
                doc.captureRequested = nil
                doc.captured(selected, page: page)
            }
        }
        for (index, overlay) in context.coordinator.overlays {
            overlay.apply(tool: doc.tool)
            // Ink that changed in the model (the other device's, or a merge)
            // shows on the page it belongs to.
            if let latest = doc.ink[index], latest != overlay.canonical { overlay.show(latest) }
        }
        // Selecting publishes the selection, which brings this update round
        // again while the request is still set: each request runs once.
        if let req = doc.selectRequested, req.id != context.coordinator.handledSelect,
           let page = view.document?.page(at: req.page) {
            context.coordinator.handledSelect = req.id
            context.coordinator.select(from: (page, req.from), to: (page, req.to), at: view.convert(req.to, from: page), ended: true)
            DispatchQueue.main.async { doc.selectRequested = nil }
        }
        if let wanted = doc.requestedPage {
            if let page = view.document?.page(at: wanted), view.currentPage != page {
                view.go(to: page)
            }
            DispatchQueue.main.async { doc.requestedPage = nil }
        }
    }

    func makeCoordinator() -> Coordinator { Coordinator(doc: doc) }

    @MainActor final class Coordinator: NSObject, @preconcurrency PDFPageOverlayViewProvider,
                                        @preconcurrency UIGestureRecognizerDelegate, @preconcurrency UIEditMenuInteractionDelegate {
        let doc: DocumentSession
        weak var view: PDFView?
        var overlays: [Int: InkOverlay] = [:]
        /// With the select tool up, a Pencil drag selects the text under
        /// it, as the highlighter's drag does on a chapter page; a finger
        /// keeps PDFKit's own long press and handles.
        let selector = UIPanGestureRecognizer()
        private var selectionStart: (page: PDFPage, point: CGPoint)?
        private var menu: UIEditMenuInteraction?
        var handledSelect: UUID?
        init(doc: DocumentSession) { self.doc = doc }

        func installSelector(on view: PDFView) {
            selector.allowedTouchTypes = [NSNumber(value: UITouch.TouchType.pencil.rawValue)]
            selector.maximumNumberOfTouches = 1
            selector.delegate = self
            selector.addTarget(self, action: #selector(selectPan(_:)))
            selector.isEnabled = false
            view.addGestureRecognizer(selector)
            // The page scrolls only once the selector has declined the
            // touch, which for a finger is at once.
            if let scroll = view.subviews.first(where: { $0 is UIScrollView }) as? UIScrollView {
                scroll.panGestureRecognizer.require(toFail: selector)
            }
            let menu = UIEditMenuInteraction(delegate: self)
            view.addInteraction(menu)
            self.menu = menu
        }

        @objc private func selectPan(_ g: UIPanGestureRecognizer) {
            guard let view, let document = view.document else { return }
            let location = g.location(in: view)
            guard let page = view.page(for: location, nearest: true) else { return }
            let point = view.convert(location, to: page)
            switch g.state {
            case .began:
                selectionStart = (page, point)
                view.clearSelection()
            case .changed, .ended:
                guard let start = selectionStart else { return }
                select(from: start, to: (page, point), at: location, ended: g.state == .ended)
            default:
                break
            }
        }

        /// Select the text between two page points, show it, and when the
        /// drag has ended offer the menu at the view point it ended on.
        func select(from start: (page: PDFPage, point: CGPoint), to end: (page: PDFPage, point: CGPoint),
                    at location: CGPoint, ended: Bool) {
            guard let view, let document = view.document else { return }
            let selection = document.selection(from: start.page, at: start.point, to: end.page, at: end.point)
            view.setCurrentSelection(selection, animate: false)
            // PDFKit draws nothing for a selection set in code, so the
            // page's overlay paints it, and moves with the page.
            ownSelection = selection
            paint(selection)
            doc.selection = selection?.string ?? ""
            if ended, let selection, !(selection.string ?? "").isEmpty {
                menu?.presentEditMenu(with: UIEditMenuConfiguration(identifier: nil, sourcePoint: location))
            }
        }

        func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer,
                               shouldRecognizeSimultaneouslyWith other: UIGestureRecognizer) -> Bool { false }

        func editMenuInteraction(_ interaction: UIEditMenuInteraction, menuFor configuration: UIEditMenuConfiguration,
                                 suggestedActions: [UIMenuElement]) -> UIMenu? {
            guard let view = view as? DocumentPDFView, let selection = view.currentSelection,
                  let text = selection.string, !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return nil }
            let page = selection.pages.first.flatMap { view.document?.index(for: $0) } ?? 0
            let copy = UIAction(title: "Copy", image: UIImage(systemName: "doc.on.doc")) { _ in UIPasteboard.general.string = text }
            let send = UIAction(title: "Send to Chiron", image: UIImage(systemName: "text.badge.plus")) { [weak view] _ in
                view?.onCapture?(text, page)
                view?.clearSelection()
            }
            return UIMenu(children: [send, copy])
        }

        // One canvas per page, made when the page comes into view and kept,
        // so scrolling back does not rebuild the strokes.
        func pdfView(_ view: PDFView, overlayViewFor page: PDFPage) -> UIView? {
            guard let index = view.document?.index(for: page) else { return nil }
            if let overlay = overlays[index] { return overlay }
            let overlay = InkOverlay(pageSize: page.bounds(for: .mediaBox).size, drawing: doc.ink[index] ?? PKDrawing())
            overlay.apply(tool: doc.tool)
            overlay.onChange = { [weak self] drawing in
                Task { @MainActor in self?.doc.drew(on: index, drawing) }
            }
            overlays[index] = overlay
            doc.overlaidPages.insert(index)
            return overlay
        }

        func pdfView(_ view: PDFView, willDisplayOverlayView overlayView: UIView, for page: PDFPage) {
            // Ink that arrived from the other device after the overlay was made.
            guard let overlay = overlayView as? InkOverlay, let index = view.document?.index(for: page),
                  let latest = doc.ink[index], latest != overlay.canonical else { return }
            overlay.show(latest)
        }

        @objc func pageChanged() {
            guard let view, let page = view.currentPage, let document = view.document else { return }
            let index = document.index(for: page)
            Task { @MainActor in self.doc.turned(to: index, position: 0) }
        }

        @objc func selectionChanged() {
            let text = view?.currentSelection?.string ?? ""
            Task { @MainActor in
                self.doc.selection = text
                // PDFKit's own selection (a finger's long press) or none:
                // the painted one is stale.
                // PDFView keeps its own copy of the selection, so the two
                // are compared by their text, not their identity.
                if self.view?.currentSelection?.string != self.ownSelection?.string {
                    self.ownSelection = nil
                    self.paint(nil)
                }
            }
        }

        private var ownSelection: PDFSelection?

        /// Paint the selection's lines on the overlays of the pages it
        /// touches, and clear every other page's.
        private func paint(_ selection: PDFSelection?) {
            guard let document = view?.document else { return }
            var byPage: [Int: [CGRect]] = [:]
            let lines = selection?.selectionsByLine() ?? []
            for line in lines {
                for page in line.pages {
                    byPage[document.index(for: page), default: []].append(line.bounds(for: page))
                }
            }
            if byPage.isEmpty, let selection {
                // No line breakdown for this selection: the block it spans.
                for page in selection.pages {
                    byPage[document.index(for: page), default: []].append(selection.bounds(for: page))
                }
            }
            #if DEBUG
            doc.highlightProbe = "lines \(lines.count) pages \(selection?.pages.count ?? -1) "
            #endif
            for (index, overlay) in overlays {
                overlay.highlight(byPage[index] ?? [])
            }
            #if DEBUG
            doc.highlightProbe += "\(byPage) on overlays \(overlays.keys.sorted()); " + (overlays.values.map(\.highlightProbe).joined(separator: " | "))
            #endif
        }
    }
}

/// A PencilKit canvas laid over one page. Strokes are kept in the page's
/// own points, so the same ink fits the page at any zoom and on either
/// device; the canvas shows them scaled to the size PDFKit gives it.
final class InkOverlay: UIView, PKCanvasViewDelegate {
    let canvas = PKCanvasView()
    let pageSize: CGSize
    /// The selected lines, painted under the ink; in page points like
    /// the ink, so a zoom moves them with the page.
    private let selected = CAShapeLayer()
    private var selectedRects: [CGRect] = []
    private(set) var canonical: PKDrawing
    var onChange: ((PKDrawing) -> Void)?
    private var shownAt: CGFloat = 0
    private var applying = false

    init(pageSize: CGSize, drawing: PKDrawing) {
        self.pageSize = pageSize
        self.canonical = drawing
        super.init(frame: .zero)
        backgroundColor = .clear
        isOpaque = false
        canvas.backgroundColor = .clear
        canvas.isOpaque = false
        // The canvas never scrolls: with its pan gesture off, a finger on
        // the page reaches PDFKit's scroll view and scrolls the document.
        canvas.isScrollEnabled = false
        #if targetEnvironment(simulator)
        canvas.drawingPolicy = .anyInput  // no Pencil in the Simulator
        #else
        canvas.drawingPolicy = .pencilOnly
        #endif
        canvas.delegate = self
        addSubview(canvas)
        selected.fillColor = UIColor.systemYellow.withAlphaComponent(0.45).cgColor
        layer.addSublayer(selected)  // over the ink: the wash shows either way
    }

    #if DEBUG
    /// What the overlay was last asked to paint, for the harness.
    var highlightProbe: String { "\(selectedRects.count) rects at scale \(scale), first \(selectedRects.first.map { "\($0)" } ?? "-")" }
    #endif

    /// Rectangles in the page's own coordinates (origin bottom left).
    func highlight(_ rects: [CGRect]) {
        selectedRects = rects
        let path = CGMutablePath()
        for r in rects {
            path.addRect(CGRect(x: r.minX * scale, y: (pageSize.height - r.maxY) * scale,
                                width: r.width * scale, height: r.height * scale))
        }
        selected.path = rects.isEmpty ? nil : path
    }

    required init?(coder: NSCoder) { nil }

    private var scale: CGFloat { pageSize.width > 0 ? bounds.width / pageSize.width : 1 }

    override func layoutSubviews() {
        super.layoutSubviews()
        canvas.frame = bounds
        selected.frame = bounds
        if shownAt != scale {
            show(canonical)
            highlight(selectedRects)
        }
    }

    func show(_ drawing: PKDrawing) {
        canonical = drawing
        shownAt = scale
        applying = true
        canvas.drawing = drawing.transformed(using: CGAffineTransform(scaleX: scale, y: scale))
        applying = false
    }

    func apply(tool: DocumentSession.Tool) {
        switch tool {
        case .select: break  // the page is PDFKit's; the canvas is not hit-tested
        case .pen: canvas.tool = PKInkingTool(.pen, color: .label, width: 2.5)
        case .eraser: canvas.tool = PKEraserTool(.bitmap, width: 24)  // rubs out what it covers, not whole strokes
        }
    }

    func canvasViewDrawingDidChange(_ canvasView: PKCanvasView) {
        guard !applying, scale > 0 else { return }
        canonical = canvasView.drawing.transformed(using: CGAffineTransform(scaleX: 1 / scale, y: 1 / scale))
        onChange?(canonical)
    }
}

/// PDFKit's view with one more item in the selection menu: the selected
/// passage goes to Chiron, the way a page selection in a book does.
final class DocumentPDFView: PDFView {
    var onCapture: ((String, Int) -> Void)?

    override func buildMenu(with builder: any UIMenuBuilder) {
        super.buildMenu(with: builder)
        guard let selection = currentSelection, let text = selection.string,
              !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        let page = selection.pages.first.flatMap { document?.index(for: $0) } ?? 0
        let send = UIAction(title: "Send to Chiron", image: UIImage(systemName: "text.badge.plus")) { [weak self] _ in
            self?.onCapture?(text, page)
            self?.clearSelection()
        }
        builder.insertChild(UIMenu(options: .displayInline, children: [send]), atStartOfMenu: .standardEdit)
    }
}
