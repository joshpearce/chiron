import Foundation
import UIKit

/// The sprite agent's line into this app. The iPad has no inbound path, so
/// the app connects out to the server's /agent/app and holds the socket
/// open; commands arrive as {id, verb, args} and each gets a {id, result}
/// or {id, error} back. Off unless the reader turns it on, and visibly on
/// while connected: a remote agent moving the screen without a sign of it
/// would be a trust failure.
@MainActor
final class AgentLink: ObservableObject {
    @Published var enabled: Bool {
        didSet { UserDefaults.standard.set(enabled, forKey: "agentEnabled") }
    }
    @Published private(set) var connected = false {
        didSet { AgentBadgeState.shared.driven = connected }
    }

    private weak var library: Library?
    private var task: Task<Void, Never>?
    private var socket: URLSessionWebSocketTask?

    init() {
        enabled = UserDefaults.standard.bool(forKey: "agentEnabled")
    }

    func attach(_ library: Library) {
        self.library = library
        if enabled { start() }
    }

    func start() {
        guard task == nil else { return }
        task = Task { await loop() }
    }

    func stop() {
        task?.cancel()
        task = nil
        socket?.cancel(with: .normalClosure, reason: nil)
        socket = nil
        connected = false
    }

    private func loop() async {
        var backoff: UInt64 = 1
        while !Task.isCancelled, enabled {
            if await session() {
                backoff = 1
            }
            connected = false
            guard !Task.isCancelled, enabled else { break }
            try? await Task.sleep(nanoseconds: backoff * 1_000_000_000)
            backoff = min(backoff * 2, 30)
        }
        task = nil
    }

    /// One connection, until it drops. Returns whether it ever came up.
    private func session() async -> Bool {
        guard let library, let sync = Optional(library.sync) else { return false }
        var base = sync.baseURL.trimmingCharacters(in: .whitespacesAndNewlines)
        while base.hasSuffix("/") { base.removeLast() }
        base = base.replacingOccurrences(of: "^http", with: "ws", options: .regularExpression)
        let name = UIDevice.current.name.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "iPad"
        guard let url = URL(string: "\(base)/agent/app?name=\(name)") else { return false }
        var req = URLRequest(url: url, timeoutInterval: 30)
        Credentials.authorize(&req, serverID: sync.serverID)
        let ws = URLSession.shared.webSocketTask(with: req)
        ws.maximumMessageSize = 16 << 20
        socket = ws
        ws.resume()
        var cameUp = false
        while !Task.isCancelled {
            let message: URLSessionWebSocketTask.Message
            do {
                message = try await ws.receive()
            } catch {
                break
            }
            if !cameUp { cameUp = true; connected = true }
            let data: Data
            switch message {
            case .data(let d): data = d
            case .string(let s): data = Data(s.utf8)
            @unknown default: continue
            }
            guard let cmd = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let id = cmd["id"] as? Int, let verb = cmd["verb"] as? String else { continue }
            let args = cmd["args"] as? [String: Any] ?? [:]
            var reply: [String: Any] = ["id": id]
            do {
                reply["result"] = try await AppCommands.run(verb, args: args, library: library)
            } catch {
                reply["error"] = "\(error)"
            }
            if let out = try? JSONSerialization.data(withJSONObject: reply) {
                try? await ws.send(.data(out))
            }
        }
        // A connection that opened counts as up even if it later dropped;
        // the first receive completing is the sign the server accepted us.
        if !cameUp, ws.closeCode == .invalid, ws.state == .running { cameUp = true }
        ws.cancel(with: .normalClosure, reason: nil)
        socket = nil
        return cameUp
    }
}
