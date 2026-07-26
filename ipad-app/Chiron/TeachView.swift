import SwiftUI

/// "Teach me something else": the learner says what they want, the tutor asks
/// until it can write a brief, and the server generates a whole subject from it.
///
/// The transcript lives here, not on the server - each turn posts the whole
/// conversation, so a dropped connection costs a retry rather than the thread.
/// Generation itself takes minutes per unit, so it runs server-side and this
/// screen polls; leaving does not cancel it, and the subject appears in the
/// library when every chapter is authored.
@MainActor
final class TeachModel: ObservableObject {
    struct Message: Identifiable, Codable {
        let id: UUID
        let role: String        // learner | tutor
        let text: String

        init(role: String, text: String) {
            self.id = UUID()
            self.role = role
            self.text = text
        }

        // The server sees only role and text.
        enum CodingKeys: String, CodingKey { case role, text }
        init(from decoder: Decoder) throws {
            let c = try decoder.container(keyedBy: CodingKeys.self)
            id = UUID()
            role = try c.decode(String.self, forKey: .role)
            text = try c.decode(String.self, forKey: .text)
        }
    }

    struct TurnResponse: Decodable {
        let replyMd: String
        let done: Bool
        let brief: String
        let slug: String
        let title: String

        enum CodingKeys: String, CodingKey {
            case replyMd = "reply_md"
            case done, brief, slug, title
        }
    }

    struct Job: Decodable {
        let slug: String
        let title: String
        let stage: String       // queued | planning | authoring | ready | failed
        let unitsTotal: Int
        let unitsDone: Int
        let done: Bool
        let error: String?

        enum CodingKeys: String, CodingKey {
            case slug, title, stage, done, error
            case unitsTotal = "units_total"
            case unitsDone = "units_done"
        }

        var line: String {
            switch stage {
            case "queued": return "Queued."
            case "planning": return "Designing the syllabus…"
            case "authoring":
                return unitsTotal > 0
                    ? "Writing chapters — \(unitsDone) of \(unitsTotal) done."
                    : "Writing chapters…"
            case "ready": return "Ready. It is in your library."
            default: return error ?? "Generation failed."
            }
        }
    }

    @Published var messages: [Message] = []
    @Published var thinking = false
    @Published var brief = ""
    @Published var slug = ""
    @Published var title = ""
    @Published var job: Job?
    @Published var error: String?

    var readyToBuild: Bool { !brief.isEmpty && job == nil }

    private let sync: Sync

    init(sync: Sync, demo: Bool = false) {
        self.sync = sync
        if demo { messages = TeachModel.demoTranscript }
    }

    /// Inspection fixture, reachable only via `showteach demo`.
    static let demoTranscript: [Message] = [
        Message(role: "learner", text: "I want to learn about databases I guess."),
        Message(role: "tutor", text: "What do you want to be able to do once you have learned this? Design a schema for your own app, tune a slow production database, or understand how they work under the hood?"),
        Message(role: "learner", text: "I dunno, mostly the tuning side. Stuff comes up at work sometimes."),
        Message(role: "tutor", text: "Which database, and how comfortable are you with reading a query plan or choosing indexes already - total beginner, or have you poked at this before?"),
        Message(role: "learner", text: "Postgres. I'm fine with EXPLAIN and indexes, I'm not looking for that stuff."),
    ]

    func send(_ text: String) async {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        messages.append(Message(role: "learner", text: trimmed))
        error = nil
        thinking = true
        defer { thinking = false }
        do {
            let body = try JSONEncoder().encode(["messages": messages])
            // Elicitation is a model call behind a subprocess; allow for it.
            let turn: TurnResponse = try await post("/teach/turn", body, timeout: 300)
            messages.append(Message(role: "tutor", text: turn.replyMd))
            if turn.done {
                brief = turn.brief
                slug = turn.slug
                title = turn.title
            }
        } catch {
            // The learner's turn stays in the transcript so a retry does not
            // make them type it again.
            self.error = "Could not reach the tutor. Check the connection and send again."
        }
    }

    func build() async {
        do {
            let body = try JSONEncoder().encode(["slug": slug, "title": title, "brief": brief])
            job = try await post("/teach/create", body, timeout: 30)
            await poll()
        } catch {
            self.error = "Could not start generation."
        }
    }

    func poll() async {
        while let j = job, !j.done, !Task.isCancelled {
            try? await Task.sleep(nanoseconds: 5_000_000_000)
            guard let url = URL(string: "\(sync.baseURL)/teach/jobs?slug=\(slug)") else { return }
            if let (data, _) = try? await URLSession.shared.data(for: URLRequest(url: url, timeoutInterval: 10)),
               let fresh = try? JSONDecoder().decode(Job.self, from: data) {
                job = fresh
            }
        }
    }

