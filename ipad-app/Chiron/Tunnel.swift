import Foundation
import Network

/// The gate's /ssh tunnel, presented as a loopback TCP port so an ssh client
/// that only knows host:port can use it. Each accepted connection gets its
/// own WebSocket; bytes are relayed both ways until either side closes.
///
/// Opening the WebSocket is an ordinary HTTPS request to the sprite, which
/// is also what wakes it from hibernation.
enum TunnelError: LocalizedError {
    case url(String), listen(String)
    var errorDescription: String? {
        switch self {
        case .url(let u): return "Not a usable server address: \(u)"
        case .listen(let why): return "Could not open the local tunnel port: \(why)"
        }
    }
}

final class Tunnel {
    let port: UInt16
    private let listener: NWListener
    private let request: URLRequest
    private static let queue = DispatchQueue(label: "chiron.tunnel")
    private var links: [Link] = []

    private init(listener: NWListener, port: UInt16, request: URLRequest) {
        self.listener = listener
        self.port = port
        self.request = request
        // Set before start: a listener started without a connection handler
        // fails with EINVAL, which reads like a bad port and is not.
        listener.newConnectionHandler = { [weak self] conn in self?.accept(conn) }
    }

    /// Listen on a loopback port and be ready to bridge. The port is chosen
    /// here rather than by the system so a collision is a retry, not a
    /// failure.
    static func open(base: String, key: String?) async throws -> Tunnel {
        var url = base.trimmingCharacters(in: .whitespacesAndNewlines)
        while url.hasSuffix("/") { url.removeLast() }
        url = url.replacingOccurrences(of: "^http", with: "ws", options: .regularExpression) + "/ssh"
        guard let u = URL(string: url), let scheme = u.scheme, ["ws", "wss"].contains(scheme), u.host != nil else {
            throw TunnelError.url(url)
        }
        var req = URLRequest(url: u, timeoutInterval: 30)
        if let key, !key.isEmpty { req.setValue("Bearer \(key)", forHTTPHeaderField: "Authorization") }

        var failure = "the loopback listener never became ready"
        for _ in 0..<8 {
            let candidate = UInt16.random(in: 20000...60000)
            let l = try NWListener(using: .tcp, on: NWEndpoint.Port(rawValue: candidate)!)
            let tunnel = Tunnel(listener: l, port: candidate, request: req)
            if let why = await tunnel.start() {
                failure = why
                l.cancel()
                continue
            }
            return tunnel
        }
        throw TunnelError.listen(failure)
    }

    /// Start listening; nil on ready, otherwise why not.
    private func start() async -> String? {
        await withCheckedContinuation { cont in
            var done = false
            listener.stateUpdateHandler = { state in
                guard !done else { return }
                switch state {
                case .ready: done = true; cont.resume(returning: nil)
                case .failed(let err): done = true; cont.resume(returning: String(describing: err))
                case .cancelled: done = true; cont.resume(returning: "cancelled")
                default: break
                }
            }
            listener.start(queue: Self.queue)
        }
    }

    func close() {
        listener.cancel()
        Self.queue.async { self.links.forEach { $0.close() }; self.links.removeAll() }
    }

    private func accept(_ conn: NWConnection) {
        // The port is for the ssh client in this process, nothing else.
        guard case .hostPort(let host, _) = conn.endpoint, Self.isLoopback(host) else {
            conn.cancel()
            return
        }
        let link = Link(conn: conn, socket: URLSession.shared.webSocketTask(with: request), queue: Self.queue)
        links.append(link)
        link.start()
    }

    private static func isLoopback(_ host: NWEndpoint.Host) -> Bool {
        switch host {
        case .ipv4(let a): return a.isLoopback
        case .ipv6(let a): return a.isLoopback
        case .name(let n, _): return n == "localhost"
        @unknown default: return false
        }
    }
    /// One TCP connection paired with one WebSocket.
    private final class Link {
        let conn: NWConnection
        let socket: URLSessionWebSocketTask
        let queue: DispatchQueue
        private var closed = false

        init(conn: NWConnection, socket: URLSessionWebSocketTask, queue: DispatchQueue) {
            self.conn = conn
            self.socket = socket
            self.queue = queue
        }

        func start() {
            socket.maximumMessageSize = 1 << 20
            socket.resume()
            conn.stateUpdateHandler = { [weak self] state in
                switch state {
                case .failed, .cancelled: self?.close()
                default: break
                }
            }
            conn.start(queue: queue)
            pumpSocket()
            pumpConn()
        }

        func close() {
            guard !closed else { return }
            closed = true
            socket.cancel(with: .normalClosure, reason: nil)
            conn.cancel()
        }

        private func pumpSocket() {
            socket.receive { [weak self] result in
                guard let self else { return }
                switch result {
                case .success(.data(let data)):
                    self.conn.send(content: data, completion: .contentProcessed { _ in })
                    self.pumpSocket()
                case .success(.string(let s)):
                    self.conn.send(content: Data(s.utf8), completion: .contentProcessed { _ in })
                    self.pumpSocket()
                case .success:
                    self.pumpSocket()
                case .failure:
                    self.close()
                }
            }
        }

        private func pumpConn() {
            conn.receive(minimumIncompleteLength: 1, maximumLength: 32 * 1024) { [weak self] data, _, done, err in
                guard let self else { return }
                if let data, !data.isEmpty {
                    self.socket.send(.data(data)) { _ in }
                }
                if done || err != nil { self.close(); return }
                self.pumpConn()
            }
        }
    }
}
