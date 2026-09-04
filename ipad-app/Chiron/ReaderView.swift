import SwiftUI
import WebKit
import PencilKit

struct ReaderContainer: View {
    @EnvironmentObject var session: BookSession
    /// Passed in rather than read from the session. Leaving the reader to
    /// unwrap `session.chapter` itself crashed on the way out: closing the
    /// book nils the chapter and switches screen, and SwiftUI re-evaluated
    /// this body against the nil chapter before the screen change took effect.
    let chapter: ChapterPayload

    var body: some View {
        VStack(spacing: 0) {
            ReaderView(chapter: chapter)
                .ignoresSafeArea(edges: .bottom)
                .overlay(alignment: .trailing) {
                    Palette()
                        .padding(.trailing, 10)
                }
                .overlay(alignment: .bottomTrailing) {
                    if session.asking != nil {
                        AskCard()
                            .padding(16)
                            .transition(.move(edge: .bottom).combined(with: .opacity))
                    }
                }
            // A primer has no check: nothing to take, nothing to skip.
            if !session.chromeHidden && !session.isPrimer {
                Divider()
                bottomBar
            }
        }
        .animation(.easeInOut(duration: 0.2), value: session.chromeHidden)
        .animation(.easeInOut(duration: 0.2), value: session.asking != nil)
    }

    private var bottomBar: some View {
        Group {
            HStack {
                if let state = session.bookState, !state.debt.isEmpty {
                    Button {
                        Task { await session.catchMeUp() }
                    } label: {
                        Label("Catch me up", systemImage: "arrow.uturn.backward.circle")
                    }
                    .buttonStyle(.bordered)
                }
                Spacer()
                Text("\(chapter.check.count) questions")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                Button {
                    session.beginCheck()
                } label: {
                    Label("Take the check", systemImage: "checkmark.seal")
                        .padding(.horizontal, 8)
                }
                .buttonStyle(.borderedProminent)
                .keyboardShortcut(.return, modifiers: .command)
                Button("Skip") {
                    Task { await session.skipCheck() }
                }
                .buttonStyle(.bordered)
                .tint(.orange)
            }
            .disabled(session.busy)
            .padding(12)
            .background(.bar)
        }
    }
}

/// The chapter page: a web view for the prose, with the reader's ink riding
/// inside its scroll view so it scrolls with the text, and a gesture layer
/// over it that turns a drag into a run of text for the highlighter and the
/// ask tool. The Pencil Pro's squeeze and double-tap land here too.
struct ReaderView: UIViewRepresentable {
    let chapter: ChapterPayload
    @EnvironmentObject var session: BookSession
    @Environment(\.sizeCategory) private var sizeCategory

    func makeCoordinator() -> Coordinator {
        let c = Coordinator(session: session)
        #if DEBUG
        Coordinator.probe = c
        #endif
        return c
    }

