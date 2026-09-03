import SwiftUI

/// A question about a highlighted passage, and the tutor's answer. A card
/// over the page's corner rather than a sheet: the passage stays readable
/// while the question is typed and the answer read.
struct AskCard: View {
    @EnvironmentObject var session: BookSession
    @State private var question = ""
    @FocusState private var typing: Bool

    private var asking: BookSession.Asking? { session.asking }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top) {
                Text(asking?.mark.text ?? "")
                    .font(Typography.serifItalic(15))
                    .foregroundStyle(.secondary)
                    .lineLimit(3)
                Spacer(minLength: 8)
                Button {
                    session.closeAsking()
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.title3)
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Close")
                .keyboardShortcut(.cancelAction)
            }

            if let a = asking, let q = a.mark.question {
                Text(q)
                    .font(Typography.sans(17, weight: .semibold))
                if a.busy {
                    HStack(spacing: 10) {
                        ProgressView()
                        Text(isNote ? "Extending the primer." : "Asking the tutor.")
                            .font(Typography.serif(16))
                            .foregroundStyle(.secondary)
                    }
                } else if let answer = a.mark.answer {
                    if isNote {
                        // The section is on the page; the card only says where.
                        Label("Added at the end: \(answer)", systemImage: "text.append")
                            .font(Typography.serif(16))
                    } else {
                        // Hugs a short answer; scrolls a long one.
                        ScrollView {
                            MathText(text: answer, size: 16, rich: true)
                        }
                        .frame(maxHeight: 320)
                        .fixedSize(horizontal: false, vertical: true)
                    }
                } else if let err = a.error {
                    Text(err)
                        .font(Typography.serif(16))
                        .foregroundStyle(.red)
                    Button("Try again") { Task { await send(q) } }
                        .buttonStyle(.bordered)
                }
            } else {
                TextField(isNote ? "Your note on this passage" : "Your question about this passage", text: $question, axis: .vertical)
                    .font(Typography.serif(17))
                    .lineLimit(1...4)
                    .textFieldStyle(.roundedBorder)
                    .focused($typing)
                    .onSubmit { submit() }
                HStack {
                    Spacer()
                    Button(isNote ? "Add to the primer" : "Ask") { submit() }
                        .buttonStyle(.borderedProminent)
                        .keyboardShortcut(.defaultAction)
                        .disabled(question.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
        .padding(16)
        .frame(width: 400)
        .glassEffect(.regular, in: RoundedRectangle(cornerRadius: 18))
        .shadow(color: .black.opacity(0.12), radius: 14, y: 4)
        .onAppear { typing = asking?.mark.question == nil }
        .onChange(of: asking?.mark.id) { _, _ in
            question = ""
            typing = asking?.mark.question == nil
        }
    }

    private var isNote: Bool { asking?.mark.kind == .note }

    private func submit() {
        let q = question
        guard !q.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        Task { await send(q) }
    }

    private func send(_ q: String) async {
        if isNote { await session.extend(q) } else { await session.ask(q) }
    }
}
