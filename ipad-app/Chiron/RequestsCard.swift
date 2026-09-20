import SwiftUI

/// "Request a change": what the reader wants different, sent to the
/// development agent on the sprite with a picture of where they were,
/// and every request so far with what the agent has done about it. A
/// ready request whose build is newer than this app offers Install.
struct RequestsCard: View {
    @EnvironmentObject var library: Library
    @State private var text = ""
    @State private var withPicture = true
    @State private var sending = false
    @FocusState private var typing: Bool

    var body: some View {
        NavigationStack {
            List {
                Section {
                    TextField("What should change?", text: $text, axis: .vertical)
                        .lineLimit(3...8)
                        .focused($typing)
                        .accessibilityLabel("The change you want")
                    Toggle("Send a picture of this screen", isOn: $withPicture)
                    if let err = library.requestError {
                        Text(err).foregroundStyle(.red).font(.callout)
                    }
                    Button {
                        send()
                    } label: {
                        if sending { ProgressView() } else { Text("Send to the agent") }
                    }
                    .disabled(sending || text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                } footer: {
                    Text("The agent on the sprite makes the change, runs the tests, and builds the app on the MacBook. Watch it here; Install appears when the build is ready.")
                }
                Section("Requests") {
                    if library.requests.isEmpty {
                        Text("None yet.").foregroundStyle(.secondary)
                    }
                    ForEach(library.requests) { r in
                        RequestRow(request: r)
                    }
                }
            }
            .navigationTitle("Request a change")
            .toolbarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { library.requestsShown = false }
                }
                ToolbarItem(placement: .topBarLeading) {
                    Button { Task { await library.refreshRequests() } } label: {
                        Label("Refresh", systemImage: "arrow.clockwise")
                    }
                }
            }
            .task {
                await library.refreshRequests()
                // A request in the agent's hands moves; keep up while the card is open.
                while !Task.isCancelled, library.requests.contains(where: \.open) {
                    try? await Task.sleep(for: .seconds(5))
                    await library.refreshRequests()
                    await library.checkForBuild()
                }
            }
            .onAppear { typing = true }
        }
        .presentationSizing(.form)
    }

    private func send() {
        let t = text
        sending = true
        Task {
            if await library.requestChange(t, withPicture: withPicture) != nil { text = "" }
            sending = false
        }
    }
}

struct RequestRow: View {
    @EnvironmentObject var library: Library
    let request: ChangeRequest

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline) {
                Text(request.text).font(Typography.sans(16, weight: .semibold))
                Spacer()
                StatusPill(status: request.status)
            }
            if let last = request.last, request.open {
                Text(last).font(.callout).foregroundStyle(.secondary).lineLimit(3)
            }
            if let summary = request.summary, !summary.isEmpty, !request.open {
                Text(summary).font(.callout)
            }
            if let reason = request.reason, request.status == "failed" {
                Text(reason).font(.footnote.monospaced()).foregroundStyle(.red).lineLimit(6)
            }
            HStack {
                if let c = request.commit { Text("commit \(c)").font(.footnote.monospaced()).foregroundStyle(.secondary) }
                if let b = request.build { Text("build \(b.version) (\(b.build))").font(.footnote).foregroundStyle(.secondary) }
                Spacer()
                if let b = request.build, b.build > library.runningBuild, let url = library.installURL, library.availableBuild?.build == b.build {
                    Button("Install") { UIApplication.shared.open(url) }.buttonStyle(.borderedProminent)
                }
            }
        }
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
    }
}

struct StatusPill: View {
    let status: String

    var body: some View {
        Text(status)
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 8).padding(.vertical, 3)
            .background(colour.opacity(0.18), in: Capsule())
            .foregroundStyle(colour)
    }

    private var colour: Color {
        switch status {
        case "ready": return .green
        case "failed": return .red
        case "queued": return .secondary
        default: return .blue
        }
    }
}

struct RequestButton: View {
    @EnvironmentObject var library: Library

    var body: some View {
        Button { library.requestsShown = true } label: {
            Label("Request a change", systemImage: "wrench.and.screwdriver")
        }
        .keyboardShortcut("r", modifiers: [.command, .shift])
        .accessibilityHint("Ask the agent on the sprite to change the app")
    }
}
