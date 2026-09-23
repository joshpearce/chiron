import XCTest
@testable import Chiron

/// Getting things out of Chiron and into something else: the planning
/// conversation as text worth handing to another agent, and a picture
/// handed the other way, into a change request.
@MainActor
final class CopyOutTests: XCTestCase {
    private func plan(done: Bool = true) -> PlanState {
        PlanState(id: "primer-x", title: "Uber Agent Identity vs Macaroons", scale: "primer",
                  status: "planning", error: nil,
                  prompt: "What did Uber do, and how does it differ from macaroons?",
                  source: PrimerSource(text: "Uber is at the forefront of leveraging AI...", url: "https://uber.com/blog/x", app: nil),
                  brief: done ? "Write a primer of a few pages contrasting actor-chain JWTs with macaroons." : nil,
                  done: done,
                  plan: [PlanMessage(role: "tutor", text: "What does your macaroon carry?"),
                         PlanMessage(role: "learner", text: "A caveat per hop.")],
                  book: nil)
    }

    /// Everything said, in the order it was said, with the brief at the end -
    /// the thing you paste into another agent.
    func testTheConversationCopiesOutAsText() {
        let text = plan().transcript
        for want in ["Uber Agent Identity vs Macaroons",
                     "What did Uber do, and how does it differ from macaroons?",
                     "What does your macaroon carry?",
                     "A caveat per hop.",
                     "Write a primer of a few pages contrasting actor-chain JWTs with macaroons.",
                     "https://uber.com/blog/x"] {
            XCTAssertTrue(text.contains(want), "the transcript is missing \(want):\n\(text)")
        }
        // Who said what has to survive the paste.
        XCTAssertTrue(text.contains("Tutor:"), text)
        XCTAssertTrue(text.contains("You:"), text)
    }

    /// A conversation with no brief yet still copies; there is simply no
    /// brief in it.
    func testAConversationWithoutABriefStillCopies() {
        let text = plan(done: false).transcript
        XCTAssertTrue(text.contains("What does your macaroon carry?"), text)
        XCTAssertFalse(text.contains("The brief"), text)
    }

    /// A picture the reader picked goes up with the request instead of a
    /// picture of the screen.
    func testAChosenPictureGoesUpWithTheRequest() async {
        let fake = FakeService()
        fake.onSubjects = { SubjectsResponse(subjects: [], active: nil, shelves: []) }
        let library = Library(storage: FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString),
                              service: fake)
        let picture = Data("not really a png, but it is what was picked".utf8)

        _ = await library.requestChange("The toolbar is too tall", withPicture: true, picture: picture)

        XCTAssertEqual(fake.requests.count, 1)
        XCTAssertEqual(fake.requests.first?.png, picture,
                       "the picture the reader chose is the one that went up")
    }
}
