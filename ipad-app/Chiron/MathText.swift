import SwiftUI
import WebKit

/// Text that may contain LaTeX, rendered with the bundled KaTeX.
///
/// SwiftUI's `Text` renders markdown but not math, so check prompts like
/// "compare $q \cdot k_1$ with $q \cdot k_2$" would reach the learner as raw
/// TeX - and the checks are the most math-heavy moment in the app. This keeps
/// chapter typography and check typography identical, because both go through
/// KaTeX.
///
/// Falls back to plain SwiftUI text when there is no `$`, so the common case
/// costs nothing.
struct MathText: View {
    let text: String
    var size: CGFloat = 19

    @State private var height: CGFloat = 24

    var body: some View {
        if text.contains("$") {
            MathWebView(text: text, size: size, height: $height)
                .frame(height: height)
        } else {
            Text(.init(text))
                .font(.system(size: size, design: .serif))
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

private struct MathWebView: UIViewRepresentable {
    let text: String
    let size: CGFloat
    @Binding var height: CGFloat

    func makeCoordinator() -> Coordinator { Coordinator(height: $height) }

    func makeUIView(context: Context) -> WKWebView {
        let web = WKWebView()
        web.isOpaque = false
        web.backgroundColor = .clear
        web.scrollView.isScrollEnabled = false
        web.scrollView.backgroundColor = .clear
        web.navigationDelegate = context.coordinator
        context.coordinator.load(web, text: text, size: size)
        return web
    }

    func updateUIView(_ web: WKWebView, context: Context) {
        if context.coordinator.loadedText != text {
            context.coordinator.load(web, text: text, size: size)
        }
    }

    final class Coordinator: NSObject, WKNavigationDelegate {
        @Binding var height: CGFloat
        var loadedText: String?

        init(height: Binding<CGFloat>) { _height = height }

        func load(_ web: WKWebView, text: String, size: CGFloat) {
            guard let res = Bundle.main.resourceURL else { return }
            loadedText = text
            let escaped = text
                .replacingOccurrences(of: "&", with: "&amp;")
                .replacingOccurrences(of: "<", with: "&lt;")
                .replacingOccurrences(of: ">", with: "&gt;")
            let html = """
            <!DOCTYPE html><html><head><meta charset="utf-8">
            <meta name="viewport" content="width=device-width, initial-scale=1">
            <link rel="stylesheet" href="katex/katex.min.css">
            <style>
              :root { color-scheme: light dark; }
              html, body { margin:0; padding:0; background:transparent; }
              body {
                font: \(size)px/1.5 -apple-system-ui-serif, ui-serif, Georgia, serif;
                color: #1a1a1a;
                -webkit-text-size-adjust: 100%;
              }
              @media (prefers-color-scheme: dark) { body { color: #d8d3c8; } }
              .katex-display { overflow-x:auto; overflow-y:hidden; margin:0.4em 0; }
              code { font-family: ui-monospace, Menlo, monospace; font-size: 0.85em; }
            </style>
            <script src="katex/katex.min.js"></script>
            <script src="katex/contrib/auto-render.min.js"></script>
            </head><body><div id="c">\(escaped)</div>
            <script>
              renderMathInElement(document.getElementById('c'), {
                delimiters: [{left:'$$',right:'$$',display:true},
                             {left:'$',right:'$',display:false}],
                throwOnError: false
              });
            </script></body></html>
            """
            web.loadHTMLString(html, baseURL: res)
        }

        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            // KaTeX lays out after load, so the first measurement can be short
            // and would clip tall display equations. Re-measure a few times.
            var tries = 0
            func measure() {
                webView.evaluateJavaScript("document.body.scrollHeight") { result, _ in
                    if let h = result as? CGFloat, h > 0, abs(h - self.height) > 1 {
                        self.height = h
                    }
                    tries += 1
                    if tries < 5 {
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2, execute: measure)
                    }
                }
            }
            measure()
        }
    }
}
