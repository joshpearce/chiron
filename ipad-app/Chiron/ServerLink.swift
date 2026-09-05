import CoreImage.CIFilterBuiltins
import UIKit

/// One server's setup as a URL, chiron://server?name=&url=&key=, so that a
/// second device gets it from a QR code rather than a keyboard. The Camera
/// app opens it in Chiron; the scanner in "Add a server" reads the same.
struct ServerLink: Equatable {
    var name: String
    var url: String
    var key: String?

    static let host = "server"

    init(name: String, url: String, key: String?) {
        self.name = name
        self.url = url
        self.key = key.flatMap { $0.isEmpty ? nil : $0 }
    }

    /// The link a URL carries, if it is one: chiron scheme, "server" host,
    /// and an address.
    init?(_ url: URL) {
        guard url.scheme == CaptureInbox.scheme, url.host == Self.host,
              let items = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems else { return nil }
        var fields: [String: String] = [:]
        for item in items { fields[item.name] = item.value ?? "" }
        guard let address = fields["url"], !address.isEmpty else { return nil }
        self.init(name: fields["name"] ?? "", url: address, key: fields["key"])
    }

    var asURL: URL {
        var c = URLComponents()
        c.scheme = CaptureInbox.scheme
        c.host = Self.host
        var items = [URLQueryItem(name: "url", value: url)]
        if !name.isEmpty { items.insert(URLQueryItem(name: "name", value: name), at: 0) }
        if let key { items.append(URLQueryItem(name: "key", value: key)) }
        c.queryItems = items
        return c.url!
    }
}

/// A QR code of a string, drawn crisp at any size.
enum QRCode {
    static func image(of text: String, side: CGFloat = 512) -> UIImage? {
        let filter = CIFilter.qrCodeGenerator()
        filter.message = Data(text.utf8)
        filter.correctionLevel = "M"
        guard let output = filter.outputImage else { return nil }
        let scale = side / output.extent.width
        let scaled = output.transformed(by: CGAffineTransform(scaleX: scale, y: scale))
        guard let cg = CIContext().createCGImage(scaled, from: scaled.extent) else { return nil }
        return UIImage(cgImage: cg)
    }
}
