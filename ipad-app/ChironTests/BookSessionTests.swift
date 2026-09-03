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
    var onInk: (String, InkSubmission) throws -> ExchangeResponse = { _, _ in throw URLError(.cannotConnectToHost) }
    var onAsk: (String, String, String) throws -> AskResponse = { _, _, _ in throw URLError(.cannotConnectToHost) }
    var exchanges: [ExchangeRequest] = []
    var inks: [InkSubmission] = []
    var asks: [(unit: String, quote: String, question: String)] = []
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
    func ink(subject: String, _ submission: InkSubmission) async throws -> ExchangeResponse {
        inks.append(submission)
        return try onInk(subject, submission)
    }
    func ask(subject: String, unit: String, quote: String, question: String) async throws -> AskResponse {
        asks.append((unit, quote, question))
        return try onAsk(unit, quote, question)
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
        // Like a real server: no chapter until a graded series has had u1
        // authored, then u1.
        fake.onChapter = { [unowned self] _ in
            self.fake.exchanges.contains { $0.checkResponses.count > 1 }
                ? self.fixture(ChapterStatus.self, "chapter")
                : ChapterStatus(chapter: nil, authoring: false, authoringError: "")
        }
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
        XCTAssertTrue(fake.exchanges.contains { $0.breakMinutes != nil }, "the break was reported")
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

    func testReadingPositionSurvivesRelaunch() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] _ in self.deliversU1() }
        let s = session()
        await s.open()
        s.recordPosition(unit: "u1", offset: 1234.5)
        s.persist()

        let s2 = session()
        await s2.open()
        XCTAssertEqual(s2.position(for: "u1"), 1234.5)
        XCTAssertEqual(s2.position(for: "u2"), 0, "an unread chapter opens at the top")
    }

    func testAnyInkSendsTheWholeCheckThroughTheInkCheckIn() async {
        scriptFreshBook()
        fake.onInk = { [unowned self] _, _ in self.fixture(ExchangeResponse.self, "exchange-series") }
        let s = session()
        await s.open()
        await s.place(level: 3)
        let items = s.chapter!.check
        let strokes = [[InkAnswer.Point(x: 0.1, y: 0.2), InkAnswer.Point(x: 0.5, y: 0.6)]]
        var responses = [
            ItemResponse(itemId: items[0].id, response: nil, selectedIndex: nil, confidence: 3,
                         ink: InkAnswer(strokes: strokes, aspect: 3)),
            ItemResponse(itemId: items[1].id, response: "typed", selectedIndex: nil, confidence: 4),
            ItemResponse(itemId: items[2].id, response: nil, selectedIndex: nil, confidence: 1, idk: true),
        ]
        responses += items.dropFirst(3).map {
            ItemResponse(itemId: $0.id, response: nil, selectedIndex: 0, confidence: 2)
        }
        let exchangesBefore = fake.exchanges.count
        await s.submitCheck(responses)

        XCTAssertEqual(fake.exchanges.count, exchangesBefore, "handwriting never travels on the exchange")
        let sub = try! XCTUnwrap(fake.inks.last)
        XCTAssertEqual(sub.unit, "u0")
        XCTAssertEqual(sub.items.count, items.count)
        XCTAssertEqual(sub.items[0].strokes, strokes)
        XCTAssertEqual(sub.items[0].aspect, 3)
        XCTAssertNil(sub.items[0].text)
        XCTAssertEqual(sub.items[1].text, "typed")
        XCTAssertTrue(sub.items[1].strokes.isEmpty)
        XCTAssertEqual(sub.items[2].idk, true)
        XCTAssertEqual(sub.items[3].selectedIndex, 0)
        guard case .results = s.screen else { return XCTFail("\(s.screen)") }

        // The ink never leaks onto the exchange wire either.
        let encoded = String(data: try! JSONEncoder().encode(responses[0]), encoding: .utf8)!
        XCTAssertFalse(encoded.contains("strokes"))
    }

    func testTypedOnlyChecksStayOnTheExchange() async {
        scriptFreshBook()
        let s = session()
        await s.open()
        await s.place(level: 3)
        await s.submitCheck(s.chapter!.check.map {
            ItemResponse(itemId: $0.id, response: "x", selectedIndex: nil, confidence: 2)
        })
        XCTAssertTrue(fake.inks.isEmpty)
        guard case .results = s.screen else { return XCTFail("\(s.screen)") }
    }

    func testChunkMinutesCountFromTheReaderOpening() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] req in
            req.phase == "start" ? self.deliversU1() : self.fixture(ExchangeResponse.self, "exchange-fail")
        }
        let s = session()
        await s.open()
        guard case .reading = s.screen else { return XCTFail("\(s.screen)") }

        // A relaunch onto the cached chapter starts the clock again.
        let s2 = session()
        await s2.open()
        guard case .reading = s2.screen else { return XCTFail("\(s2.screen)") }
        await s2.submitCheck([ItemResponse(itemId: "u1-q1", response: "x", selectedIndex: nil, confidence: 2)])
        let minutes = try! XCTUnwrap(fake.exchanges.last?.chunkMinutes)
        XCTAssertGreaterThanOrEqual(minutes, 0)
        XCTAssertLessThan(minutes, 1, "the clock started at this open, not at delivery")

        // The series is measurement, not reading: no chunk is reported.
        fake.onExchange = { [unowned self] req in
            if req.phase == "start" { return self.fixture(ExchangeResponse.self, "exchange-start") }
            if req.checkResponses.count == 1 { return self.fixture(ExchangeResponse.self, "exchange-screener") }
            return self.fixture(ExchangeResponse.self, "exchange-series")
        }
        let s3 = BookSession(subjectID: "data", title: "d", service: fake, storage: storage)
        s3.pollInterval = 0
        await s3.open()
        await s3.place(level: 3)
        await s3.submitCheck(s3.chapter!.check.map {
            ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
        })
        XCTAssertNil(fake.exchanges.last?.chunkMinutes)
    }

    func testTheServersChapterReplacesTheCachedOneOnOpen() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] _ in self.deliversU1() }
        let s = session()
        await s.open()
        XCTAssertEqual(s.chapter?.unit, "u1")
        let oldHTML = s.chapter!.html

        // Re-authored server-side: same unit, new prose.
        fake.onChapter = { [unowned self] _ in
            let ch = self.fixture(ChapterStatus.self, "chapter").chapter!
            let json = try! JSONEncoder().encode(ch)
            var obj = try! JSONSerialization.jsonObject(with: json) as! [String: Any]
            obj["html"] = "<p>rechained</p>"
            let fresh = try! JSONDecoder().decode(ChapterPayload.self, from: try! JSONSerialization.data(withJSONObject: obj))
            return ChapterStatus(chapter: fresh, authoring: false, authoringError: "")
        }
        let s2 = session()
        await s2.open()
        guard case .reading = s2.screen else { return XCTFail("\(s2.screen)") }
        XCTAssertEqual(s2.chapter?.html, "<p>rechained</p>")
        XCTAssertNotEqual(s2.chapter?.html, oldHTML)
        XCTAssertTrue(fake.exchanges.filter { $0.phase == "start" }.count == 1, "no second start: the server already had the chapter")

        // Still being rewritten: the wait screen, then the new chapter.
        var polls = 0
        fake.onChapter = { [unowned self] _ in
            polls += 1
            if polls < 2 { return ChapterStatus(chapter: nil, authoring: true, authoringError: "") }
            return self.fixture(ChapterStatus.self, "chapter")
        }
        let s3 = session()
        await s3.open()
        guard case .reading = s3.screen else { return XCTFail("\(s3.screen)") }
        XCTAssertEqual(polls, 2)
    }

    func testMarksAndInkBelongToTheChapterAndSurviveRelaunch() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] _ in self.deliversU1() }
        let s = session()
        await s.open()
        XCTAssertTrue(s.marks.isEmpty)
        s.addMark(kind: .highlight, start: 10, end: 40, text: "a model trained to do nothing")
        s.saveInk(Data([1, 2, 3]))

        let s2 = session()
        await s2.open()
        XCTAssertEqual(s2.marks.count, 1)
        XCTAssertEqual(s2.marks[0].kind, .highlight)
        XCTAssertEqual(s2.marks[0].text, "a model trained to do nothing")
        XCTAssertEqual(s2.inkData, Data([1, 2, 3]))
        s2.removeMark(s2.marks[0].id)
        s2.saveInk(nil)

        let s3 = session()
        await s3.open()
        XCTAssertTrue(s3.marks.isEmpty)
        XCTAssertNil(s3.inkData)

        // The same span marked twice is one mark.
        let first = s3.addMark(kind: .highlight, start: 10, end: 40, text: "a model trained to do nothing")
        let again = s3.addMark(kind: .highlight, start: 10, end: 40, text: "a model trained to do nothing")
        XCTAssertEqual(first.id, again.id)
        XCTAssertEqual(s3.marks.count, 1)
        s3.addMark(kind: .question, start: 10, end: 40, text: "a model trained to do nothing")
        XCTAssertEqual(s3.marks.count, 2, "a question on a highlighted span is its own mark")
    }

    func testAskingSendsThePassageAndKeepsTheAnswerOnTheMark() async {
        scriptFreshBook()
        fake.onExchange = { [unowned self] _ in self.deliversU1() }
        fake.onAsk = { unit, quote, q in AskResponse(unit: unit, answerMd: "Because $\\ln$ is what the code computes.") }
        let s = session()
        await s.open()

        // A question mark opens the card; closing it unasked leaves nothing.
        s.addMark(kind: .question, start: 100, end: 140, text: "the loss is this same quantity")
        XCTAssertNotNil(s.asking)
        s.closeAsking()
        XCTAssertNil(s.asking)
        XCTAssertTrue(s.marks.isEmpty)

        let m = s.addMark(kind: .question, start: 100, end: 140, text: "the loss is this same quantity")
        await s.ask("why nats and not bits?")
        XCTAssertEqual(fake.asks.count, 1)
        XCTAssertEqual(fake.asks[0].unit, "u1")
        XCTAssertEqual(fake.asks[0].quote, "the loss is this same quantity")
        XCTAssertEqual(fake.asks[0].question, "why nats and not bits?")
        XCTAssertEqual(s.asking?.mark.answer, "Because $\\ln$ is what the code computes.")
        XCTAssertEqual(s.marks.first?.answer, s.asking?.mark.answer)
        XCTAssertNil(s.asking?.error)

        // The answered question survives a relaunch and reopens on tap.
        s.closeAsking()
        let s2 = session()
        await s2.open()
        XCTAssertEqual(s2.marks.count, 1)
        s2.openMark(m.id)
        XCTAssertEqual(s2.asking?.mark.question, "why nats and not bits?")

        // The tutor away: the question stays on the mark, the card says so.
        fake.onAsk = { _, _, _ in throw URLError(.cannotConnectToHost) }
        s2.addMark(kind: .question, start: 1, end: 5, text: "One")
        await s2.ask("what?")
        XCTAssertNotNil(s2.asking?.error)
        XCTAssertNil(s2.asking?.mark.answer)
        XCTAssertEqual(s2.marks.count, 2)
    }
}
