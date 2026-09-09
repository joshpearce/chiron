import Foundation
import WebKit

/// The pictures an imported book carries: its charts, its exhibits, its
/// photographs.
///
/// A chapter refers to them by the name they have beside the book, and
/// the page asks for them under a scheme of its own. They are fetched
/// once and kept, so a book read on a plane still has its exhibits, and
/// so a chapter turned back to does not fetch them again.
@MainActor
final class BookAssets {
    static let scheme = "chiron-asset"

    private let service: ChironService
    private let dir: URL

    init(service: ChironService, storage: URL) {
        self.service = service
        dir = storage.appendingPathComponent("assets", isDirectory: true)
    }

    /// What the page refers to a book's pictures by.
    static func base(subject: String) -> String { "\(scheme)://book/\(subject)/" }

    private func file(subject: String, name: String) -> URL {
        dir.appendingPathComponent(subject, isDirectory: true).appendingPathComponent(name)
    }

    /// The picture, from this device if it has it and from the server if
    /// it does not.
    func data(subject: String, name: String) async throws -> Data {
        let kept = file(subject: subject, name: name)
        if let data = try? Data(contentsOf: kept) { return data }
        let data = try await service.bookAsset(subject: subject, name: name)
        try? FileManager.default.createDirectory(at: kept.deletingLastPathComponent(),
                                                 withIntermediateDirectories: true)
        try? data.write(to: kept)
        return data
    }

    /// Forget a book's pictures: it is off the shelf.
    func forget(subject: String) {
        try? FileManager.default.removeItem(at: dir.appendingPathComponent(subject, isDirectory: true))
    }
}

/// Serves `chiron-asset://book/<subject>/<name>` to the page from the
/// book's own pictures. A scheme of its own rather than the server's URL:
/// the page cannot carry the server's key, and a picture already on this
/// device should not need the server at all.
final class BookAssetScheme: NSObject, WKURLSchemeHandler {
    private let assets: BookAssets
    private var running: [ObjectIdentifier: Task<Void, Never>] = [:]

    init(assets: BookAssets) { self.assets = assets }

    func webView(_ webView: WKWebView, start task: any WKURLSchemeTask) {
        let url = task.request.url
        let parts = (url?.path ?? "").split(separator: "/").map(String.init)
        guard let url, let subject = url.host == "book" ? parts.first : nil, parts.count >= 2 else {
            task.didFailWithError(URLError(.badURL))
            return
        }
        let name = parts[1].removingPercentEncoding ?? parts[1]
        let key = ObjectIdentifier(task)
        running[key] = Task { @MainActor in
            defer { self.running[key] = nil }
            do {
                let data = try await assets.data(subject: subject, name: name)
                guard !Task.isCancelled else { return }
                let response = URLResponse(url: url, mimeType: Self.mime(for: name),
                                           expectedContentLength: data.count, textEncodingName: nil)
                task.didReceive(response)
                task.didReceive(data)
                task.didFinish()
            } catch {
                guard !Task.isCancelled else { return }
                task.didFailWithError(error)
            }
        }
    }

    func webView(_ webView: WKWebView, stop task: any WKURLSchemeTask) {
        let key = ObjectIdentifier(task)
        running[key]?.cancel()
        running[key] = nil
    }

    static func mime(for name: String) -> String {
        switch (name as NSString).pathExtension.lowercased() {
        case "png": return "image/png"
        case "gif": return "image/gif"
        case "svg": return "image/svg+xml"
        default: return "image/jpeg"
        }
    }
}
