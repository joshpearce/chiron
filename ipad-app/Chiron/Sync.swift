import Foundation
import Network

/// Transport-agnostic exchange with the Mac book-server.
///
/// Wi-Fi path: plain HTTP client to the configured base URL.
/// USB path: the app runs a tiny HTTP listener; the Mac-side usb-bridge script
/// (through `iproxy`) polls GET /pending for an outboxed exchange request and
/// POSTs the server's response back to /deliver.
@MainActor
final class Sync: ObservableObject {
    @Published var baseURL: String {
        didSet { UserDefaults.standard.set(baseURL, forKey: "baseURL") }
    }
    /// Saved servers. The selected one drives baseURL and supplies its own key.
    let servers = ServerStore()
    var serverID: UUID? { servers.selected?.id }
    @Published var connected = false
    @Published var transport = "offline"   // wifi | usb | offline
    @Published var busy = false

    /// How long to wait for the Mac-side USB bridge before giving up. Chapter
    /// generation on a local model can take a while, so this is generous - but
    /// finite, so a bridge that never runs is reported instead of hanging.
    let usbTimeout: TimeInterval = 150

    private var outbox: Data?              // pending exchange request (USB path)
    private var delivered: ((Data) -> Void)?
    private var listener: NWListener?

    init() {
        baseURL = UserDefaults.standard.string(forKey: "baseURL") ?? "http://192.168.2.1:8080"
        // Anything configured before servers were a list becomes the first
        // saved entry, so upgrading does not read as losing the setup.
        servers.migrateIfNeeded(currentURL: baseURL)
        if let selected = servers.selected { baseURL = selected.url }
        // Development override: point the app at a server for this launch only,
        // without touching saved servers. Simulator runs use
        //   SIMCTL_CHILD_CHIRON_SERVER=http://localhost:8080 xcrun simctl launch ...
        if let override = ProcessInfo.processInfo.environment["CHIRON_SERVER"],
           !override.isEmpty {
            baseURL = override
        }
        startUSBListener()
        Task { await probe() }
    }

    func probe() async {
        if let url = URL(string: "\(baseURL)/health") {
            var req = URLRequest(url: url, timeoutInterval: 3)
            req.httpMethod = "GET"
            Credentials.authorize(&req, serverID: serverID)
            if let (_, resp) = try? await URLSession.shared.data(for: req),
               (resp as? HTTPURLResponse)?.statusCode == 200 {
                connected = true
                transport = "wifi"
                return
            }
        }
        connected = false
        transport = outboxWaiting ? "usb" : "offline"
    }

    var outboxWaiting: Bool { outbox != nil }

    /// Perform an exchange. Tries Wi-Fi; if unreachable, parks the request in
    /// the outbox for the USB bridge and suspends until delivery (or timeout).
    func exchange(_ request: ExchangeRequest) async throws -> ExchangeResponse {
        busy = true
        defer { busy = false }
        let body = try JSONEncoder().encode(request)

        if let url = URL(string: "\(baseURL)/exchange") {
            var req = URLRequest(url: url, timeoutInterval: 600)
            req.httpMethod = "POST"
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            Credentials.authorize(&req, serverID: serverID)
            req.httpBody = body
            if let (data, resp) = try? await URLSession.shared.data(for: req),
               (resp as? HTTPURLResponse)?.statusCode == 200 {
                connected = true
                transport = "wifi"
                return try JSONDecoder().decode(ExchangeResponse.self, from: data)
            }
        }

        // USB fallback: park in outbox, wait for the bridge to deliver.
        // Bounded so a dead bridge surfaces as a visible error rather than an
        // indefinite spinner - the learner can then switch transport or read
        // the built-in book.
        transport = "usb"
        let data: Data = try await withCheckedThrowingContinuation { cont in
            outbox = body
            var done = false
            delivered = { d in
                guard !done else { return }
                done = true
                cont.resume(returning: d)
            }
            DispatchQueue.main.asyncAfter(deadline: .now() + usbTimeout) { [weak self] in
                guard !done else { return }
                done = true
                self?.outbox = nil
                self?.delivered = nil
                self?.transport = "offline"
                cont.resume(throwing: URLError(.timedOut))
            }
        }
        outbox = nil
        delivered = nil
        return try JSONDecoder().decode(ExchangeResponse.self, from: data)
    }

    // MARK: - USB listener (port 8081, reached via `iproxy 8081 8081`)

    private func startUSBListener() {
        guard let l = try? NWListener(using: .tcp, on: 8081) else { return }
        listener = l
        l.newConnectionHandler = { [weak self] conn in
            conn.start(queue: .global())
            self?.receive(on: conn, buffer: Data())
        }
        l.start(queue: .global())
    }

    private nonisolated func receive(on conn: NWConnection, buffer: Data) {
        conn.receive(minimumIncompleteLength: 1, maximumLength: 1 << 20) { [weak self] data, _, done, err in
            guard let self, err == nil else { conn.cancel(); return }
            var buf = buffer
            if let data { buf.append(data) }
            if let request = HTTPRequest(raw: buf) {
                Task { @MainActor in self.route(request, conn) }
            } else if !done {
                self.receive(on: conn, buffer: buf)
            } else {
                conn.cancel()
            }
        }
    }

    private func route(_ req: HTTPRequest, _ conn: NWConnection) {
        switch (req.method, req.path) {
        case ("GET", "/pending"):
            respond(conn, status: outbox == nil ? "204 No Content" : "200 OK", body: outbox)
        case ("POST", "/deliver"):
            delivered?(req.body)
            respond(conn, status: "200 OK", body: Data("{\"ok\":true}".utf8))
        case ("GET", "/ping"):
            respond(conn, status: "200 OK", body: Data("{\"ok\":true}".utf8))
        default:
            respond(conn, status: "404 Not Found", body: nil)
        }
    }

    private func respond(_ conn: NWConnection, status: String, body: Data?) {
        var head = "HTTP/1.1 \(status)\r\nContent-Type: application/json\r\n"
        head += "Content-Length: \(body?.count ?? 0)\r\nConnection: close\r\n\r\n"
        var out = Data(head.utf8)
        if let body { out.append(body) }
        conn.send(content: out, completion: .contentProcessed { _ in conn.cancel() })
    }
}

/// Just enough HTTP parsing for the two bridge endpoints.
private struct HTTPRequest {
    let method: String
    let path: String
    let body: Data

    init?(raw: Data) {
        guard let headerEnd = raw.range(of: Data("\r\n\r\n".utf8)) else { return nil }
        guard let head = String(data: raw[..<headerEnd.lowerBound], encoding: .utf8) else { return nil }
        let lines = head.components(separatedBy: "\r\n")
        let parts = lines[0].components(separatedBy: " ")
        guard parts.count >= 2 else { return nil }
        method = parts[0]
        path = parts[1]
        var contentLength = 0
        for line in lines.dropFirst() where line.lowercased().hasPrefix("content-length:") {
            contentLength = Int(line.dropFirst("content-length:".count).trimmingCharacters(in: .whitespaces)) ?? 0
        }
        let bodyData = raw[headerEnd.upperBound...]
        guard bodyData.count >= contentLength else { return nil }  // wait for more
        body = Data(bodyData.prefix(contentLength))
    }
}
