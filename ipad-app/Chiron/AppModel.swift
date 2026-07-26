import Foundation
import SwiftUI

/// App state machine: library -> subject -> reading -> check -> gate -> next chapter.
/// Everything needed to survive an app relaunch mid-flight persists to
/// Documents, keyed by subject: current chapter payload, collected beat
/// responses, book state.
@MainActor
final class AppModel: ObservableObject {
    enum Screen {
        case menu
        case start
        case reading
        case pretest
        case check
        case gate(Gate, [GradeResult])
        case takingBreak(BreakSuggestion)
        case teach
    }

    @Published var screen: Screen = .menu
    @Published var subjects: [SubjectInfo] = []
    @Published var chapter: ChapterPayload?
    @Published var bookState: BookState?
    @Published var lastResults: [GradeResult] = []
    @Published var errorMessage: String?

    let sync = Sync()

    private(set) var subjectID = "ai"
    var currentSubjectTitle: String {
        subjects.first(where: { $0.id == subjectID })?.title ?? "How AI Works"
    }
    var isMenu: Bool { if case .menu = screen { return true }; return false }
    var isTeaching: Bool { if case .teach = screen { return true }; return false }

    // Static fallback: the bundled default-path book (subject "ai"), used when
    // no server is reachable. Reading + JS-graded beats + reveal-based
    // self-checks survive; adaptivity and free-text grading do not.
    @Published var staticMode = false
    private var staticIndex: Int {
        get { UserDefaults.standard.integer(forKey: "staticIndex-\(subjectID)") }
        set { UserDefaults.standard.set(newValue, forKey: "staticIndex-\(subjectID)") }
    }
    private lazy var staticBook: [ChapterPayload] = {
        guard let url = Bundle.main.url(forResource: "default-book", withExtension: "json") else {
            staticLoadError = "default-book.json not in bundle"
            return []
        }
        guard let data = try? Data(contentsOf: url) else {
            staticLoadError = "default-book.json unreadable"
            return []
        }
        do {
            let obj = try JSONDecoder().decode([String: [ChapterPayload]].self, from: data)
            return obj["chapters"] ?? []
        } catch {
            // A silent empty fallback is the worst outcome here: the built-in
            // book is the last resort when everything else has failed, so say
            // exactly why it did not load.
            staticLoadError = "default-book.json failed to decode: \(error)"
            return []
        }
    }()

    private(set) var staticLoadError: String?

