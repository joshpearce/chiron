import SwiftTerm
import SwiftUI

/// The shell sheet: a terminal on the sprite with a status line above it.
struct ShellView: View {
    @EnvironmentObject var library: Library
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var shell: ShellSession

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                Image(systemName: "terminal")
                Text(status).font(Typography.sans(15))
                    .foregroundStyle(statusColor)
                    .lineLimit(2)
                Spacer()
                if case .closed = shell.phase {
                    Button("Reconnect") { open() }
                        .buttonStyle(.bordered)
                }
                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark.circle.fill").font(.title3).foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Close the shell")
            }
            .padding(.horizontal, 14).padding(.vertical, 10)
            .background(.bar)
            TerminalHost(shell: shell)
                .ignoresSafeArea(.keyboard)
        }
        .onAppear { open() }
        // The view's first measurement can land before the PTY exists; once
        // connected, the session sends the size it has.
        .onChange(of: shell.phase) { _, phase in
            if case .connected = phase { TerminalHost.refocus() }
        }
    }

    private func open() {
        shell.connect(base: library.sync.baseURL, serverID: library.sync.serverID,
                      key: library.sync.serverID.flatMap(Credentials.token(for:)))
    }

    private var status: String {
        switch shell.phase {
        case .idle: return "Not connected."
        case .connecting: return "Waking the sprite and opening a shell."
        case .connected: return "Connected to \(library.sync.servers.selected?.name ?? "the sprite")."
        case .closed(let err): return err ?? "The shell closed."
        }
    }

    private var statusColor: SwiftUI.Color {
        if case .closed(let err) = shell.phase, err != nil { return .red }
        return .secondary
    }
}

/// SwiftTerm's view, wired to the session: keys go out, bytes come in,
/// and a resize follows the view.
struct TerminalHost: UIViewRepresentable {
    @ObservedObject var shell: ShellSession

    /// The terminal that is showing, so the keyboard can be handed back to it.
    private static weak var current: TerminalView?
    static func refocus() { DispatchQueue.main.async { _ = current?.becomeFirstResponder() } }

    func makeUIView(context: Context) -> TerminalView {
        let view = TerminalView(frame: .zero, font: UIFont.monospacedSystemFont(ofSize: 14, weight: .regular))
        view.terminalDelegate = context.coordinator
        view.nativeBackgroundColor = UIColor(red: 0.08, green: 0.09, blue: 0.11, alpha: 1)
        view.nativeForegroundColor = UIColor(white: 0.92, alpha: 1)
        view.backgroundColor = view.nativeBackgroundColor
        context.coordinator.view = view
        Self.current = view
        #if DEBUG
        ShellScreen.shared.view = view
        #endif
        shell.attach { bytes in view.feed(byteArray: bytes) }
        DispatchQueue.main.async { _ = view.becomeFirstResponder() }
        return view
    }

    func updateUIView(_ view: TerminalView, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator(shell: shell) }

    final class Coordinator: NSObject, TerminalViewDelegate {
        let shell: ShellSession
        weak var view: TerminalView?
        init(shell: ShellSession) { self.shell = shell }

        // SwiftTerm calls these on the main thread; the session is main-actor.
        func sizeChanged(source: TerminalView, newCols: Int, newRows: Int) {
            MainActor.assumeIsolated { shell.resize(cols: newCols, rows: newRows) }
        }
        func setTerminalTitle(source: TerminalView, title: String) {}
        func hostCurrentDirectoryUpdate(source: TerminalView, directory: String?) {}
        func send(source: TerminalView, data: ArraySlice<UInt8>) {
            MainActor.assumeIsolated { shell.send(data) }
        }
        func scrolled(source: TerminalView, position: Double) {}
        func requestOpenLink(source: TerminalView, link: String, params: [String: String]) {
            if let url = URL(string: link) { UIApplication.shared.open(url) }
        }
        func bell(source: TerminalView) {}
        func clipboardCopy(source: TerminalView, content: Data) {
            UIPasteboard.general.string = String(data: content, encoding: .utf8)
        }
        func rangeChanged(source: TerminalView, startY: Int, endY: Int) {}
    }
}

#if DEBUG
/// What the terminal currently shows, for the harness.
@MainActor
final class ShellScreen {
    static let shared = ShellScreen()
    weak var view: TerminalView?

    func lines() -> [String]? {
        guard let view else { return nil }
        let term = view.getTerminal()
        return (0..<term.rows).compactMap { row in
            term.getLine(row: row)?.translateToString(trimRight: true)
        }
    }
}
#endif
