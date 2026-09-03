import Citadel
import Foundation
import NIOCore
import NIOSSH

/// A shell on the sprite: the tunnel gives ssh a loopback port, Citadel
/// speaks SSH through it with this device's key, and the PTY's bytes flow
/// to and from the terminal view. tmux on the far side keeps the session
/// when the connection drops.
@MainActor
final class ShellSession: ObservableObject {
    enum Phase: Equatable {
        case idle, connecting, connected, closed(String?)
    }

    @Published private(set) var phase: Phase = .idle
    /// Bytes the terminal should show; the view drains this.
    var onOutput: ((ArraySlice<UInt8>) -> Void)?
    /// Whatever arrived before a terminal view was attached.
    private var pending: [UInt8] = []

    private var tunnel: Tunnel?
    private var client: SSHClient?
    private var writer: TTYStdinWriter?
    private var task: Task<Void, Never>?
    private var cols = 80, rows = 24

    /// The login user on the sprite. A launch-environment override lets a
    /// Simulator run reach a user-mode sshd on the Mac.
    static var username: String {
        ProcessInfo.processInfo.environment["CHIRON_SHELL_USER"] ?? "sprite"
    }

    func connect(base: String, serverID: UUID?, key: String?) {
        guard phase == .idle || phase.isClosed else { return }
        guard let serverID else {
            phase = .closed("Save the server first: the shell needs this device's key, which belongs to a saved server.")
            return
        }
        phase = .connecting
        task = Task { await run(base: base, serverID: serverID, key: key) }
    }

    private func run(base: String, serverID: UUID, key: String?) async {
        do {
            let tunnel = try await Tunnel.open(base: base, key: key)
            self.tunnel = tunnel
            let client = try await SSHClient.connect(
                host: "127.0.0.1", port: Int(tunnel.port),
                authenticationMethod: .ed25519(username: Self.username, privateKey: DeviceKey.privateKey(for: serverID)),
                // The tunnel is TLS to the sprite's own domain, and the
                // shared key had to open it, so the far end is already the
                // right host before ssh starts.
                hostKeyValidator: .acceptAnything(),
                reconnect: .never)
            self.client = client
            phase = .connected
            let pty = SSHChannelRequestEvent.PseudoTerminalRequest(
                wantReply: true, term: "xterm-256color",
                terminalCharacterWidth: cols, terminalRowHeight: rows,
                terminalPixelWidth: 0, terminalPixelHeight: 0,
                terminalModes: .init([:]))
            try await client.withPTY(pty) { output, writer in
                await MainActor.run { self.writer = writer }
                for try await chunk in output {
                    let bytes: [UInt8]
                    switch chunk {
                    case .stdout(let b), .stderr(let b):
                        bytes = Array(b.readableBytesView)
                    }
                    await MainActor.run { self.deliver(bytes[...]) }
                }
            }
            finish(nil)
        } catch {
            finish(Self.describe(error))
        }
    }

    private func deliver(_ bytes: ArraySlice<UInt8>) {
        if let onOutput {
            onOutput(bytes)
        } else {
            pending.append(contentsOf: bytes)
        }
    }

    /// Called by the terminal view once it exists, to catch up.
    func attach(_ sink: @escaping (ArraySlice<UInt8>) -> Void) {
        onOutput = sink
        if !pending.isEmpty {
            sink(pending[...])
            pending.removeAll()
        }
    }

    func send(_ bytes: ArraySlice<UInt8>) {
        guard let writer else { return }
        let data = Array(bytes)
        Task { try? await writer.write(ByteBuffer(bytes: data)) }
    }

    func send(text: String) {
        send(Array(text.utf8)[...])
    }

    func resize(cols: Int, rows: Int) {
        guard cols > 0, rows > 0, cols != self.cols || rows != self.rows else { return }
        self.cols = cols
        self.rows = rows
        guard let writer else { return }
        Task { try? await writer.changeSize(cols: cols, rows: rows, pixelWidth: 0, pixelHeight: 0) }
    }

    func close() {
        task?.cancel()
        finish(nil)
    }

    private func finish(_ error: String?) {
        guard phase != .idle, !phase.isClosed else { return }
        phase = .closed(error)
        writer = nil
        let client = self.client
        self.client = nil
        tunnel?.close()
        tunnel = nil
        Task { try? await client?.close() }
    }

    private static func describe(_ error: Error) -> String {
        let text = (error as? LocalizedError)?.errorDescription ?? String(describing: error)
        if text.contains("401") || text.contains("unauthorized") {
            return "The gate refused the shared key."
        }
        if text.lowercased().contains("authentication") || text.contains("permissionDenied") {
            return "The sprite refused this device's key. Enrol it from the server settings."
        }
        return text
    }
}

extension ShellSession.Phase {
    var isClosed: Bool {
        if case .closed = self { return true }
        return false
    }
}
