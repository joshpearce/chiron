import Foundation

/// Something captured elsewhere on the iPad, on its way to becoming a
/// primer: text, or an image, with where it came from.
struct Capture: Codable, Identifiable, Equatable {
    var id: String = UUID().uuidString
    var text: String = ""
    var imagePNG: Data?
    var sourceURL: String?
    var sourceApp: String?
    /// A PDF shared in, written beside the capture in the inbox: it goes
    /// on the shelf as a document rather than becoming text.
    var pdfFile: String?

    var isEmpty: Bool { text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && imagePNG == nil && pdfFile == nil }
}

/// The hand-off between the share extension and the app. Extensions run in
/// their own process with no way into the app's state, so the extension
/// writes the capture into the shared App Group container and opens
/// chiron://capture/<id>; the app takes it from there. Without the group
/// entitlement (a build that lacks it) the app's own support directory
/// stands in, which still serves the in-app and intent paths.
enum CaptureInbox {
    static let group = "group.dev.mjbraun.chiron"
    static let scheme = "chiron"

    static var directory: URL {
        let base = FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: group)
            ?? FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        let dir = base.appendingPathComponent("ChironInbox", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir
    }

    @discardableResult
    static func write(_ c: Capture) throws -> URL {
        let url = directory.appendingPathComponent("\(c.id).json")
        try JSONEncoder().encode(c).write(to: url, options: .atomic)
        return url
    }

    /// Read a capture and remove it: a capture is delivered once.
    static func take(_ id: String) -> Capture? {
        let url = directory.appendingPathComponent("\(id).json")
        guard let data = try? Data(contentsOf: url), let c = try? JSONDecoder().decode(Capture.self, from: data) else { return nil }
        try? FileManager.default.removeItem(at: url)
        return c
    }

    /// The URL that brings the app to a capture.
    static func url(for id: String) -> URL {
        URL(string: "\(scheme)://capture/\(id)")!
    }

    /// The capture id a chiron:// URL names, if it is one.
    static func captureID(in url: URL) -> String? {
        guard url.scheme == scheme, url.host == "capture" else { return nil }
        let id = url.lastPathComponent
        return id.isEmpty || id == "/" ? nil : id
    }
}
