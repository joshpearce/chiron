import SwiftUI

/// Shared flow for the calibration series, pretests and terminal checks:
/// one item at a time, confidence committed BEFORE any reveal, elaborated
/// feedback after. Free-text items show the reference answer for
/// self-comparison; the authoritative grade arrives with the exchange (the
/// results screen). With `reveal` off - the calibration series, which is
/// measurement - committing moves straight to the next item.
struct ItemFlowView: View {
    let title: String
    let subtitle: String
    let items: [CheckItem]
    var reveal = true
    let submitLabel: String
    /// Present on the terminal check, absent on the pretest (which the learner
    /// never enters by accident - it opens itself).
    var onExit: (() -> Void)? = nil
    let onSubmit: ([ItemResponse]) async -> Void

    @State private var index = 0
    @State private var text = ""
    @State private var selected: Int?
    @State private var confidence: Double = 2
    @State private var revealed = false
    @State private var responses: [ItemResponse] = []
    @State private var confirmingExit = false

    var item: CheckItem { items[index] }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            VStack(alignment: .leading, spacing: 6) {
                if onExit != nil {
                    Button {
                        // The check is closed-book on purpose - reading the
                        // chapter mid-check is the crutch effect the design
                        // exists to prevent. An accidental tap with nothing
                        // answered costs nothing; leaving with answers on the
                        // board discards them so a re-entered check starts
                        // honest rather than half-open-book.
                        if responses.isEmpty && !revealed {
                            onExit?()
                        } else {
                            confirmingExit = true
                        }
                    } label: {
                        Label("Back to the chapter", systemImage: "chevron.left")
                            .font(.callout)
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                }
                Text(title).font(.system(.title2, design: .serif).weight(.semibold))
                Text(subtitle).font(.callout).foregroundStyle(.secondary)
                ProgressView(value: Double(index), total: Double(max(items.count, 1)))
                Text("\(index + 1) of \(items.count)")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .accessibilityLabel("Question \(index + 1) of \(items.count)")
            }

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    MathText(text: item.prompt, size: 20)

                    if item.kind == "mcq", let options = item.options {
                        ForEach(options.indices, id: \.self) { i in
                            Button {
                                if !revealed { selected = i }
                            } label: {
                                HStack(alignment: .top) {
                                    Image(systemName: iconFor(i))
                                    MathText(text: options[i].text, size: 17)
                                    Spacer(minLength: 0)
                                }
                                .padding(10)
                                .background(backgroundFor(i), in: RoundedRectangle(cornerRadius: 10))
                            }
                            .buttonStyle(.plain)
                            if revealed, let r = item.reveal?.options?[i],
                               selected == i || r.correct {
                                MathText(text: r.explain, size: 15)
                                    .foregroundStyle(r.correct ? .green : .orange)
                                    .padding(.leading, 30)
                            }
                        }
                    } else {
                        TextEditor(text: $text)
                            .frame(minHeight: 140)
                            .padding(6)
                            .overlay(RoundedRectangle(cornerRadius: 10).stroke(.quaternary))
                            .disabled(revealed)
                        if revealed, let reveal = item.reveal {
                            VStack(alignment: .leading, spacing: 8) {
                                Text("Reference answer").font(.headline)
                                MathText(text: reveal.answer ?? "", size: 17)
                                if item.check == "llm" {
                                    Text("Your answer will be graded against the rubric at the next check-in.")
                                        .font(.footnote).foregroundStyle(.secondary)
                                }
                            }
                            .padding(12)
                            .background(.quaternary.opacity(0.4), in: RoundedRectangle(cornerRadius: 10))
                        }
                    }
                }
            }

            if !revealed {
                VStack(alignment: .leading, spacing: 4) {
                    Text("How confident are you? \(confidenceLabel)").font(.callout)
                    Slider(value: $confidence, in: 1...4, step: 1)
                        .frame(maxWidth: 320)
                }
                HStack(spacing: 16) {
                    // Not knowing is expected - especially on pretests - and
                    // saying so is better signal than typing filler to get
                    // past a required field. Always leftmost, always there.
                    Button("I don't know") {
                        answered(ItemResponse(
                            itemId: item.id,
                            response: nil,
                            selectedIndex: nil,
                            confidence: 1,
                            idk: true))
                    }
                    .buttonStyle(.bordered)

                    Button(reveal ? "Commit answer" : (index == items.count - 1 ? submitLabel : "Next")) {
                        answered(ItemResponse(
                            itemId: item.id,
                            response: item.kind == "mcq" ? nil : text,
                            selectedIndex: selected,
                            confidence: Int(confidence)))
                    }
                    .buttonStyle(.borderedProminent)
                    .keyboardShortcut(.defaultAction)
                    .disabled(item.kind == "mcq" ? selected == nil : text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            } else {
                Button(index == items.count - 1 ? submitLabel : "Next") { advance() }
                    .buttonStyle(.borderedProminent)
                    .keyboardShortcut(.defaultAction)
            }
        }
        .padding(28)
        .alert("Leave the check?", isPresented: $confirmingExit) {
            Button("Stay", role: .cancel) { }
            Button("Discard answers and go back", role: .destructive) { onExit?() }
        } message: {
            Text("The check is closed-book, so answers so far are discarded - re-entering starts it fresh.")
        }
        .frame(maxWidth: 760)
    }

    private func answered(_ r: ItemResponse) {
        responses.append(r)
        if reveal {
            withAnimation { revealed = true }
        } else {
            advance()
        }
    }

    private func advance() {
        if index == items.count - 1 {
            let out = responses
            Task { await onSubmit(out) }
        } else {
            index += 1
            text = ""; selected = nil; confidence = 2; revealed = false
        }
    }

    private var confidenceLabel: String {
        ["", "guessing", "unsure", "fairly sure", "certain"][Int(confidence)]
    }

    private func iconFor(_ i: Int) -> String {
        if !revealed { return selected == i ? "largecircle.fill.circle" : "circle" }
        guard let r = item.reveal?.options?[i] else { return "circle" }
        if r.correct { return "checkmark.circle.fill" }
        return selected == i ? "xmark.circle.fill" : "circle"
    }

    private func backgroundFor(_ i: Int) -> Color {
        if !revealed { return selected == i ? Color.accentColor.opacity(0.15) : Color.clear }
        guard let r = item.reveal?.options?[i] else { return .clear }
        if r.correct { return .green.opacity(0.12) }
        return selected == i ? .red.opacity(0.10) : .clear
    }
}

