import XCTest
@testable import Chiron

/// A server made of fixtures. Each hook is what the next call returns; the
/// requests made are kept for assertions.
final class FakeService: ChironService {
    var onSubjects: () throws -> SubjectsResponse = { throw URLError(.cannotConnectToHost) }
    var onState: (String) throws -> BookState = { _ in throw URLError(.cannotConnectToHost) }
    var onChapter: (String) throws -> ChapterStatus = { _ in throw URLError(.cannotConnectToHost) }
    var onExchange: (ExchangeRequest) throws -> ExchangeResponse = { _ in throw URLError(.cannotConnectToHost) }
    var onReset: (String) throws -> BookState = { _ in throw URLError(.cannotConnectToHost) }
    var exchanges: [ExchangeRequest] = []
    var chapterPolls = 0

    func subjects() async throws -> SubjectsResponse { try onSubjects() }
    func state(subject: String) async throws -> BookState { try onState(subject) }
    func chapter(subject: String) async throws -> ChapterStatus {
        chapterPolls += 1
        return try onChapter(subject)
    }
    func exchange(_ request: ExchangeRequest) async throws -> ExchangeResponse {
        exchanges.append(request)
        return try onExchange(request)
    }
    func reset(subject: String) async throws -> BookState { try onReset(subject) }
}

@MainActor
final class BookSessionTests: XCTestCase {
    private var fake: FakeService!
    private var storage: URL!

    override func setUp() {
        fake = FakeService()
        storage = FileManager.default.temporaryDirectory
            .appendingPathComponent("chiron-tests-\(UUID().uuidString)", isDirectory: true)
    }

    override func tearDown() {
        try? FileManager.default.removeItem(at: storage)
    }

    private func fixture<T: Decodable>(_ type: T.Type, _ name: String) -> T {
        let url = Bundle(for: FixtureDecodingTests.self)
            .url(forResource: "fixtures", withExtension: nil)!
            .appendingPathComponent("\(name).json")
        return try! JSONDecoder().decode(type, from: try! Data(contentsOf: url))
    }

    private func session() -> BookSession {
        let s = BookSession(subjectID: "ai", title: "How AI Works", service: fake, storage: storage)
        s.pollInterval = 0
        return s
    }

    /// The fake behaves like a fresh server that authors instantly: start
    /// delivers the screener, the screener answer delivers the series, a
    /// graded series returns results with u1 authoring, and /chapter has u1.
    private func scriptFreshBook() {
        fake.onState = { [unowned self] _ in self.fixture(BookState.self, "state") }
        fake.onChapter = { [unowned self] _ in self.fixture(ChapterStatus.self, "chapter") }
        fake.onExchange = { [unowned self] req in
            if req.phase == "start" { return self.fixture(ExchangeResponse.self, "exchange-start") }
            if req.checkResponses.count == 1, req.checkResponses[0].itemId == "u0-s1" {
                return self.fixture(ExchangeResponse.self, "exchange-screener")
            }
            return self.fixture(ExchangeResponse.self, "exchange-series")
        }
    }

    func testFreshBookRunsPlacementSeriesResultsThenTheChapter() async {
        scriptFreshBook()
        let s = session()

        await s.open()
        guard case .placement = s.screen else { return XCTFail("after open: \(s.screen)") }
        XCTAssertEqual(fake.exchanges.map(\.phase), ["start"])

        await s.place(level: 2)
        guard case .series = s.screen else { return XCTFail("after placement: \(s.screen)") }
        XCTAssertEqual(fake.exchanges.last?.checkResponses.first?.selectedIndex, 1, "level 2 is index 1")
        XCTAssertEqual(s.chapter?.check.count, 11)

        let answers = s.chapter!.check.map {
            ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
        }
        await s.submitCheck(answers)
        guard case .results(let doc, let gate) = s.screen else { return XCTFail("after series: \(s.screen)") }
        XCTAssertTrue(doc.isCalibration)
        XCTAssertTrue(gate.passed)
        XCTAssertEqual(fake.exchanges.last?.async, true, "graded checks never wait for authoring")
        XCTAssertNil(s.wait)

        await s.proceed()
        guard case .reading = s.screen else { return XCTFail("after results: \(s.screen)") }
        XCTAssertEqual(s.chapter?.unit, "u1")
        XCTAssertGreaterThanOrEqual(fake.chapterPolls, 1)
    }

