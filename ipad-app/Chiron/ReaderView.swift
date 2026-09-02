import SwiftUI
import WebKit

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
            if !session.chromeHidden {
                Divider()
                bottomBar
            }
        }
        .animation(.easeInOut(duration: 0.2), value: session.chromeHidden)
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

struct ReaderView: UIViewRepresentable {
    let chapter: ChapterPayload
    @EnvironmentObject var session: BookSession
    @Environment(\.sizeCategory) private var sizeCategory

    func makeCoordinator() -> Coordinator { Coordinator(session: session) }

    /// The system's body text scale, handed to the stylesheet so the page
    /// reflows with Dynamic Type rather than zooming.
    private var scale: CGFloat { UIFontMetrics(forTextStyle: .body).scaledValue(for: 100) / 100 }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "bridge")
        let web = WKWebView(frame: .zero, configuration: config)
        if #available(iOS 16.4, *) {
            web.isInspectable = true
        }
        web.scrollView.contentInsetAdjustmentBehavior = .never
        context.coordinator.web = web
        load(into: web, context: context)
        return web
    }

    func updateUIView(_ web: WKWebView, context: Context) {
        if context.coordinator.loadedUnit != chapter.unit {
            load(into: web, context: context)
        } else if context.coordinator.scale != scale {
            context.coordinator.scale = scale
            web.evaluateJavaScript("setScale(\(scale))")
        }
    }

    private func load(into web: WKWebView, context: Context) {
        context.coordinator.loadedUnit = chapter.unit
        context.coordinator.pendingChapter = chapter
        context.coordinator.scale = scale
        context.coordinator.position = session.position(for: chapter.unit)
        guard let template = Bundle.main.url(forResource: "chapter", withExtension: "html") else { return }
        web.loadFileURL(template, allowingReadAccessTo: template.deletingLastPathComponent())
    }

    final class Coordinator: NSObject, WKScriptMessageHandler {
        let session: BookSession
        weak var web: WKWebView?
        var loadedUnit: String?
        var pendingChapter: ChapterPayload?
        var scale: CGFloat = 1
        var position: Double = 0

        init(session: BookSession) { self.session = session }

        func userContentController(_ ucc: WKUserContentController,
                                   didReceive message: WKScriptMessage) {
            guard let body = message.body as? [String: Any],
                  let type = body["type"] as? String else { return }
            switch type {
            case "ready":
                if let ch = pendingChapter,
                   let data = try? JSONEncoder().encode(ch),
                   let json = String(data: data, encoding: .utf8) {
                    web?.evaluateJavaScript("setScale(\(scale)); initChapter(\(json), \(position))")
                }
            case "scroll":
                if let unit = loadedUnit, let offset = body["offset"] as? Double {
                    Task { @MainActor in self.session.recordPosition(unit: unit, offset: offset) }
                }
            case "tap":
                Task { @MainActor in self.session.toggleChrome() }
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
