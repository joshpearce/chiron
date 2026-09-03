import Foundation

/// The bookshelf: every subject the server offers, which one is open, and
/// the connection they share. One `BookSession` per subject is kept alive
/// once opened, so switching books later costs nothing.
@MainActor
final class Library: ObservableObject {
    /// One shell for the app, kept across books; the sheet shows it.
    let shell = ShellSession()
    @Published var shellShown = false
    /// The sprite agent's line in, off unless the reader turns it on.
    let agent = AgentLink()
    /// A capture waiting for its question; the capture card shows it.
    @Published var pendingCapture: Capture?
    private var shelfPoller: Task<Void, Never>?
    @Published var subjects: [SubjectInfo] = []
    /// The book the server says was last open, across every client.
    @Published var activeSubjectID: String?
    /// The open book, or nil for the shelf.
    @Published var session: BookSession?
    @Published var teaching = false
    @Published var shelfError: String?
    @Published var loadingShelf = false

    let sync = Sync()
    private var sessions: [String: BookSession] = [:]
    let storage: URL

    init(storage: URL? = nil) {
        self.storage = storage ?? Library.defaultStorage()
        agent.attach(self)
    }

    /// Per-subject caches live in Application Support, not Documents: they
    /// are rebuilt from the server and have no business in a backup.
    static func defaultStorage() -> URL {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        var dir = base.appendingPathComponent("Chiron", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        var values = URLResourceValues()
        values.isExcludedFromBackup = true
        try? dir.setResourceValues(values)
        return dir
    }

    func refresh() async {
        loadingShelf = true
        defer { loadingShelf = false }
        do {
            let shelf = try await sync.subjects()
            subjects = shelf.subjects
            activeSubjectID = shelf.activeID
            shelfError = nil
        } catch {
            shelfError = BookSession.unreachable
        }
        // A primer still authoring turns into a book without the reader
        // asking: the shelf checks back while any card is grey.
        if subjects.contains(where: \.authoring), shelfPoller == nil {
            shelfPoller = Task { [weak self] in
                try? await Task.sleep(nanoseconds: 4_000_000_000)
                guard let self else { return }
                self.shelfPoller = nil
                if self.session == nil { await self.refresh() }
            }
        }
    }

    /// A capture with its question becomes a primer on the shelf.
    func submitCapture(_ c: Capture, prompt: String) async throws -> String {
        var req = CaptureRequest(prompt: prompt)
        let text = c.text.trimmingCharacters(in: .whitespacesAndNewlines)
        if !text.isEmpty { req.text = text }
        if let png = c.imagePNG { req.imagePngB64 = png.base64EncodedString() }
        req.sourceUrl = c.sourceURL
        req.sourceApp = c.sourceApp
        let reply = try await sync.capture(req)
        pendingCapture = nil
        await refresh()
        return reply.subject
    }

    /// A chiron://capture/<id> URL, from the share extension or an intent.
    func receiveCapture(id: String) {
        guard let c = CaptureInbox.take(id) else { return }
        pendingCapture = c
    }

    /// At launch the app reopens on the book last open, on whichever client.
    func openActiveAtLaunch() async {
        await refresh()
        if let id = activeSubjectID, subjects.contains(where: { $0.id == id }) {
            await open(id)
        }
    }

    func open(_ id: String) async {
        let info = subjects.first(where: { $0.id == id })
        let s = sessions[id] ?? BookSession(
            subjectID: id,
            title: info?.title ?? id,
            service: sync, storage: storage)
        if let k = info?.kind { s.kind = k }
        sessions[id] = s
        session = s
        activeSubjectID = id
        await s.open()
    }

    func closeBook() {
        session?.persist()
        session = nil
        Task { await refresh() }
    }
}
