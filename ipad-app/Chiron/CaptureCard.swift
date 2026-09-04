import SwiftUI
import Combine

/// "What do you want to know about this?" A capture arrives with its text
/// or image; the reader adds the question and the primer starts on the
/// shelf. From the shelf's Capture button the text is the reader's to
/// paste.
struct CaptureCard: View {
    @EnvironmentObject var library: Library
    @Environment(\.dismiss) private var dismiss
    @State var capture: Capture
    @State private var prompt = ""
    @State private var sending = false
    @State private var error: String?
    @FocusState private var typing: Bool

    private var fromElsewhere: Bool { capture.sourceApp != nil || capture.sourceURL != nil || capture.imagePNG != nil }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    if let png = capture.imagePNG, let image = UIImage(data: png) {
                        Image(uiImage: image)
                            .resizable().scaledToFit()
                            .frame(maxHeight: 220)
                            .clipShape(RoundedRectangle(cornerRadius: 10))
                    }
                    TextEditor(text: $capture.text)
                        .font(Typography.serif(16))
                        .frame(minHeight: 120, maxHeight: 260)
                        .disabled(fromElsewhere && !capture.text.isEmpty)
                } header: {
                    Text(capture.imagePNG != nil && capture.text.isEmpty ? "Captured image" : "Captured text")
                } footer: {
                    if let src = sourceLine { Text(src) }
                    else if !fromElsewhere { Text("Paste or type what you want a primer on.") }
                }
                Section {
                    TextField("What do you want to know about this?", text: $prompt, axis: .vertical)
                        .font(Typography.serif(17))
                        .lineLimit(1...4)
                        .focused($typing)
                        .onSubmit { submit() }
                    if let error {
                        Text(error).font(.footnote).foregroundStyle(.red)
                    }
                } header: {
                    Text("Your question")
                } footer: {
                    Text("The server writes a short primer that answers it from what you captured. It takes about a minute and opens by itself; margin notes extend it later.")
                }
            }
            .navigationTitle("New primer")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { library.pendingCapture = nil }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        submit()
                    } label: {
                        if sending { ProgressView() } else { Text("Make a primer") }
                    }
                    .disabled(!canSubmit)
                }
            }
            .onAppear { typing = true }
        }
        .presentationSizing(.form)
    }

    private var canSubmit: Bool {
        !sending && !capture.isEmpty && !prompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private var sourceLine: String? {
        var parts: [String] = []
        if let app = capture.sourceApp { parts.append("from \(app)") }
        if let url = capture.sourceURL { parts.append(url) }
        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }

    private func submit() {
        guard canSubmit else { return }
        sending = true
        error = nil
        Task {
            do {
                _ = try await library.submitCapture(capture, prompt: prompt)
            } catch {
                self.error = "The server did not take the capture. Try again."
                sending = false
            }
        }
    }
}
