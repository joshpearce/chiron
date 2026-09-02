import Foundation

/// One book, open: its chapter, the learner's position in it, and the state
/// machine from placement through checks, results, and the next chapter.
/// A `Library` holds one per subject; the UI shows one at a time.
///
/// Everything needed to survive a relaunch mid-flight persists under
/// Application Support, keyed by subject: the current chapter, collected beat
/// responses, the cached book state, and a pending authoring wait.
@MainActor
final class BookSession: ObservableObject {
    enum Screen {
        case placement
        case series
        case reading
        case pretest
        case check
        case results(ResultsDoc, Gate)
        case authoring(String)
        case takingBreak(BreakSuggestion)
        case error(String)
        /// Nothing to show yet: opening, or the server is away and there is
        /// no cached chapter.
        case empty
    }

    /// What the learner is waiting on, in the tablet's words.
    enum Wait: String {
        case opening = "Opening the book."
        case grading = "Grading your answers."
        case resetting = "Starting over."
    }

    static let unreachable = "The server can't be reached."

    @Published var screen: Screen = .empty
    @Published var wait: Wait?
    @Published var chapter: ChapterPayload?
    @Published var bookState: BookState?
    @Published var lastResults: [GradeResult] = []
    /// A non-fatal problem worth a line on the current screen.
    @Published var errorMessage: String?
    /// Reader chrome (top strip, bottom bar) hidden by a tap on the page.
    @Published var chromeHidden = false
    /// The contents beside the reader (regular width) or over it (compact).
    @Published var contentsShown = false

    let subjectID: String
    let title: String
    let service: ChironService
    /// Seconds between /chapter polls while the server writes.
    var pollInterval: TimeInterval = 2

    private var beatResponses: [BeatResponse] = []
    private var chapterOpenedAt: Date?
    private var pretestDone = false
    /// Where the reader left each chapter: scroll offset in CSS pixels.
    private var positions: [String: Double] = [:]
    /// Held from a graded exchange until the learner leaves the results.
    private var pendingBreak: BreakSuggestion?
    private var pendingAuthoring: String?
    /// What "Try again" does after a failure.
    private var retryAction: (() async -> Void)?
    private let dir: URL

