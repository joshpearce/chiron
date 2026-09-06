import SwiftUI

/// Two copies of a unit's marks and ink that both changed while the
/// devices were apart. The reader keeps this device's, keeps the
/// server's, or hands both to the agent on the sprite to reconcile.
struct ConflictCard: View {
    @EnvironmentObject var session: BookSession
    let conflict: BookSession.Conflict

    private func describe(_ a: Annotations) -> String {
        let highlights = a.marks.filter { $0.kind == .highlight }.count
        let questions = a.marks.filter { $0.kind != .highlight }.count
        var parts: [String] = []
        parts.append(highlights == 1 ? "1 highlight" : "\(highlights) highlights")
        parts.append(questions == 1 ? "1 question or note" : "\(questions) questions or notes")
        parts.append(a.ink == nil ? "no ink" : "ink")
        parts.append("read to \(Int((a.position * 100).rounded()))%")
        return parts.joined(separator: " · ")
    }

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 18) {
                Text("This unit was marked up on two devices while they were apart.")
                    .font(Typography.serif(18))
                VStack(alignment: .leading, spacing: 10) {
                    row("This device", describe(conflict.mine))
                    row(conflict.theirs.device.map { "The server (last from \($0))" } ?? "The server", describe(conflict.theirs))
                }
                .padding(16)
                .background(.fill.tertiary, in: .rect(cornerRadius: 16))
                Text("Keep one copy, or let the agent on the sprite put them together: every mark from both, the further reading position, and both inks laid over each other.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
                Spacer()
                VStack(spacing: 10) {
                    choice("Keep this device's copy", .mine, prominent: false)
                    choice("Keep the server's copy", .theirs, prominent: false)
                    choice("Let the agent reconcile", .agent, prominent: true)
                }
                if session.resolving {
                    HStack(spacing: 10) {
                        ProgressView()
                        Text("Working on it.").foregroundStyle(.secondary)
                    }
                }
            }
            .padding(24)
            .navigationTitle("Two copies")
            .toolbarTitleDisplayMode(.inline)
        }
        .interactiveDismissDisabled()
    }

    private func row(_ who: String, _ what: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(who).font(Typography.sans(16, weight: .semibold))
            Text(what).font(.footnote).foregroundStyle(.secondary)
        }
    }

    @ViewBuilder
    private func choice(_ label: String, _ r: BookSession.Resolution, prominent: Bool) -> some View {
        let button = Button {
            Task { await session.resolveConflict(r) }
        } label: {
            Text(label).frame(maxWidth: .infinity)
        }
        .disabled(session.resolving)
        .accessibilityIdentifier("conflict-\(r.rawValue)")
        if prominent {
            button.buttonStyle(.borderedProminent)
        } else {
            button.buttonStyle(.bordered)
        }
    }
}
