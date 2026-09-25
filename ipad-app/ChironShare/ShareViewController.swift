import UIKit
import UniformTypeIdentifiers

/// The share sheet's "Chiron": take what was shared, put it in the shared
/// inbox, and open the app on it. Extensions are short-lived and sandboxed,
/// so this does nothing else; the question is asked in the app.
final class ShareViewController: UIViewController {
    private let label = UILabel()

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground
        label.text = "Sending to Chiron…"
        label.font = .preferredFont(forTextStyle: .body)
        label.textAlignment = .center
        label.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(label)
        NSLayoutConstraint.activate([
            label.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            label.centerYAnchor.constraint(equalTo: view.centerYAnchor),
        ])
        Task { await collect() }
    }

    private func collect() async {
        var capture = Capture(sourceApp: sourceApp)
        let items = (extensionContext?.inputItems as? [NSExtensionItem]) ?? []
        for item in items {
            for provider in item.attachments ?? [] {
                if provider.hasItemConformingToTypeIdentifier(UTType.plainText.identifier),
                   let s = try? await provider.loadItem(forTypeIdentifier: UTType.plainText.identifier) as? String {
                    capture.text += (capture.text.isEmpty ? "" : "\n\n") + s
                } else if provider.hasItemConformingToTypeIdentifier(UTType.url.identifier),
                          let url = try? await provider.loadItem(forTypeIdentifier: UTType.url.identifier) as? URL {
                    if url.isFileURL {
                        if url.pathExtension.lowercased() == "pdf", let data = try? Data(contentsOf: url) {
                            keepPDF(data, in: &capture)
                        } else if let text = fileText(url) {
                            capture.text += (capture.text.isEmpty ? "" : "\n\n") + text
                        }
                    } else {
                        capture.sourceURL = url.absoluteString
                    }
                } else if provider.hasItemConformingToTypeIdentifier(UTType.pdf.identifier),
                          let loaded = try? await provider.loadItem(forTypeIdentifier: UTType.pdf.identifier),
                          let data = pdfData(loaded) {
                    keepPDF(data, in: &capture)
                } else if provider.hasItemConformingToTypeIdentifier(UTType.image.identifier),
                          let loaded = try? await provider.loadItem(forTypeIdentifier: UTType.image.identifier),
                          let png = pngData(loaded) {
                    capture.imagePNG = png
                }
            }
            if capture.text.isEmpty, let text = item.attributedContentText?.string, !text.isEmpty {
                capture.text = text
            }
        }
        guard !capture.isEmpty, (try? CaptureInbox.write(capture)) != nil else {
            label.text = "Nothing Chiron can read was shared."
            try? await Task.sleep(nanoseconds: 1_200_000_000)
            extensionContext?.cancelRequest(withError: NSError(domain: Bundle.main.bundleIdentifier ?? "chiron.share", code: 1))
            return
        }
        openApp(CaptureInbox.url(for: capture.id))
        extensionContext?.completeRequest(returningItems: nil)
    }

    private var sourceApp: String? {
        // The host app is not named to an extension; the item's title is
        // the closest thing (Safari sets the page title).
        (extensionContext?.inputItems.first as? NSExtensionItem)?.attributedTitle?.string
    }

    private func fileText(_ url: URL) -> String? {
        try? String(contentsOf: url, encoding: .utf8)
    }

    private func pdfData(_ any: Any) -> Data? {
        switch any {
        case let d as Data: return d
        case let u as URL: return try? Data(contentsOf: u)
        default: return nil
        }
    }

    /// The PDF itself goes into the inbox; the app puts it on the shelf.
    private func keepPDF(_ data: Data, in capture: inout Capture) {
        let name = "\(capture.id).pdf"
        let url = CaptureInbox.directory.appendingPathComponent(name)
        if (try? data.write(to: url, options: .atomic)) != nil {
            capture.pdfFile = name
        }
    }

    private func pngData(_ any: Any) -> Data? {
        switch any {
        case let image as UIImage: return image.pngData()
        case let data as Data: return UIImage(data: data)?.pngData()
        case let url as URL: return (try? Data(contentsOf: url)).flatMap { UIImage(data: $0)?.pngData() }
        default: return nil
        }
    }

    /// Extensions may not call UIApplication.open; the responder chain
    /// reaches an object that can.
    private func openApp(_ url: URL) {
        var responder: UIResponder? = self
        while let r = responder {
            if let app = r as? UIApplication {
                app.open(url, options: [:], completionHandler: nil)
                return
            }
            if r.responds(to: Selector(("openURL:"))) {
                r.perform(Selector(("openURL:")), with: url)
                return
            }
            responder = r.next
        }
    }
}
