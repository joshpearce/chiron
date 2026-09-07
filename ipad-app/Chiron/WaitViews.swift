import SwiftUI

/// The learner is waiting on the server: a short line in the tablet's words
/// over a dimmed screen. Every exchange button is disabled while it shows.
struct WaitOverlay: View {
    let text: String

    var body: some View {
        ZStack {
            Color.black.opacity(0.12).ignoresSafeArea()
            VStack(spacing: 14) {
                ProgressView().controlSize(.large)
                Text(text)
                    .font(Typography.serif(20, relativeTo: .title3))
            }
            .padding(30)
            .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 16))
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(text)
    }
}

/// The next chapter is being written. The server says which of its three
/// stages it is at and when it began; the screen shows the stages ticked
/// off and the time it has been at it, so minutes of waiting read as
/// progress rather than a hang.
struct AuthoringView: View {
    @EnvironmentObject var session: BookSession

    static let stages: [(id: String, label: String)] = [
        ("planning", "Planning the chapter around your answers"),
        ("writing", "Writing the sections"),
        ("pages", "Laying out the pages"),
    ]

    var body: some View {
        let current = Self.stages.firstIndex { $0.id == session.authoringStage } ?? 0
        VStack(alignment: .leading, spacing: 18) {
            Text("The next chapter is being written.")
                .font(Typography.display(30))
                .frame(maxWidth: .infinity)
                .multilineTextAlignment(.center)
            VStack(alignment: .leading, spacing: 12) {
                ForEach(Array(Self.stages.enumerated()), id: \.offset) { i, stage in
                    HStack(spacing: 12) {
                        if i < current {
                            Image(systemName: "checkmark.circle.fill").foregroundStyle(.green)
                        } else if i == current {
                            ProgressView().controlSize(.small)
                        } else {
                            Image(systemName: "circle").foregroundStyle(.tertiary)
                        }
                        Text(stage.label)
                            .font(Typography.serif(20, relativeTo: .title3))
                            .foregroundStyle(i <= current ? .primary : .secondary)
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(stage.label): \(i < current ? "done" : i == current ? "in progress" : "to come")")
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 24)
            TimelineView(.periodic(from: .now, by: 1)) { context in
                Text(elapsed(at: context.date))
                    .font(Typography.sans(16, relativeTo: .callout))
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .monospacedDigit()
            }
        }
        .padding(40)
        .frame(maxWidth: 560)
    }

    private func elapsed(at now: Date) -> String {
        guard let since = session.authoringSince else { return "Starting" }
        let s = max(0, Int(now.timeIntervalSince(since)))
        let clock = String(format: "%d:%02d", s / 60, s % 60)
        return s < 180 ? "\(clock) so far, usually two to four minutes" : "\(clock) so far, still writing"
    }
}

/// The server failed the learner: say so, and offer the one way out.
struct ErrorView: View {
    @EnvironmentObject var session: BookSession
    let message: String

    var body: some View {
        VStack(spacing: 18) {
            Image(systemName: "wifi.exclamationmark")
                .font(.system(size: 44))
                .foregroundStyle(.secondary)
                .accessibilityHidden(true)
            Text(message)
                .font(Typography.display(26))
                .multilineTextAlignment(.center)
            Button("Try again") {
                Task { await session.retry() }
            }
            .buttonStyle(.borderedProminent)
            .disabled(session.busy)
        }
        .padding(40)
        .frame(maxWidth: 560)
    }
}
