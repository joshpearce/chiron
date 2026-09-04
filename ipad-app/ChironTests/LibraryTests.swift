import XCTest
@testable import Chiron

/// The shelf's own behaviour: what happens to a capture once its question
/// is sent, without a server.
@MainActor
final class LibraryTests: XCTestCase {
    private func subjects(_ status: String, scale: String = "primer", extra: String = "") -> SubjectsResponse {
        let json = """
        {"subjects":[
          {"id":"ai","title":"How AI Works","kind":"book"},
          {"id":"primer-why","title":"Why","kind":"primer","status":"\(status)","scale":"\(scale)","source":{"text":"words"},"captured_at":"2026-09-04T12:00:00Z"}
          \(extra)
        ],"active":"ai"}
        """
        return try! JSONDecoder().decode(SubjectsResponse.self, from: Data(json.utf8))
    }

    private func library(_ fake: FakeService) -> Library {
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let l = Library(storage: storage, service: fake)
        l.shelfPollInterval = 0.05
        return l
    }

    private func planned(_ reply: String, done: Bool = false, brief: String? = nil, title: String? = nil) -> CaptureResponse {
        CaptureResponse(scale: "primer", answerMd: nil, subject: "primer-why", status: "planning", title: title ?? "Why",
                        replyMd: reply, done: done, brief: brief)
    }

    private func waitForSession(_ library: Library, _ id: String) async throws {
        let deadline = Date().addingTimeInterval(3)
        while library.session?.subjectID != id, Date() < deadline {
            try await Task.sleep(nanoseconds: 20_000_000)
        }
    }