struct BreakView: View {
    @EnvironmentObject var session: BookSession
    let suggestion: BreakSuggestion
    @State private var remaining: Int = 0
    @State private var started = Date()

    var body: some View {
        VStack(spacing: 24) {
            Image(systemName: "moon.zzz").font(.system(size: 56)).foregroundStyle(.indigo)
            Text("Break time").font(.largeTitle.weight(.semibold))
            Text(suggestion.note).multilineTextAlignment(.center).foregroundStyle(.secondary)
            Text(timeString)
                .font(.system(size: 54, weight: .light, design: .monospaced))
            Text("Genuinely unstimulated rest consolidates what you just learned.\nEyes closed beats a movie.")
                .font(.callout).multilineTextAlignment(.center).foregroundStyle(.secondary)
            Button("Back to the book") {
                Task { await session.breakFinished(minutes: Date().timeIntervalSince(started) / 60) }
            }
            .buttonStyle(.borderedProminent)
        }
        .onAppear { remaining = suggestion.minutes * 60 }
        .task {
            while remaining > 0 && !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 1_000_000_000)
                remaining -= 1
            }
        }
    }

    private var timeString: String {
        String(format: "%d:%02d", remaining / 60, remaining % 60)
    }
}

/// The contents: every chapter of the book with its status, what is owed,
/// and the way to start the book over.
struct ContentsView: View {
    @EnvironmentObject var session: BookSession
    @Environment(\.presentationMode) private var presentation
    @State private var confirmingReset = false

    var body: some View {
        NavigationView {
            List {
                if session.bookState == nil {
                    Section {
                        Text("No progress loaded yet.")
                        Text("If the server is reachable this fills in on its own; pull to refresh otherwise.")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
                if let state = session.bookState {
                    Section("Progress") {
                        ForEach(state.spine) { entry in
                            HStack {
                                Image(systemName: icon(entry.status))
                                    .foregroundStyle(color(entry.status))
                                VStack(alignment: .leading) {
                                    Text(entry.title)
                                    if let s = entry.score {
                                        Text("\(Int(s * 100))%").font(.caption).foregroundStyle(.secondary)
                                    }
                                }
                                Spacer()
                                if entry.inFringe && entry.status != "active" {
                                    Button("Read next") {
                                        Task { await session.start(choice: entry.unit) }
                                        presentation.wrappedValue.dismiss()
                                    }
                                    .buttonStyle(.bordered).controlSize(.small)
                                }
                            }
                        }
                    }
                    if !state.debt.isEmpty {
                        Section("Knowledge debt") {
                            Text("\(state.debt.count) unit(s) skipped or below gate")
                                .foregroundStyle(.orange)
                            Button("Catch me up now") {
                                Task { await session.catchMeUp() }
                                presentation.wrappedValue.dismiss()
                            }
                        }
                    }
                    Section("Where you are") {
                        Text(state.summary).font(.callout).foregroundStyle(.secondary)
                    }
                }
                if let err = session.errorMessage {
                    Section {
                        Text(err).foregroundStyle(.red).font(.callout)
                    }
                }
                Section {
                    Button(role: .destructive) {
                        confirmingReset = true
                    } label: {
                        Label("Start this subject over", systemImage: "arrow.counterclockwise")
                    }
                } footer: {
                    Text("Clears every grade, gate result and debt entry for this subject.")
                }
            }
            .navigationTitle("Contents")
            .refreshable { await session.refreshState() }
            .task { await session.refreshState() }
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { presentation.wrappedValue.dismiss() }
                }
            }
            .alert("Start over?", isPresented: $confirmingReset) {
                Button("Cancel", role: .cancel) { }
                Button("Start over", role: .destructive) {
                    // The contents is a sheet over the screen startOver()
                    // changes. Without dismissing it the reset happens and
                    // the learner sees nothing at all.
                    presentation.wrappedValue.dismiss()
                    Task { await session.startOver() }
                }
            } message: {
                Text("Every grade, gate result and debt entry for this subject is discarded. This cannot be undone from the app.")
            }
        }
        .navigationViewStyle(.stack)
    }

    private func icon(_ s: String) -> String {
        switch s {
        case "passed": return "checkmark.circle.fill"
        case "overridden": return "exclamationmark.triangle.fill"
        case "active": return "book.fill"
        case "failed": return "arrow.counterclockwise.circle"
        default: return "circle.dotted"
        }
    }

    private func color(_ s: String) -> Color {
        switch s {
        case "passed": return .green
        case "overridden": return .orange
        case "active": return .blue
        case "failed": return .red
        default: return .secondary
        }
    }
}