    func testAuthoringWaitPollsUntilTheChapterArrives() async {
        scriptFreshBook()
        var polls = 0
        fake.onChapter = { [unowned self] _ in
            polls += 1
            if polls < 3 {
                return ChapterStatus(chapter: nil, authoring: true, authoringError: "")
            }
            return self.fixture(ChapterStatus.self, "chapter")
        }
        let s = session()
        await s.open()
        await s.place(level: 3)
        await s.submitCheck(s.chapter!.check.map {
            ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
        })
        await s.proceed()
        guard case .reading = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertEqual(polls, 3)
    }

    func testAuthoringFailureIsAnErrorWithAWayOut() async {
        scriptFreshBook()
        fake.onChapter = { _ in ChapterStatus(chapter: nil, authoring: false, authoringError: "model timed out") }
        let s = session()
        await s.open()
        await s.place(level: 3)
        await s.submitCheck(s.chapter!.check.map {
            ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
        })
        await s.proceed()
        guard case .error(let message) = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertTrue(message.contains("model timed out"))

        // Try again asks the server for a chapter from the top.
        fake.onExchange = { [unowned self] _ in self.fixture(ExchangeResponse.self, "exchange-start") }
        await s.retry()
        XCTAssertEqual(fake.exchanges.last?.phase, "start")
        guard case .placement = s.screen else { return XCTFail("\(s.screen)") }
    }

    /// An exchange that delivers the persisted u1 chapter and nothing else.
    private func deliversU1() -> ExchangeResponse {
        let state = fixture(BookState.self, "state")
        let ch = fixture(ChapterStatus.self, "chapter").chapter!
        return ExchangeResponse(results: [], gate: nil, chapter: ch, state: state,
                                breakSuggestion: nil, authoring: nil, resultsDoc: nil)
    }

    private func deliversNothing() -> ExchangeResponse {
        ExchangeResponse(results: [], gate: nil, chapter: nil, state: fixture(BookState.self, "state"),
                         breakSuggestion: nil, authoring: nil, resultsDoc: nil)
    }

