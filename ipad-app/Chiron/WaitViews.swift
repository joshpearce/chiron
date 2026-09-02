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
                    .font(.system(.title3, design: .serif))
            }
            .padding(30)
            .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 16))
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(text)
    }
}

/// The next chapter is being written. Static on purpose: the model takes
/// minutes, and a spinner for minutes reads as a hang.
struct AuthoringView: View {
    @State private var long = false

    var body: some View {
        VStack(spacing: 18) {
            Text("The next chapter is being written.")
                .font(.system(size: 30, weight: .semibold, design: .serif))
                .multilineTextAlignment(.center)
            Text(long
                 ? "Still writing. This can take a few minutes; the book opens on it as soon as it is done."
                 : "It is being shaped by what you just answered.")
                .font(.system(.title3, design: .serif))
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
            ProgressView()
        }
        .padding(40)
        .frame(maxWidth: 560)
        .task {
            try? await Task.sleep(nanoseconds: 60_000_000_000)
            long = true
        }
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
                .font(.system(size: 26, weight: .semibold, design: .serif))
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
