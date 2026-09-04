import XCTest
@testable import Chiron

/// The shelf's own behaviour: what happens to a capture once its question
/// is sent, without a server.
@MainActor
final class LibraryTests: XCTestCase {
    private func subjects(_ status: String) -> SubjectsResponse {
        let json = """
        {"subjects":[
          {"id":"ai","title":"How AI Works","kind":"book"},
          {"id":"primer-why","title":"Why","kind":"primer","status":"\(status)","source":{"text":"words"},"captured_at":"2026-09-04T12:00:00Z"}
        ],"active":"ai"}
        """
        return try! JSONDecoder().decode(SubjectsResponse.self, from: Data(json.utf8))
    }

    /// A capture sent from inside a book: the reader lands on the shelf to
    /// watch the card, and the primer opens by itself when it is ready.
    func testAPrimerOpensWhenTheServerHasWrittenIt() async throws {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.subjects("authoring") }
        fake.onCapture = { _ in CaptureResponse(subject: "primer-why", status: "authoring", title: "Why") }
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let library = Library(storage: storage, service: fake)
        library.shelfPollInterval = 0.05
        await library.refresh()
        await library.open("ai")
        XCTAssertEqual(library.session?.subjectID, "ai")

        let id = try await library.submitCapture(Capture(text: "words"), prompt: "why?")
        XCTAssertEqual(id, "primer-why")
        XCTAssertNil(library.session, "the shelf shows the card being written")
        XCTAssertEqual(fake.captures.first?.prompt, "why?")

        fake.onSubjects = { [unowned self] in self.subjects("ready") }
        let deadline = Date().addingTimeInterval(3)
        while library.session?.subjectID != "primer-why", Date() < deadline {
            try await Task.sleep(nanoseconds: 20_000_000)
        }
        XCTAssertEqual(library.session?.subjectID, "primer-why", "the primer opened on its own")
        XCTAssertEqual(library.session?.kind, "primer")
    }

    /// A primer that failed stays on the shelf with its error; nothing opens.
    func testAFailedPrimerStaysOnTheShelf() async throws {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.subjects("authoring") }
        fake.onCapture = { _ in CaptureResponse(subject: "primer-why", status: "authoring", title: "Why") }
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let library = Library(storage: storage, service: fake)
        library.shelfPollInterval = 0.05
        _ = try await library.submitCapture(Capture(text: "words"), prompt: "why?")
        fake.onSubjects = { [unowned self] in self.subjects("failed") }
        try await Task.sleep(nanoseconds: 300_000_000)
        XCTAssertNil(library.session)
        XCTAssertTrue(library.subjects.contains(where: { $0.id == "primer-why" && $0.failed }))
    }
}