    private var beatResponses: [BeatResponse] = []
    private var chapterOpenedAt: Date?
    private let dir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]

    // MARK: - library

    func refreshSubjects() async {
        var req: URLRequest?
        if let url = URL(string: "\(sync.baseURL)/subjects") {
            req = URLRequest(url: url, timeoutInterval: 3)
            Credentials.authorize(&req!)
        }
        if let r = req,
           let (data, resp) = try? await URLSession.shared.data(for: r),
           (resp as? HTTPURLResponse)?.statusCode == 200,
           let obj = try? JSONDecoder().decode([String: [SubjectInfo]].self, from: data) {
            subjects = obj["subjects"] ?? []
        } else if subjects.isEmpty {
            // offline: the built-in subject is always available
            subjects = [SubjectInfo(id: "ai", title: "How AI Works",
                                    unitsTotal: 11, unitsCleared: nil,
                                    currentUnit: nil, debt: nil)]
        }
    }

    func openSubject(_ id: String) async {
        subjectID = id
        staticMode = false
        restore()
        if chapter != nil {
            screen = .reading
        } else {
            screen = .start
        }
    }

    func backToLibrary() {
        persist()
        chapter = nil
        bookState = nil
        errorMessage = nil
        staticMode = false
        screen = .menu
        Task { await refreshSubjects() }
    }

    // MARK: - persistence (per subject)

    private func file(_ name: String) -> URL {
        dir.appendingPathComponent("\(name)-\(subjectID).json")
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
    }

    private func persist() {
        if let ch = chapter, let d = try? JSONEncoder().encode(ch) {
            try? d.write(to: file("chapter"))
        }
        if let st = bookState, let d = try? JSONEncoder().encode(st) {
            try? d.write(to: file("state"))
        }
        if let d = try? JSONEncoder().encode(beatResponses) {
            try? d.write(to: file("beats"))
        }
    }

    // MARK: - beat responses from the webview bridge

    func recordBeat(_ r: BeatResponse) {
        beatResponses.removeAll { $0.beatId == r.beatId }
        beatResponses.append(r)
        persist()
    }

    // MARK: - flow

    func start(choice: String? = nil) async {
        await run(ExchangeRequest(subject: subjectID, phase: "start", choice: choice))
    }

    func startStatic() {
        staticMode = true
        guard !staticBook.isEmpty else {
            errorMessage = staticLoadError ?? "No bundled book found."
            return
        }
        setChapter(staticBook[min(staticIndex, staticBook.count - 1)])
        openedChapter()
    }

    /// Debug affordance: the bundled chapters, for inspection harnesses.
    var staticChapters: [ChapterPayload] { staticBook }

    /// Debug affordances: show a specific chapter or screen without going
    /// through an exchange, so any screen can be inspected in isolation.
    func loadChapterForInspection(_ ch: ChapterPayload) { setChapter(ch) }
    func showReadingForInspection() { screen = .reading }
    func showPretestForInspection() { screen = .pretest }
    func showBreakForInspection() {
        screen = .takingBreak(BreakSuggestion(
            minutes: 5, kind: "short",
            note: "Chunk done. Five minutes, eyes off screens."))
    }

    private func advanceStatic() {
        staticIndex = min(staticIndex + 1, staticBook.count - 1)
        setChapter(staticBook[staticIndex])
        openedChapter()
        persist()
    }

    func openedChapter() {
        chapterOpenedAt = Date()
        if chapter?.pretest.isEmpty == false && !pretestDone {
            screen = .pretest
        } else {
            screen = .reading
        }
    }

    var pretestDone = false

    func submitPretest(_ responses: [ItemResponse]) async {
        pretestDone = true
        if staticMode { screen = .reading; return }
        // Pretest grades fold into the boundary exchange offline; if connected,
        // send now so the planner can compress the CURRENT chapter.
        guard sync.connected else { screen = .reading; return }
        var req = ExchangeRequest(subject: subjectID, phase: "pretest", unit: chapter?.unit)
        req.pretestResponses = responses
        await run(req, keepReadingOnNil: true)
    }

    func beginCheck() { screen = .check }

    func submitCheck(_ responses: [ItemResponse], override: Bool = false) async {
        if staticMode { advanceStatic(); return }
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.checkResponses = responses
        req.beatResponses = beatResponses
        req.override = override
        req.chunkMinutes = chunkMinutes()
        await run(req)
    }

    func skipCheck() async {
        if staticMode { advanceStatic(); return }
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.skippedCheck = true
        req.beatResponses = beatResponses
        req.chunkMinutes = chunkMinutes()
        await run(req)
    }

    func catchMeUp() async {
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.catchMeUp = true
        req.beatResponses = beatResponses
        await run(req)
    }

    func continueAfterGate(override: Bool) async {
        guard case .gate = screen else { return }
        if override {
            var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
            req.override = true
            await run(req)
        } else {
            // remediation chapter was already delivered with the gate response
            screen = .reading
        }
    }

    func breakFinished(minutes: Double) async {
        var req = ExchangeRequest(subject: subjectID, unit: chapter?.unit)
        req.breakMinutes = minutes
        await run(req, keepReadingOnNil: true)
    }

    private func chunkMinutes() -> Double? {
        guard let t = chapterOpenedAt else { return nil }
        return Date().timeIntervalSince(t) / 60
    }

    private func run(_ req: ExchangeRequest, keepReadingOnNil: Bool = false) async {
        errorMessage = nil
        do {
            let resp = try await sync.exchange(req)
            bookState = resp.state
            lastResults = resp.results
            if !req.checkResponses.isEmpty, let gate = resp.gate {
                // Show gate first; the delivered chapter (next or remediation)
                // is already stored for when the learner proceeds.
                if let ch = resp.chapter { setChapter(ch) }
                screen = .gate(gate, resp.results)
            } else if let ch = resp.chapter {
                setChapter(ch)
                openedChapter()
            } else if let bs = resp.breakSuggestion {
                screen = .takingBreak(bs)
            } else if !keepReadingOnNil {
                screen = .start
            } else {
                screen = .reading
            }
            persist()
        } catch {
            errorMessage = "Exchange failed (\(sync.transport)): \(error.localizedDescription)"
            if chapter != nil { screen = .reading }
        }
    }

    private func setChapter(_ ch: ChapterPayload) {
        chapter = ch
        beatResponses = []
        pretestDone = false
        chapterOpenedAt = Date()
    }
}
