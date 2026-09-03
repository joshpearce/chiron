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
    }

    /// At launch the app reopens on the book last open, on whichever client.
    func openActiveAtLaunch() async {
        await refresh()
        if let id = activeSubjectID, subjects.contains(where: { $0.id == id }) {
            await open(id)
        }
    }

    func open(_ id: String) async {
        let s = sessions[id] ?? BookSession(
            subjectID: id,
            title: subjects.first(where: { $0.id == id })?.title ?? id,
            service: sync, storage: storage)
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
