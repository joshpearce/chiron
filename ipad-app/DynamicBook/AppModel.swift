import Foundation
import SwiftUI

/// App state machine: reading -> check -> gate -> next chapter.
/// Everything needed to survive an app relaunch mid-flight persists to
/// Documents: current chapter payload, collected beat responses, book state.
@MainActor
final class AppModel: ObservableObject {
    enum Screen {
        case start
        case reading
        case pretest
        case check
        case gate(Gate, [GradeResult])
        case takingBreak(BreakSuggestion)
    }

    @Published var screen: Screen = .start
    @Published var chapter: ChapterPayload?
    @Published var bookState: BookState?
    @Published var lastResults: [GradeResult] = []
    @Published var errorMessage: String?

    let sync = Sync()

    // Static fallback: the bundled default-path book, used when no server is
    // reachable. Reading + JS-graded beats + reveal-based self-checks survive;
    // adaptivity and free-text grading do not.
    @Published var staticMode = false
    private var staticIndex: Int {
        get { UserDefaults.standard.integer(forKey: "staticIndex") }
        set { UserDefaults.standard.set(newValue, forKey: "staticIndex") }
    }
    private lazy var staticBook: [ChapterPayload] = {
        guard let url = Bundle.main.url(forResource: "default-book", withExtension: "json"),
              let data = try? Data(contentsOf: url),
              let obj = try? JSONDecoder().decode([String: [ChapterPayload]].self, from: data)
        else { return [] }
        return obj["chapters"] ?? []
    }()

    private var beatResponses: [BeatResponse] = []
    private var chapterOpenedAt: Date?
    private let dir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]

    init() {
        restore()
    }

    // MARK: - persistence

    private func restore() {
        if let d = try? Data(contentsOf: dir.appendingPathComponent("chapter.json")),
           let ch = try? JSONDecoder().decode(ChapterPayload.self, from: d) {
            chapter = ch
            screen = .reading
        }
        if let d = try? Data(contentsOf: dir.appendingPathComponent("state.json")),
           let st = try? JSONDecoder().decode(BookState.self, from: d) {
            bookState = st
        }
        if let d = try? Data(contentsOf: dir.appendingPathComponent("beats.json")),
           let br = try? JSONDecoder().decode([BeatResponse].self, from: d) {
            beatResponses = br
        }
    }

    private func persist() {
        if let ch = chapter, let d = try? JSONEncoder().encode(ch) {
            try? d.write(to: dir.appendingPathComponent("chapter.json"))
        }
        if let st = bookState, let d = try? JSONEncoder().encode(st) {
            try? d.write(to: dir.appendingPathComponent("state.json"))
        }
        if let d = try? JSONEncoder().encode(beatResponses) {
            try? d.write(to: dir.appendingPathComponent("beats.json"))
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
        await run(ExchangeRequest(phase: "start", choice: choice))
    }

    func startStatic() {
        staticMode = true
        guard !staticBook.isEmpty else {
            errorMessage = "No bundled book found."
            return
        }
        setChapter(staticBook[min(staticIndex, staticBook.count - 1)])
        openedChapter()
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
        var req = ExchangeRequest(phase: "pretest", unit: chapter?.unit)
        req.pretestResponses = responses
        await run(req, keepReadingOnNil: true)
    }

    func beginCheck() { screen = .check }

    func submitCheck(_ responses: [ItemResponse], override: Bool = false) async {
        if staticMode { advanceStatic(); return }
        var req = ExchangeRequest(unit: chapter?.unit)
        req.checkResponses = responses
        req.beatResponses = beatResponses
        req.override = override
        req.chunkMinutes = chunkMinutes()
        await run(req)
    }

    func skipCheck() async {
        if staticMode { advanceStatic(); return }
        var req = ExchangeRequest(unit: chapter?.unit)
        req.skippedCheck = true
        req.beatResponses = beatResponses
        req.chunkMinutes = chunkMinutes()
        await run(req)
    }

    func catchMeUp() async {
        var req = ExchangeRequest(unit: chapter?.unit)
        req.catchMeUp = true
        req.beatResponses = beatResponses
        await run(req)
    }

    func continueAfterGate(override: Bool) async {
        guard case .gate = screen else { return }
        if override {
            var req = ExchangeRequest(unit: chapter?.unit)
            req.override = true
            await run(req)
        } else {
            // remediation chapter was already delivered with the gate response
            screen = .reading
        }
    }

    func breakFinished(minutes: Double) async {
        var req = ExchangeRequest(unit: chapter?.unit)
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