    private func post<T: Decodable>(_ path: String, _ body: Data, timeout: TimeInterval) async throws -> T {
        guard let url = URL(string: "\(sync.baseURL)\(path)") else { throw URLError(.badURL) }
        var req = URLRequest(url: url, timeoutInterval: timeout)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = body
        let (data, resp) = try await URLSession.shared.data(for: req)
        guard (resp as? HTTPURLResponse)?.statusCode == 200 else { throw URLError(.badServerResponse) }
        return try JSONDecoder().decode(T.self, from: data)
    }
}

struct TeachView: View {
    @EnvironmentObject var model: AppModel
    @StateObject private var teach: TeachModel
    @State private var draft = ""
    @FocusState private var inputFocused: Bool

    init(sync: Sync, demo: Bool = false) {
        _teach = StateObject(wrappedValue: TeachModel(sync: sync, demo: demo))
    }

    var body: some View {
        VStack(spacing: 0) {
            header
            Divider()
            transcript
            Divider()
            footer
        }
        .frame(maxWidth: 760)
    }

    private var header: some View {
        HStack {
            Text("Teach me something else")
                .font(.system(size: 24, weight: .semibold, design: .serif))
            Spacer()
            Button("Library") { model.backToLibrary() }
                .buttonStyle(.plain).foregroundStyle(.secondary)
        }
        .padding()
    }

    private var transcript: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if teach.messages.isEmpty {
                        Text("Say what you want to learn. A few questions later you will have a book.")
                            .foregroundStyle(.secondary)
                            .padding(.vertical, 8)
                    }
                    ForEach(teach.messages) { m in
                        bubble(m)
                    }
                    if teach.thinking {
                        HStack(spacing: 8) {
                            ProgressView().controlSize(.small)
                            Text("Thinking…").foregroundStyle(.secondary).font(.callout)
                        }
                    }
                    Color.clear.frame(height: 1).id("bottom")
                }
                .padding()
            }
            .onChange(of: teach.messages.count) { _ in
                withAnimation { proxy.scrollTo("bottom", anchor: .bottom) }
            }
        }
    }

    private func bubble(_ m: TeachModel.Message) -> some View {
        HStack {
            if m.role == "learner" { Spacer(minLength: 60) }
            Text(m.text)
                .padding(12)
                .background(m.role == "learner" ? Color.accentColor.opacity(0.16) : Color.gray.opacity(0.14),
                            in: RoundedRectangle(cornerRadius: 12))
                .frame(maxWidth: .infinity, alignment: m.role == "learner" ? .trailing : .leading)
            if m.role != "learner" { Spacer(minLength: 60) }
        }
    }

    @ViewBuilder
    private var footer: some View {
        VStack(spacing: 12) {
            if let err = teach.error {
                Text(err).foregroundStyle(.red).font(.callout)
            }
            if let job = teach.job {
                VStack(spacing: 8) {
                    if !job.done { ProgressView() }
                    Text(job.line).font(.callout)
                        .foregroundStyle(job.stage == "failed" ? .red : .secondary)
                    if job.done && job.stage == "ready" {
                        Button("Open the library") {
                            model.backToLibrary()
                        }
                        .buttonStyle(.borderedProminent)
                    } else if !job.done {
                        Text("This takes a while. You can leave - it keeps going.")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            } else if teach.readyToBuild {
                VStack(spacing: 10) {
                    Text(teach.title).font(.title3.weight(.semibold))
                    Button {
                        Task { await teach.build() }
                    } label: {
                        Text("Build this book").padding(.horizontal, 24).padding(.vertical, 6)
                    }
                    .buttonStyle(.borderedProminent)
                    Button("Keep talking") { teach.brief = "" }
                        .buttonStyle(.plain).foregroundStyle(.secondary).font(.callout)
                }
            } else {
                HStack(spacing: 10) {
                    // Single-line: the multi-line TextField is iOS 16+, and the
                    // iPad this ships to runs 15.8.
                    TextField("I want to learn…", text: $draft)
                        .textFieldStyle(.roundedBorder)
                        .focused($inputFocused)
                        .disabled(teach.thinking)
                        .onSubmit(sendDraft)
                    Button("Send", action: sendDraft)
                        .buttonStyle(.borderedProminent)
                        .disabled(teach.thinking || draft.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
        }
        .padding()
    }

    private func sendDraft() {
        let text = draft
        draft = ""
        Task { await teach.send(text) }
    }
}