    /// A summary comes back into the card; the shelf is untouched.
    func testASummaryIsAnsweredInTheCard() async throws {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.subjects("ready") }
        fake.onCapture = { _ in CaptureResponse(scale: "summary", answerMd: "It is a robots.txt line.") }
        let library = library(fake)
        library.pendingCapture = Capture(text: "words")
        let reply = try await library.submitCapture(Capture(text: "words"), prompt: "what is this?", scale: .summary)
        XCTAssertEqual(reply.answerMd, "It is a robots.txt line.")
        XCTAssertEqual(library.captureAnswer, "It is a robots.txt line.")
        XCTAssertNotNil(library.pendingCapture, "the card stays up with the answer in it")
        XCTAssertNil(library.planning)
        XCTAssertEqual(fake.captures.first?.scale, .summary)
    }

    /// A primer capture from inside a book: the tutor's first question
    /// opens the planning card, the reader answers, the brief arrives, the
    /// build sends them to the shelf, and the primer opens when written.
    func testAPrimerIsPlannedThenOpensWhenWritten() async throws {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.subjects("planning") }
        fake.onCapture = { [unowned self] _ in self.planned("Which part matters to you?") }
        let library = library(fake)
        await library.refresh()
        await library.open("ai")
        library.pendingCapture = Capture(text: "words")

        let reply = try await library.submitCapture(Capture(text: "words"), prompt: "why?")
        XCTAssertEqual(reply.subject, "primer-why")
        XCTAssertNil(library.pendingCapture, "the capture card gave way")
        let plan = try XCTUnwrap(library.planning, "the planning card opened")
        XCTAssertEqual(plan.plan, [PlanMessage(role: "tutor", text: "Which part matters to you?")])
        XCTAssertFalse(plan.done)
        XCTAssertEqual(library.session?.subjectID, "ai", "planning does not close the book")
        XCTAssertTrue(library.subjects.contains(where: { $0.id == "primer-why" && $0.drafting }), "the draft is on the shelf")

        fake.onPlanTurn = { [unowned self] _, _ in self.planned("A primer on the ai-input part, then.", done: true, brief: "I want the ai-input part.", title: "Content-Signal") }
        await library.planReply("the ai-input part")
        let after = try XCTUnwrap(library.planning)
        XCTAssertEqual(after.plan.count, 3)
        XCTAssertEqual(after.plan[1], PlanMessage(role: "learner", text: "the ai-input part"))
        XCTAssertTrue(after.done)
        XCTAssertEqual(after.brief, "I want the ai-input part.")
        XCTAssertEqual(after.title, "Content-Signal")
        XCTAssertEqual(fake.planTurns.first?.text, "the ai-input part")

        fake.onBuild = { id in BuildResponse(subject: id, status: "authoring", title: "Content-Signal", book: nil) }
        fake.onSubjects = { [unowned self] in self.subjects("authoring") }
        await library.buildDraft()
        XCTAssertEqual(fake.builds, ["primer-why"])
        XCTAssertNil(library.planning)
        XCTAssertNil(library.session, "the shelf shows the primer being written")

        fake.onSubjects = { [unowned self] in self.subjects("ready") }
        try await waitForSession(library, "primer-why")
        XCTAssertEqual(library.session?.subjectID, "primer-why", "the primer opened on its own")
        XCTAssertEqual(library.session?.kind, "primer")
    }

    /// A book draft: the build starts a generation; the book opens when
    /// it arrives on the shelf under its own name.
    func testABookDraftOpensTheBookWhenItArrives() async throws {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.subjects("planning", scale: "book") }
        fake.onCapture = { _ in CaptureResponse(scale: "book", subject: "primer-why", status: "planning", title: "Why", replyMd: "Enough.", done: true, brief: "a book") }
        let library = library(fake)
        _ = try await library.submitCapture(Capture(text: "words"), prompt: "why?", scale: .book)
        XCTAssertEqual(library.planning?.isBook, true)
        fake.onBuild = { id in BuildResponse(subject: id, status: "building", title: nil, book: "why-book") }
        fake.onSubjects = { [unowned self] in self.subjects("building", scale: "book") }
        await library.buildDraft()
        XCTAssertTrue(library.subjects.contains(where: { $0.id == "primer-why" && $0.building && $0.authoring }))
        fake.onSubjects = { [unowned self] in
            self.subjects("building", scale: "book", extra: ",{\"id\":\"why-book\",\"title\":\"Why, the book\",\"kind\":\"book\"}")
        }
        try await waitForSession(library, "why-book")
        XCTAssertEqual(library.session?.subjectID, "why-book", "the book opened when it arrived")
    }

    /// Leaving the plan keeps the draft; the shelf reopens it with its
    /// conversation, and a discard removes it.
    func testADraftLeftOnTheShelfReopensAndCanBeDiscarded() async throws {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.subjects("planning") }
        fake.onCapture = { [unowned self] _ in self.planned("Which part?") }
        let library = library(fake)
        _ = try await library.submitCapture(Capture(text: "words"), prompt: "why?")
        library.planning = nil
        XCTAssertTrue(library.subjects.contains(where: { $0.id == "primer-why" && $0.drafting }))

        fake.onPlan = { id in
            PlanState(id: id, title: "Why", scale: "primer", status: "planning", error: nil, prompt: "why?",
                      source: PrimerSource(text: "words", url: nil, app: nil), brief: nil, done: false,
                      plan: [PlanMessage(role: "tutor", text: "Which part?")], book: nil)
        }
        await library.openDraft("primer-why")
        XCTAssertEqual(library.planning?.plan.count, 1)
        fake.onSubjects = { [unowned self] in self.subjects("ready", extra: "") }
        await library.discardDraft()
        XCTAssertEqual(fake.discards, ["primer-why"])
        XCTAssertNil(library.planning)
    }

    /// A primer that failed stays on the shelf with its error; nothing opens.
    func testAFailedPrimerStaysOnTheShelf() async throws {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.subjects("planning") }
        fake.onCapture = { [unowned self] _ in self.planned("Which part?") }
        fake.onBuild = { id in BuildResponse(subject: id, status: "authoring", title: "Why", book: nil) }
        let library = library(fake)
        _ = try await library.submitCapture(Capture(text: "words"), prompt: "why?")
        fake.onSubjects = { [unowned self] in self.subjects("authoring") }
        await library.buildDraft()
        fake.onSubjects = { [unowned self] in self.subjects("failed") }
        try await Task.sleep(nanoseconds: 300_000_000)
        XCTAssertNil(library.session)
        XCTAssertTrue(library.subjects.contains(where: { $0.id == "primer-why" && $0.failed }))
    }
}