    func testBelowTheGateOffersTheBreakThenRemediationOrOverride() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] req in
            if req.phase == "start" { return self.deliversU1() }
            if req.override { return self.fixture(ExchangeResponse.self, "exchange-screener") }
            if req.breakMinutes != nil { return self.deliversNothing() }
            return self.fixture(ExchangeResponse.self, "exchange-fail")
        }
        let s = session()
        await s.open()
        guard case .reading = s.screen else { return XCTFail("\(s.screen)") }
        await s.submitCheck([ItemResponse(itemId: "u1-q1", response: "wrong", selectedIndex: nil, confidence: 4)])
        guard case .results(let doc, let gate) = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertFalse(gate.passed)
        XCTAssertTrue(doc.headline.hasPrefix("Below the gate"))

        // Leaving the results takes the suggested break first.
        await s.proceed()
        guard case .takingBreak(let b) = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertEqual(b.kind, "short")

        // After the break, the remediation chapter that was authoring.
        await s.breakFinished(minutes: 5)
        XCTAssertNotNil(fake.exchanges.last?.breakMinutes)
        guard case .reading = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertEqual(s.chapter?.unit, "u1")

        // Override instead: the exchange carries the flag and whatever it
        // delivers is shown.
        await s.submitCheck([ItemResponse(itemId: "u1-q1", response: "wrong", selectedIndex: nil, confidence: 4)])
        await s.override()
        XCTAssertEqual(fake.exchanges.last?.override, true)
        guard case .series = s.screen else { return XCTFail("\(s.screen)") }
    }

    func testServerAwayIsAnErrorScreenAndTryAgainRecovers() async {
        let s = session()
        await s.open()
        guard case .error(let message) = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertEqual(message, "The server can't be reached.")

        scriptFreshBook()
        await s.retry()
        guard case .placement = s.screen else { return XCTFail("\(s.screen)") }
    }

    func testCachedChapterReadsWhenTheServerIsAway() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] _ in self.deliversU1() }
        let s = session()
        await s.open()
        guard case .reading = s.screen else { return XCTFail("\(s.screen)") }

        // Relaunch with the server gone: the cached chapter is readable.
        let away = FakeService()
        let s2 = BookSession(subjectID: "ai", title: "How AI Works", service: away, storage: storage)
        await s2.open()
        guard case .reading = s2.screen else { return XCTFail("\(s2.screen)") }
        XCTAssertEqual(s2.chapter?.unit, "u1")
    }

    func testStaleCachedChapterIsDroppedWhenTheServerMovedOn() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] _ in self.deliversU1() }
        let s = session()
        await s.open()
        XCTAssertEqual(s.chapter?.unit, "u1")

        // The server was reset: u1 is locked and not in the fringe, so the
        // cached chapter goes and the book starts from the top.
        fake.onState = { _ in
            BookState(spine: [
                SpineEntry(unit: "u0", title: "Calibration", status: "locked", score: nil, inFringe: true),
                SpineEntry(unit: "u1", title: "The core bet", status: "locked", score: nil, inFringe: false),
            ], fringe: ["u0"], debt: [], activeMisconceptions: [], summary: "", sessionMinutes: 0)
        }
        fake.onExchange = { [unowned self] _ in self.fixture(ExchangeResponse.self, "exchange-start") }
        let s2 = BookSession(subjectID: "ai", title: "How AI Works", service: fake, storage: storage)
        await s2.open()
        guard case .placement = s2.screen else { return XCTFail("\(s2.screen)") }
    }

    func testRelaunchMidAuthoringResumesTheWait() async {
        scriptFreshBook()
        fake.onChapter = { _ in ChapterStatus(chapter: nil, authoring: true, authoringError: "") }
        let s = session()
        s.pollInterval = 60
        await s.open()
        await s.place(level: 3)
        await s.submitCheck(s.chapter!.check.map {
            ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
        })
        // The authoring wait persists before the first poll sleeps.
        let waiting = Task { await s.proceed() }
        try? await Task.sleep(nanoseconds: 100_000_000)
        guard case .authoring(let unit) = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertEqual(unit, "u1")
        waiting.cancel()

        fake.onChapter = { [unowned self] _ in self.fixture(ChapterStatus.self, "chapter") }
        let s2 = session()
        await s2.open()
        guard case .reading = s2.screen else { return XCTFail("\(s2.screen)") }
        XCTAssertEqual(s2.chapter?.unit, "u1")
    }

    func testDuplicateSubmitsAreDroppedWhileOneIsInFlight() async {
        scriptFreshBook()
        let s = session()
        await s.open()
        await s.place(level: 3)
        let answers = s.chapter!.check.map {
            ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
        }
        let before = fake.exchanges.count
        async let a: Void = s.submitCheck(answers)
        async let b: Void = s.submitCheck(answers)
        _ = await (a, b)
        XCTAssertEqual(fake.exchanges.count - before, 1)
    }

    func testStartOverResetsAndReopensFromTheTop() async {
        scriptFreshBook()
        fake.onReset = { [unowned self] _ in self.fixture(BookState.self, "state") }
        let s = session()
        await s.open()
        await s.place(level: 3)
        await s.startOver()
        guard case .placement = s.screen else { return XCTFail("\(s.screen)") }
        XCTAssertEqual(fake.exchanges.last?.phase, "start")
    }
}
