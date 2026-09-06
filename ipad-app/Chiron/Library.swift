import Foundation

/// The bookshelf: every subject the server offers, which one is open, and
/// the connection they share. One `BookSession` per subject is kept alive
/// once opened, so switching books later costs nothing.
@MainActor
final class Library: ObservableObject {
    /// One shell for the app, kept across books; the sheet shows it.
    let shell = ShellSession()
    @Published var shellShown = false
    /// The "set up another device" code, on screen.
    @Published var deviceSetupShown = false
    /// The shelves of the library, and the one open (a path of one id).
    @Published var shelves: [ShelfInfo] = []
    @Published var shelfPath: [String] = []
    #if DEBUG
    /// The last URL the app was opened with, for the harness.
    var lastOpenedURL: String?
    #endif
    /// The sprite agent's line in, off unless the reader turns it on.
    let agent = AgentLink()
    /// A capture waiting for its question; the capture card shows it.
    @Published var pendingCapture: Capture?
    /// A summary or a description, answered in the capture card.
    @Published var captureAnswer: String?
    /// A draft being planned in conversation; the planning card shows it.
    @Published var planning: PlanState?
    @Published var planBusy = false
    @Published var planError: String?
    private var shelfPoller: Task<Void, Never>?
    /// How often the shelf checks back while a primer is being written.
    var shelfPollInterval: TimeInterval = 4
    /// The primer the reader just asked for: it opens on its own once the
    /// server has written it, as long as the reader is still on the shelf.
    private var awaitedPrimer: String?
    @Published var subjects: [SubjectInfo] = []
    /// The book the server says was last open, across every client.
    @Published var activeSubjectID: String?
    /// The open book, or nil for the shelf.
    @Published var session: BookSession?
    @Published var teaching = false
    @Published var shelfError: String?
    @Published var loadingShelf = false

    let sync = Sync()
    /// What the shelf and its books talk to: the connection, or a stand-in
    /// under test.
    let service: ChironService
    private var sessions: [String: BookSession] = [:]
    let storage: URL

