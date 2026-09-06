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
    var onTurn: ((Int, Double) -> Void)?
    /// Ink per page, in the page's own points, with the version the server
    /// last gave for each and the pages drawn on since.
    @Published private(set) var ink: [Int: PKDrawing] = [:]
    var inkVersions: [Int: Int] = [:]
    private var dirtyInk: Set<Int> = []
    var onInk: ((Int) -> Void)?
    enum Tool: String { case pen, eraser }
    @Published var tool: Tool = .pen
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
        return view
    }

    func updateUIView(_ view: PDFView, context: Context) {
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
        if let wanted = doc.requestedPage {
            if let page = view.document?.page(at: wanted), view.currentPage != page {
                view.go(to: page)
            }
            DispatchQueue.main.async { doc.requestedPage = nil }
        }
    }

    func makeCoordinator() -> Coordinator { Coordinator(doc: doc) }

    @MainActor final class Coordinator: NSObject, @preconcurrency PDFPageOverlayViewProvider {
        let doc: DocumentSession
        weak var view: PDFView?
        var overlays: [Int: InkOverlay] = [:]
        init(doc: DocumentSession) { self.doc = doc }

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
    }
}

/// A PencilKit canvas laid over one page. Strokes are kept in the page's
/// own points, so the same ink fits the page at any zoom and on either
/// device; the canvas shows them scaled to the size PDFKit gives it.
final class InkOverlay: UIView, PKCanvasViewDelegate {
    let canvas = PKCanvasView()
    let pageSize: CGSize
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
        canvas.drawingPolicy = .pencilOnly
        canvas.delegate = self
        addSubview(canvas)
    }

    required init?(coder: NSCoder) { nil }

    private var scale: CGFloat { pageSize.width > 0 ? bounds.width / pageSize.width : 1 }

    override func layoutSubviews() {
        super.layoutSubviews()
        canvas.frame = bounds
        if shownAt != scale { show(canonical) }
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
        case .pen: canvas.tool = PKInkingTool(.pen, color: .label, width: 2.5)
        case .eraser: canvas.tool = PKEraserTool(.vector)
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