    /// The system's body text scale, handed to the stylesheet so the page
    /// reflows with Dynamic Type rather than zooming.
    private var scale: CGFloat { UIFontMetrics(forTextStyle: .body).scaledValue(for: 100) / 100 }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "bridge")
        let web = WKWebView(frame: .zero, configuration: config)
        web.isInspectable = true
        web.scrollView.contentInsetAdjustmentBehavior = .never
        context.coordinator.attach(to: web)
        load(into: web, context: context)
        return web
    }

    func updateUIView(_ web: WKWebView, context: Context) {
        let c = context.coordinator
        // A new unit, or the same unit rewritten (a primer that grew from a
        // margin note), is a fresh page; marks re-apply on top.
        if c.loadedUnit != chapter.unit || c.loadedHTML != chapter.html.hashValue {
            load(into: web, context: context)
        } else if c.scale != scale {
            c.scale = scale
            web.evaluateJavaScript("setScale(\(scale))")
        }
        c.apply(tool: session.tool, color: session.penColor.uiColor)
        c.apply(marks: session.marks)
        c.apply(ink: session.inkData)
    }

    private func load(into web: WKWebView, context: Context) {
        let c = context.coordinator
        c.loadedUnit = chapter.unit
        c.loadedHTML = chapter.html.hashValue
        c.pendingChapter = chapter
        c.scale = scale
        c.position = session.position(for: chapter.unit)
        c.pageReady = false
        c.appliedMarks = nil
        guard let template = Bundle.main.url(forResource: "chapter", withExtension: "html") else { return }
        web.loadFileURL(template, allowingReadAccessTo: template.deletingLastPathComponent())
    }

    final class Coordinator: NSObject, WKScriptMessageHandler, PKCanvasViewDelegate,
                             UIPencilInteractionDelegate, PageBridge {
        let session: BookSession
        weak var web: WKWebView?
        var loadedUnit: String?
        var loadedHTML: Int?
        var pendingChapter: ChapterPayload?
        var scale: CGFloat = 1
        var position: Double = 0
        var pageReady = false
        var appliedMarks: [Mark]?
        private var appliedTool: BookSession.Tool = .none
        private var appliedInk: Data?
        private var loadingInk = false

        private let canvas = PageInkCanvas()
        private let marker = MarkGestureView()
        private var scrollObservations: [NSKeyValueObservation] = []

        init(session: BookSession) {
            self.session = session
            super.init()
            session.page = self
        }

        func attach(to web: WKWebView) {
            self.web = web

            // The ink is a layer over the web view, not inside its scroll
            // view: WebKit holds touches inside its own scroll view for the
            // page's sake and hands them on late, so a stroke drawn there
            // only showed once the Pencil lifted. The canvas is itself a
            // scroll view; its content mirrors the page's size and offset,
            // so strokes are stored in page coordinates and stay with the
            // text they annotate. With the default drawing policy a paired
            // Pencil draws and fingers keep scrolling; without one, fingers
            // draw (the Simulator).
            #if targetEnvironment(simulator)
            // The Simulator has no Pencil and the default policy draws
            // nothing there; the tests draw with a finger.
            canvas.drawingPolicy = .anyInput
            #else
            canvas.drawingPolicy = .default
            #endif
            canvas.backgroundColor = .clear
            canvas.isOpaque = false
            canvas.isScrollEnabled = false
            canvas.contentInsetAdjustmentBehavior = .never
            canvas.delegate = self
            canvas.isUserInteractionEnabled = false
            canvas.translatesAutoresizingMaskIntoConstraints = false
            web.addSubview(canvas)
            NSLayoutConstraint.activate([
                canvas.leadingAnchor.constraint(equalTo: web.leadingAnchor),
                canvas.trailingAnchor.constraint(equalTo: web.trailingAnchor),
                canvas.topAnchor.constraint(equalTo: web.topAnchor),
                canvas.bottomAnchor.constraint(equalTo: web.bottomAnchor),
            ])
            scrollObservations = [
                web.scrollView.observe(\.contentSize, options: [.initial, .new]) { [weak self] sv, _ in
                    guard let self else { return }
                    self.canvas.contentSize = CGSize(width: max(sv.contentSize.width, sv.bounds.width),
                                                     height: max(sv.contentSize.height, sv.bounds.height))
                    self.canvas.contentOffset = sv.contentOffset
                },
                web.scrollView.observe(\.contentOffset, options: [.initial, .new]) { [weak self] sv, _ in
                    self?.canvas.contentOffset = sv.contentOffset
                },
            ]

            // The text-selection layer sits over the viewport (the page's
            // caret lookup works in viewport coordinates).
            marker.translatesAutoresizingMaskIntoConstraints = false
            marker.isUserInteractionEnabled = false
            marker.onPreview = { [weak self] a, b in self?.preview(from: a, to: b) }
            marker.onSelect = { [weak self] a, b in self?.select(from: a, to: b) }
            web.addSubview(marker)
            NSLayoutConstraint.activate([
                marker.leadingAnchor.constraint(equalTo: web.leadingAnchor),
                marker.trailingAnchor.constraint(equalTo: web.trailingAnchor),
                marker.topAnchor.constraint(equalTo: web.topAnchor),
                marker.bottomAnchor.constraint(equalTo: web.bottomAnchor),
            ])

            web.addInteraction(UIPencilInteraction(delegate: self))
        }

        // MARK: tools

        private var appliedColor: UIColor?

        #if DEBUG
        /// The harness reads what the canvas actually holds, not what the
        /// session thinks it asked for.
        static weak var probe: Coordinator?
        var canvasPen: String {
            guard let ink = canvas.tool as? PKInkingTool else { return String(describing: type(of: canvas.tool)) }
            var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
            ink.color.getRed(&r, green: &g, blue: &b, alpha: &a)
            return String(format: "pen rgb(%.0f,%.0f,%.0f) w%.1f", r * 255, g * 255, b * 255, ink.width)
        }
        var canvasTouches: Int { canvas.touchesSeen }
        var canvasFrame: String { "\(canvas.frame) content \(canvas.contentSize) offset \(canvas.contentOffset) enabled \(canvas.isUserInteractionEnabled)" }
        #endif

        func apply(tool: BookSession.Tool, color: UIColor) {
            guard tool != appliedTool || (tool == .pen && color != appliedColor) else { return }
            appliedTool = tool
            appliedColor = color
            switch tool {
            case .pen:
                canvas.tool = PKInkingTool(.pen, color: color, width: 2.5)
                canvas.isUserInteractionEnabled = true
                marker.isUserInteractionEnabled = false
            case .eraser:
                canvas.tool = PKEraserTool(.vector)
                canvas.isUserInteractionEnabled = true
                marker.isUserInteractionEnabled = false
            case .highlighter, .ask, .note:
                canvas.isUserInteractionEnabled = false
                marker.isUserInteractionEnabled = true
            case .none:
                canvas.isUserInteractionEnabled = false
                marker.isUserInteractionEnabled = false
            }
        }

        // MARK: ink

        func apply(ink: Data?) {
            guard ink != appliedInk else { return }
            appliedInk = ink
            loadingInk = true
            canvas.drawing = ink.flatMap { try? PKDrawing(data: $0) } ?? PKDrawing()
            loadingInk = false
        }

        func canvasViewDrawingDidChange(_ canvasView: PKCanvasView) {
            guard !loadingInk else { return }
            let data = canvasView.drawing.strokes.isEmpty ? nil : canvasView.drawing.dataRepresentation()
            appliedInk = data
            Task { @MainActor in self.session.saveInk(data) }
        }

        // MARK: marks

        func apply(marks: [Mark]) {
            guard pageReady, marks != appliedMarks else { return }
            appliedMarks = marks
            let list = marks.map { ["id": $0.id, "kind": $0.kind.rawValue, "start": $0.start, "end": $0.end] as [String: Any] }
            if let data = try? JSONSerialization.data(withJSONObject: list),
               let json = String(data: data, encoding: .utf8) {
                web?.evaluateJavaScript("applyMarks(\(json))")
            }
        }

        private func preview(from a: CGPoint, to b: CGPoint) {
            web?.evaluateJavaScript("previewRange(\(a.x), \(a.y), \(b.x), \(b.y))")
        }

        private func select(from a: CGPoint, to b: CGPoint) {
            web?.evaluateJavaScript("unmark('preview'); offsetsFromPoints(\(a.x), \(a.y), \(b.x), \(b.y))") { [weak self] result, _ in
                guard let self, let r = result as? [String: Any],
                      let start = r["start"] as? Int, let end = r["end"] as? Int, let text = r["text"] as? String else { return }
                Task { @MainActor in
                    let kind: Mark.Kind
                    switch self.session.tool {
                    case .ask: kind = .question
                    case .note: kind = .note
                    default: kind = .highlight
                    }
                    self.session.addMark(kind: kind, start: start, end: end, text: text)
                }
            }
        }

        func find(_ text: String) async -> (start: Int, end: Int, text: String)? {
            guard let web, pageReady else { return nil }
            let escaped = text.replacingOccurrences(of: "\\", with: "\\\\").replacingOccurrences(of: "'", with: "\\'")
            guard let r = try? await web.evaluateJavaScript("findText('\(escaped)')") as? [String: Any],
                  let start = r["start"] as? Int, let end = r["end"] as? Int, let found = r["text"] as? String else { return nil }
            return (start, end, found)
        }

        // MARK: pencil

        func pencilInteractionDidTap(_ interaction: UIPencilInteraction) {
            Task { @MainActor in self.session.flipEraser() }
        }

        func pencilInteraction(_ interaction: UIPencilInteraction, didReceiveSqueeze squeeze: UIPencilInteraction.Squeeze) {
            guard squeeze.phase == .ended else { return }
            Task { @MainActor in self.session.cycleTool() }
        }

        // MARK: the page's messages

        func userContentController(_ ucc: WKUserContentController,
                                   didReceive message: WKScriptMessage) {
            guard let body = message.body as? [String: Any],
                  let type = body["type"] as? String else { return }
            switch type {
            case "ready":
                if let ch = pendingChapter,
                   let data = try? JSONEncoder().encode(ch),
                   let json = String(data: data, encoding: .utf8) {
                    web?.evaluateJavaScript("setScale(\(scale)); initChapter(\(json), \(position))") { [weak self] _, _ in
                        guard let self else { return }
                        self.pageReady = true
                        Task { @MainActor in self.apply(marks: self.session.marks) }
                    }
                }
            case "scroll":
                if let unit = loadedUnit, let offset = body["offset"] as? Double {
                    Task { @MainActor in self.session.recordPosition(unit: unit, offset: offset) }
                }
            case "tap":
                Task { @MainActor in self.session.toggleChrome() }
            case "mark":
                if let id = body["id"] as? String {
                    Task { @MainActor in self.session.openMark(id) }
                }
            case "beat":
                let r = BeatResponse(
                    beatId: body["beatId"] as? String ?? "",
                    response: body["response"] as? String ?? "",
                    selfVerdict: body["selfVerdict"] as? String,
                    mechanicalVerdict: body["mechanicalVerdict"] as? String)
                Task { @MainActor in self.session.recordBeat(r) }
            default:
                break
            }
        }
    }
}

