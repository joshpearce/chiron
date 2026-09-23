import SwiftUI

/// The conversation before a primer or a book is written: the tutor asks
/// what would make it the right one, the reader answers, and the build
/// starts when the reader says so. Leaving keeps the draft on the shelf,
/// with a hammer, to come back to.
struct PlanCard: View {
    @EnvironmentObject var library: Library
    @State private var draft = ""
    @State private var confirmingDiscard = false
    @FocusState private var typing: Bool

    private var state: PlanState? { library.planning }
    private var thing: String { state?.isBook == true ? "book" : "primer" }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                transcript
                Divider()
                footer
            }
            .navigationTitle(state?.title ?? "New \(thing)")
            .navigationSubtitle(state?.isBook == true ? "A smart book from your capture" : "A primer from your capture")
            .toolbarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Later") { library.planning = nil }
                        .accessibilityHint("Keeps the draft on the shelf to come back to")
                }
                ToolbarItem(placement: .topBarLeading) {
                    // What was said here is often wanted elsewhere: handed to
                    // another agent, pasted into a note, kept.
                    Button {
                        if let s = state { UIPasteboard.general.string = s.transcript }
                    } label: { Label("Copy the conversation", systemImage: "doc.on.doc") }
                    .keyboardShortcut("c", modifiers: [.command, .shift])
                    .disabled(state == nil)
                }
                ToolbarItem(placement: .destructiveAction) {
                    Button {
                        confirmingDiscard = true
                    } label: { Label("Discard", systemImage: "trash") }
                    .confirmationDialog("Discard this \(thing)?", isPresented: $confirmingDiscard, titleVisibility: .visible) {
                        Button("Discard the draft", role: .destructive) { Task { await library.discardDraft() } }
                        Button("Keep it", role: .cancel) {}
                    } message: {
                        Text("The capture and this conversation go with it.")
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    // Prominent once the tutor has a brief; there before that
                    // too, for a reader who wants it built from what was said.
                    if state?.done == true {
                        buildButton.buttonStyle(.borderedProminent)
                    } else {
                        buildButton
                    }
                }
            }
        }
        .presentationSizing(.form)
        .onAppear { typing = true }
    }

    private var buildButton: some View {
        Button {
            Task { await library.buildDraft() }
        } label: {
            Text(state?.isBook == true ? "Build the book" : "Write the primer")
        }
        .disabled(library.planBusy)
    }

    private var transcript: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if let s = state {
                        VStack(alignment: .leading, spacing: 6) {
                            if let text = s.source?.text, !text.isEmpty {
                                Text(text)
                                    .font(Typography.serifItalic(15))
                                    .foregroundStyle(.secondary)
                                    .lineLimit(4)
                            }
                            Text(s.prompt).font(Typography.serif(17, weight: .semibold))
                                .textSelection(.enabled)
                        }
                        .padding(14)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(.fill.tertiary, in: .rect(cornerRadius: 16))
                        ForEach(Array(s.plan.enumerated()), id: \.offset) { _, m in
                            bubble(m)
                        }
                        if s.done, let brief = s.brief, !brief.isEmpty {
                            VStack(alignment: .leading, spacing: 6) {
                                Text("The brief").font(.footnote.weight(.semibold)).foregroundStyle(.secondary)
                                Text(brief).font(Typography.serif(16))
                                    .textSelection(.enabled)
                                Button {
                                    UIPasteboard.general.string = brief
                                } label: { Label("Copy the brief", systemImage: "doc.on.doc") }
                                    .font(.footnote)
                                    .buttonStyle(.bordered)
                            }
                            .padding(14)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.accentColor.opacity(0.1), in: .rect(cornerRadius: 16))
                        }
                        if library.planBusy {
                            HStack(spacing: 8) {
                                ProgressView().controlSize(.small)
                                Text("Thinking…").foregroundStyle(.secondary).font(.callout)
                            }
                        }
                        if let err = library.planError {
                            Text(err).foregroundStyle(.red).font(.callout)
                        }
                        if let err = s.error, s.status == "failed" {
                            Text(err).foregroundStyle(.red).font(.callout)
                        }
                    }
                    Color.clear.frame(height: 1).id("bottom")
                }
                .padding()
            }
            .onChange(of: state?.plan.count) { _, _ in
                withAnimation { proxy.scrollTo("bottom", anchor: .bottom) }
            }
        }
    }

    private func bubble(_ m: PlanMessage) -> some View {
        HStack {
            if m.role == "learner" { Spacer(minLength: 60) }
            Text(m.text)
                .font(Typography.serif(16))
                .textSelection(.enabled)
                .padding(12)
                .background(m.role == "learner" ? AnyShapeStyle(Color.accentColor.opacity(0.15)) : AnyShapeStyle(.fill.tertiary),
                            in: .rect(cornerRadius: 16))
                .frame(maxWidth: .infinity, alignment: m.role == "learner" ? .trailing : .leading)
            if m.role != "learner" { Spacer(minLength: 60) }
        }
        .contextMenu {
            Button {
                UIPasteboard.general.string = m.text
            } label: { Label("Copy", systemImage: "doc.on.doc") }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(m.role == "learner" ? "You" : "Tutor"): \(m.text)")
    }

    private var footer: some View {
        HStack(spacing: 10) {
            TextField(state?.done == true ? "Anything to change?" : "Your answer", text: $draft, axis: .vertical)
                .font(Typography.serif(17))
                .lineLimit(1...4)
                .textFieldStyle(.roundedBorder)
                .focused($typing)
                .disabled(library.planBusy)
                .onSubmit(send)
            Button("Send", action: send)
                .buttonStyle(.borderedProminent)
                .disabled(library.planBusy || draft.trimmingCharacters(in: .whitespaces).isEmpty)
        }
        .padding()
    }

    private func send() {
        let text = draft
        guard !text.trimmingCharacters(in: .whitespaces).isEmpty else { return }
        draft = ""
        Task { await library.planReply(text) }
    }
}