    init(storage: URL? = nil, service: ChironService? = nil) {
        self.storage = storage ?? Library.defaultStorage()
        self.service = service ?? sync
        agent.attach(self)
        loadShelfCache()
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
            let shelf = try await service.subjects()
            subjects = shelf.subjects
            shelves = shelf.shelves ?? []
            activeSubjectID = shelf.activeID
            shelfError = nil
            // A shelf deleted elsewhere closes here.
            shelfPath.removeAll { id in !shelves.contains { $0.id == id } }
            saveShelfCache(shelf)
            // The server is back: whatever was marked up while it was away goes up.
            await session?.pushAnnotations()
        } catch {
            shelfError = subjects.isEmpty ? BookSession.unreachable : Library.offline
        }
        if let id = awaitedPrimer, let primer = subjects.first(where: { $0.id == id }), !primer.authoring {
            awaitedPrimer = nil
            if !primer.failed && session == nil { await open(id) }
        }
        // A primer still authoring turns into a book without the reader
        // asking: the shelf checks back while any card is grey.
        if subjects.contains(where: \.authoring), shelfPoller == nil {
            shelfPoller = Task { [weak self] in
                guard let self else { return }
                try? await Task.sleep(nanoseconds: UInt64(self.shelfPollInterval * 1_000_000_000))
                self.shelfPoller = nil
                if self.session == nil || self.awaitedPrimer != nil { await self.refresh() }
            }
        }
    }

    /// A capture with its question and its scale: a summary or a
    /// description comes back into the card; a primer or a book opens its
    /// planning conversation.
    @discardableResult
    func submitCapture(_ c: Capture, prompt: String, scale: CaptureScale = .primer) async throws -> CaptureResponse {
        var req = CaptureRequest(prompt: prompt, scale: scale)
        let text = c.text.trimmingCharacters(in: .whitespacesAndNewlines)
        if !text.isEmpty { req.text = text }
        if let png = c.imagePNG { req.imagePngB64 = png.base64EncodedString() }
        req.sourceUrl = c.sourceURL
        req.sourceApp = c.sourceApp
        let reply = try await service.capture(req)
        if scale.immediate {
            captureAnswer = reply.answerMd ?? ""
            return reply
        }
        guard let id = reply.subject else { throw URLError(.badServerResponse) }
        var plan: [PlanMessage] = []
        if let line = reply.replyMd, !line.isEmpty { plan.append(PlanMessage(role: "tutor", text: line)) }
        let state = PlanState(id: id, title: reply.title ?? prompt, scale: scale.rawValue, status: reply.status ?? "planning",
                              error: nil, prompt: prompt,
                              source: PrimerSource(text: text.isEmpty ? nil : text, url: c.sourceURL, app: c.sourceApp),
                              brief: reply.brief, done: reply.done ?? false, plan: plan, book: nil)
        pendingCapture = nil
        captureAnswer = nil
        await refresh()
        // One sheet gives way to the next; presenting both in the same
        // beat leaves the second unshown.
        try? await Task.sleep(nanoseconds: 350_000_000)
        planning = state
        return reply
    }

    /// A draft on the shelf reopens its conversation.
    func openDraft(_ id: String) async {
        do {
            planning = try await service.plan(subject: id)
            planError = nil
        } catch {
            shelfError = BookSession.unreachable
        }
    }

    /// The reader's next line to the tutor.
    func planReply(_ text: String) async {
        guard var state = planning else { return }
        let line = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !line.isEmpty else { return }
        state.plan.append(PlanMessage(role: "learner", text: line))
        planning = state
        planBusy = true
        planError = nil
        defer { planBusy = false }
        do {
            let turn = try await service.planTurn(subject: state.id, text: line)
            guard var now = planning, now.id == state.id else { return }
            if let reply = turn.replyMd { now.plan.append(PlanMessage(role: "tutor", text: reply)) }
            if turn.done == true {
                now.done = true
                now.brief = turn.brief
                if let t = turn.title, !t.isEmpty { now.title = t }
            }
            planning = now
        } catch {
            // The line stays in the transcript; the next send carries on.
            planError = "The tutor could not be reached. Send again."
        }
    }

    /// Build what the plan has: the reader lands on the shelf and the
    /// primer, or the book, opens when it is done.
    func buildDraft() async {
        guard let state = planning else { return }
        planBusy = true
        defer { planBusy = false }
        do {
            let built = try await service.build(subject: state.id)
            awaitedPrimer = built.book ?? built.subject
            planning = nil
            planError = nil
            session?.persist()
            session = nil
            await refresh()
        } catch {
            planError = "The server could not start the build. Try again."
        }
    }

    /// The draft is not wanted after all.
    func discardDraft() async {
        guard let state = planning else { return }
        do {
            try await service.discard(subject: state.id)
            planning = nil
            planError = nil
            await refresh()
        } catch {
            planError = "The server could not discard it. Try again."
        }
    }

    static let offline = "The server is out of reach; this is the library as it was last seen."

    // The library as last seen, kept on disk so it shows without the server.

    private var shelfCacheURL: URL { storage.appendingPathComponent("library.json") }

    private func saveShelfCache(_ shelf: SubjectsResponse) {
        try? FileManager.default.createDirectory(at: storage, withIntermediateDirectories: true)
        if let d = try? JSONEncoder().encode(shelf) { try? d.write(to: shelfCacheURL) }
    }

    private func loadShelfCache() {
        guard let d = try? Data(contentsOf: shelfCacheURL),
              let shelf = try? JSONDecoder().decode(SubjectsResponse.self, from: d) else { return }
        subjects = shelf.subjects
        shelves = shelf.shelves ?? []
        activeSubjectID = shelf.activeID
    }

    // The library's shape: what is on no shelf, and what is on one.

    var unfiled: [SubjectInfo] { subjects.filter { $0.shelf == nil } }

    func subjects(on shelf: String) -> [SubjectInfo] { subjects.filter { $0.shelf == shelf } }

    func shelf(_ id: String) -> ShelfInfo? { shelves.first { $0.id == id } }

    func createShelf(named name: String) async {
        let trimmed = name.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { return }
        do {
            _ = try await service.createShelf(name: trimmed)
            shelfError = nil
        } catch {
            shelfError = "The server could not make the shelf."
        }
        await refresh()
    }

    func renameShelf(_ id: String, to name: String) async {
        let trimmed = name.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { return }
        do {
            _ = try await service.renameShelf(id, name: trimmed)
            shelfError = nil
        } catch {
            shelfError = "The server could not rename the shelf."
        }
        await refresh()
    }

    /// Deleting a shelf returns what it held to the library.
    func deleteShelf(_ id: String) async {
        do {
            try await service.deleteShelf(id)
            shelfError = nil
        } catch {
            shelfError = "The server could not delete the shelf."
        }
        await refresh()
    }

    /// File a subject on a shelf, or, with nil, back in the library.
    func move(_ subject: String, to shelf: String?) async {
        do {
            try await service.move(subject: subject, toShelf: shelf)
            shelfError = nil
        } catch {
            shelfError = "The server could not move it."
        }
        await refresh()
    }

    /// A server from another device's code: saved, selected, probed, and
    /// this device's ssh key enrolled when the link carries the shared key,
    /// as the settings sheet does on save.
    func adopt(_ link: ServerLink) async {
        let server = sync.servers.adopt(link)
        sync.baseURL = server.url
        await sync.probe()
        await refresh()
        if link.key != nil {
            _ = try? await sync.enrolDeviceKey(name: DeviceKeySection.deviceName)
        }
    }

    /// A chiron://capture/<id> URL, from the share extension or an intent.
    func receiveCapture(id: String) {
        guard let c = CaptureInbox.take(id) else { return }
        pendingCapture = c
    }

    /// At launch the app lands on the shelf; the book last open, on
    /// whichever client, is the card marked "Open now".
    func launch() async {
        await refresh()
    }

    func open(_ id: String) async {
        let info = subjects.first(where: { $0.id == id })
        let s = sessions[id] ?? BookSession(
            subjectID: id,
            title: info?.title ?? id,
            service: service, storage: storage)
        if let k = info?.kind { s.kind = k }
        // A passage sent on from inside the book: the card opens over it,
        // naming the book as where the words came from.
        s.onCapture = { [weak self] text in
            self?.captureAnswer = nil
            self?.pendingCapture = Capture(text: text, sourceApp: s.title)
        }
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