/// A drag across the page selects a run of text. Reports the drag as it
/// moves (for the provisional highlight) and when it ends.
final class MarkGestureView: UIView {
    var onPreview: ((CGPoint, CGPoint) -> Void)?
    var onSelect: ((CGPoint, CGPoint) -> Void)?
    private var start: CGPoint = .zero

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = .clear
        let pan = UIPanGestureRecognizer(target: self, action: #selector(pan(_:)))
        pan.maximumNumberOfTouches = 1
        addGestureRecognizer(pan)
    }

    required init?(coder: NSCoder) { fatalError() }

    @objc private func pan(_ g: UIPanGestureRecognizer) {
        let p = g.location(in: self)
        switch g.state {
        case .began: start = p
        case .changed: onPreview?(start, p)
        case .ended: onSelect?(start, p)
        default: break
        }
    }
}

/// The ink layer. It sits over the page, so a finger that should scroll
/// the page must fall through it: when the iPad is set to draw only with
/// the Pencil, anything but a Pencil touch is not ours. Elsewhere (no
/// Pencil paired, the Simulator) fingers draw, as PencilKit's default
/// policy intends.
final class PageInkCanvas: PKCanvasView {
    #if DEBUG
    /// Touches that reached the canvas, for the harness.
    private(set) var touchesSeen = 0
    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent?) {
        touchesSeen += touches.count
        super.touchesBegan(touches, with: event)
    }
    #endif

    override func hitTest(_ point: CGPoint, with event: UIEvent?) -> UIView? {
        if UIPencilInteraction.prefersPencilOnlyDrawing,
           let touch = event?.allTouches?.first, touch.type != .pencil {
            return nil
        }
        return super.hitTest(point, with: event)
    }
}
