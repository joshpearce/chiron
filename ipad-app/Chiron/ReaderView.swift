import SwiftUI
import WebKit

struct ReaderContainer: View {
    @EnvironmentObject var model: AppModel
    /// Passed in rather than read from the model. Leaving the reader to unwrap
    /// `model.chapter` itself crashed on the way out: backToLibrary() nils the
    /// chapter and switches screen, and SwiftUI re-evaluated this body against
    /// the nil chapter before the screen change took effect.
    let chapter: ChapterPayload

    var body: some View {
        VStack(spacing: 0) {
            ReaderView(chapter: chapter)
                .ignoresSafeArea(edges: .bottom)
            Divider()
            HStack {
                if let state = model.bookState, !state.debt.isEmpty {
                    Button {
                        Task { await model.catchMeUp() }
                    } label: {
                        Label("Catch me up", systemImage: "arrow.uturn.backward.circle")
                    }
                    .buttonStyle(.bordered)
                }
                Spacer()
                Button {
                    model.beginCheck()
                } label: {
                    Label("Take the check", systemImage: "checkmark.seal")
                        .padding(.horizontal, 8)
                }
                .buttonStyle(.borderedProminent)
                Button("Skip") {
                    Task { await model.skipCheck() }
                }
                .buttonStyle(.bordered)
                .tint(.orange)
            }
            .padding(12)
            .background(.bar)
        }
    }
}

struct ReaderView: UIViewRepresentable {
    let chapter: ChapterPayload
    @EnvironmentObject var model: AppModel

    func makeCoordinator() -> Coordinator { Coordinator(model: model) }

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
        }
    }

    private func load(into web: WKWebView, context: Context) {
        context.coordinator.loadedUnit = chapter.unit
        context.coordinator.pendingChapter = chapter
        guard let template = Bundle.main.url(forResource: "chapter", withExtension: "html") else { return }
        web.loadFileURL(template, allowingReadAccessTo: template.deletingLastPathComponent())
    }

    final class Coordinator: NSObject, WKScriptMessageHandler {
        let model: AppModel
        weak var web: WKWebView?
        var loadedUnit: String?
        var pendingChapter: ChapterPayload?

        init(model: AppModel) { self.model = model }

        func userContentController(_ ucc: WKUserContentController,
                                   didReceive message: WKScriptMessage) {
            guard let body = message.body as? [String: Any],
                  let type = body["type"] as? String else { return }
            switch type {
            case "ready":
                if let ch = pendingChapter,
                   let data = try? JSONEncoder().encode(ch),
                   let json = String(data: data, encoding: .utf8) {
                    web?.evaluateJavaScript("initChapter(\(json))")
                }
            case "beat":
                let r = BeatResponse(
                    beatId: body["beatId"] as? String ?? "",
                    response: body["response"] as? String ?? "",
                    selfVerdict: body["selfVerdict"] as? String,
                    mechanicalVerdict: body["mechanicalVerdict"] as? String)
                Task { @MainActor in self.model.recordBeat(r) }
            default:
                break
            }
        }
    }
}