    init(subjectID: String, title: String, service: ChironService, storage: URL) {
        self.subjectID = subjectID
        self.title = title
        self.service = service
        dir = storage.appendingPathComponent(subjectID, isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
    }

    var busy: Bool { wait != nil }

    // MARK: - opening

    /// Open the book: restore what is cached, reconcile with the server, and
    /// land on whatever the book is waiting for. A book that has never been
    /// opened starts itself.
    func open() async {
        restore()
        wait = .opening
        do {
            try await reconcileState()
        } catch {
            wait = nil
            // Reading detached is the one thing the cache is for.
            if let ch = chapter, !ch.isCalibration {
                show(ch)
            } else {
                fail(Self.unreachable) { [weak self] in await self?.open() }
            }
            return
        }
        // The wait clears before anything below runs an exchange: an
        // exchange refuses to start while one is marked in flight.
        wait = nil
        if let unit = pendingAuthoring {
            await awaitChapter(unit)
        } else if let ch = chapter {
            show(ch)
        } else {
            await start()
        }
    }

    /// Pull the authoritative state from the server.
    ///
    /// The persisted copy is a cache for reading offline, not the truth. Left
    /// unrefreshed it drifts: a spine restored from disk showed units as passed
    /// that the server had no record of, because the only thing that ever
    /// updated it was the response to an exchange.
    func refreshState() async {
        try? await reconcileState()
    }

    private func reconcileState() async throws {
        let st = try await service.state(subject: subjectID)
        bookState = st
        // A cached chapter outlives the state it was generated from. After the
        // server's progress is reset or diverges, the app would otherwise keep
        // serving a chapter written for a learner who no longer exists - one
        // authored as "already fluent" for someone starting from scratch. If
        // the server does not have this unit in progress, the cached chapter is
        // stale and goes.
        if let ch = chapter, ch.unit != "catchup" {
            let known = st.spine.first { $0.unit == ch.unit }
            let inProgress = known?.status == "active" || known?.inFringe == true
            if !inProgress {
                chapter = nil
                beatResponses = []
            }
        }
        persist()
    }

    func start(choice: String? = nil) async {
        await run(ExchangeRequest(subject: subjectID, phase: "start", choice: choice), wait: .opening)
    }

    /// Discard all progress for this subject, server-side and locally, and
    /// open the book again from the top.
    func startOver() async {
        errorMessage = nil
        wait = .resetting
        do {
            bookState = try await service.reset(subject: subjectID)
        } catch {
            wait = nil
            errorMessage = "Could not reach the server to start over."
            return
        }
        chapter = nil
        beatResponses = []
        pendingBreak = nil
        pendingAuthoring = nil
        persist()
        wait = nil
        await start()
    }

    // MARK: - the flow

    /// Where a chapter lands the learner: the screener is a placement
    /// question, a calibration series is measurement, a pretest comes before
    /// the prose, and otherwise there is a chapter to read.
    private func show(_ ch: ChapterPayload) {
        chromeHidden = false
        if ch.screener != nil {
            screen = .placement
        } else if ch.isCalibration {
            screen = .series
        } else if !ch.pretest.isEmpty && !pretestDone {
            screen = .pretest
        } else {
            screen = .reading
        }
    }

    /// Answer the placement screener: one tap, no gate, the series follows.
    func place(level: Int) async {
        guard let screener = chapter?.screener else { return }
        await submitCheck([ItemResponse(itemId: screener.id, response: nil,
                                        selectedIndex: level - 1, confidence: 3)])
    }

    func submitPretest(_ responses: [ItemResponse]) async {
        pretestDone = true
        var req = ExchangeRequest(subject: subjectID, phase: "pretest", unit: chapter?.unit)
        req.pretestResponses = responses
        await run(req, wait: .grading, keepReadingOnNil: true)
    }

    func beginCheck() { screen = .check }

    func leaveCheck() { screen = .reading }

    func submitCheck(_ responses: [ItemResponse]) async {
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.checkResponses = responses
        req.beatResponses = beatResponses
        req.chunkMinutes = chunkMinutes()
        await run(req, wait: .grading)
    }

    func skipCheck() async {
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.skippedCheck = true
        req.beatResponses = beatResponses
        req.chunkMinutes = chunkMinutes()
        await run(req, wait: .opening)
    }

    func catchMeUp() async {
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.catchMeUp = true
        req.beatResponses = beatResponses
        await run(req, wait: .opening)
    }

    /// Leave the results: a suggested break first, then the chapter the
    /// server delivered or is still writing.
    func proceed() async {
        if let b = pendingBreak {
            pendingBreak = nil
            screen = .takingBreak(b)
            return
        }
        if let unit = pendingAuthoring {
            await awaitChapter(unit)
        } else if let ch = chapter {
            show(ch)
        } else {
            await start()
        }
    }

    /// Below the gate, take the material anyway: the debt is recorded and the
    /// next chapter follows.
    func override() async {
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.override = true
        pendingAuthoring = nil
        await run(req, wait: .opening)
    }

    func breakFinished(minutes: Double) async {
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.breakMinutes = minutes
        // Reporting the break delivers nothing itself; whatever was pending
        // before the break (a chapter being written) follows.
        await run(req, wait: .opening, keepReadingOnNil: true)
    }

    /// The "Try again" on the error screen.
    func retry() async {
        guard let action = retryAction else { await open(); return }
        await action()
    }

    // MARK: - exchange

    private func run(_ req: ExchangeRequest, wait kind: Wait, keepReadingOnNil: Bool = false) async {
        // One exchange at a time. The buttons that trigger exchanges stay on
        // screen while one is in flight, and a second tap would grade the same
        // check twice - real model cost and duplicate learner-state events.
        guard !busy else { return }
        errorMessage = nil
        wait = kind
        defer { wait = nil }
        let resp: ExchangeResponse
        do {
            resp = try await service.exchange(req)
        } catch {
            fail(Self.unreachable) { [weak self] in
                await self?.run(req, wait: kind, keepReadingOnNil: keepReadingOnNil)
            }
            return
        }
        bookState = resp.state
        lastResults = resp.results
        if let ch = resp.chapter { setChapter(ch) }
        if let unit = resp.authoring { pendingAuthoring = unit }
        if let b = resp.breakSuggestion { pendingBreak = b }
        persist()

        if let doc = resp.resultsDoc, let gate = resp.gate {
            screen = .results(doc, gate)
        } else if resp.chapter != nil, let ch = chapter {
            show(ch)
        } else if let b = pendingBreak, !keepReadingOnNil {
            pendingBreak = nil
            screen = .takingBreak(b)
        } else if let unit = pendingAuthoring {
            await awaitChapter(unit)
        } else if keepReadingOnNil, chapter != nil {
            screen = .reading
        } else if let ch = chapter {
            show(ch)
        } else {
            fail("The server sent no chapter.") { [weak self] in await self?.start() }
        }
    }

    /// Wait for the server to finish writing `unit`, polling /chapter.
    private func awaitChapter(_ unit: String) async {
        screen = .authoring(unit)
        while true {
            let status: ChapterStatus
            do {
                status = try await service.chapter(subject: subjectID)
            } catch {
                fail(Self.unreachable) { [weak self] in await self?.awaitChapter(unit) }
                return
            }
            if status.authoring {
                try? await Task.sleep(nanoseconds: UInt64(pollInterval * 1_000_000_000))
                if Task.isCancelled { return }
                continue
            }
            pendingAuthoring = nil
            if let err = status.authoringError, !err.isEmpty {
                persist()
                fail("Chapter authoring failed:\n\(err)") { [weak self] in await self?.start() }
                return
            }
            if let ch = status.chapter, ch.unit == unit {
                setChapter(ch)
                persist()
                show(ch)
            } else {
                // Not writing and not the chapter we were promised (a server
                // restarted mid-build): ask for one from the top.
                persist()
                await start()
            }
            return
        }
    }

    private func fail(_ message: String, retry: @escaping () async -> Void) {
        retryAction = retry
        screen = .error(message)
    }

    private func setChapter(_ ch: ChapterPayload) {
        chapter = ch
        beatResponses = []
        pretestDone = false
        chapterOpenedAt = Date()
    }

    private func chunkMinutes() -> Double? {
        guard let t = chapterOpenedAt else { return nil }
        return Date().timeIntervalSince(t) / 60
    }

    // MARK: - the reader's bridge

    func recordBeat(_ r: BeatResponse) {
        beatResponses.removeAll { $0.beatId == r.beatId }
        beatResponses.append(r)
        persist()
    }

    /// The reader reports where it is as the page scrolls; the position is
    /// kept per chapter so a book reopens where it was left.
    func recordPosition(unit: String, offset: Double) {
        positions[unit] = offset
    }

    func position(for unit: String) -> Double {
        positions[unit] ?? 0
    }

    func toggleChrome() {
        chromeHidden.toggle()
    }

    // MARK: - persistence

    private struct Position: Codable {
        var pendingAuthoring: String?
        var pretestDone: Bool
        var positions: [String: Double]?
    }

    private func file(_ name: String) -> URL {
        dir.appendingPathComponent("\(name).json")
    }

    private func restore() {
        chapter = nil
        bookState = nil
        beatResponses = []
        if let d = try? Data(contentsOf: file("chapter")),
           let ch = try? JSONDecoder().decode(ChapterPayload.self, from: d) {
            chapter = ch
        }
        if let d = try? Data(contentsOf: file("state")),
           let st = try? JSONDecoder().decode(BookState.self, from: d) {
            bookState = st
        }
        if let d = try? Data(contentsOf: file("beats")),
           let br = try? JSONDecoder().decode([BeatResponse].self, from: d) {
            beatResponses = br
        }
        if let d = try? Data(contentsOf: file("position")),
           let p = try? JSONDecoder().decode(Position.self, from: d) {
            pendingAuthoring = p.pendingAuthoring
            pretestDone = p.pretestDone
            positions = p.positions ?? [:]
        }
    }

    func persist() {
        if let ch = chapter, let d = try? JSONEncoder().encode(ch) {
            try? d.write(to: file("chapter"))
        } else {
            try? FileManager.default.removeItem(at: file("chapter"))
        }
        if let st = bookState, let d = try? JSONEncoder().encode(st) {
            try? d.write(to: file("state"))
        }
        if let d = try? JSONEncoder().encode(beatResponses) {
            try? d.write(to: file("beats"))
        }
        if let d = try? JSONEncoder().encode(Position(pendingAuthoring: pendingAuthoring, pretestDone: pretestDone, positions: positions)) {
            try? d.write(to: file("position"))
        }
    }
}
