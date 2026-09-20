import XCTest
@testable import Chiron

/// A page the reader is reading somewhere else: the link goes to the
/// server, which fetches the article and puts it on the shelf as a
/// reading, and the app opens it. From there it is a chapter like any
/// other, so a highlight in it reaches the tutor with the whole piece
/// behind it.
@MainActor
final class ReadPageTests: XCTestCase {
    private func library(_ fake: FakeService) -> Library {
        Library(storage: FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString),
                service: fake)
    }

    func testALinkIsReadAsAPageOnTheShelf() async {
        let fake = FakeService()
        var rows = SubjectsResponse(subjects: [SubjectInfo(id: "ai", title: "How AI Works", kind: "book")],
                                    active: "ai", shelves: [])
        fake.onSubjects = { rows }
        fake.onReadPage = { url in
            rows = SubjectsResponse(subjects: rows.subjects + [SubjectInfo(id: "read-2", title: "Prompt injection in 2026", kind: "reading")],
                                    active: "ai", shelves: [])
            XCTAssertEqual(url, "https://simonwillison.net/2026/Sep/20/injection/")
            return ImportedBook(id: "read-2", title: "Prompt injection in 2026", chapters: 1)
        }
        let library = library(fake)
        await library.refresh()

        await library.readPage("https://simonwillison.net/2026/Sep/20/injection/")

        XCTAssertEqual(fake.pagesRead, ["https://simonwillison.net/2026/Sep/20/injection/"])
        XCTAssertTrue(library.subjects.contains { $0.id == "read-2" }, "the page is on the shelf")
        XCTAssertEqual(library.session?.subjectID, "read-2", "the page opens where it was read")
        XCTAssertNil(library.shelfError)
    }

    /// The card a shared link opens is the place to say "read it", so the
    /// card goes away when the page is taken.
    func testReadingALinkClosesTheCaptureCard() async {
        let fake = FakeService()
        fake.onSubjects = { SubjectsResponse(subjects: [], active: nil, shelves: []) }
        fake.onReadPage = { _ in ImportedBook(id: "read-3", title: "A post", chapters: 1) }
        let library = library(fake)
        library.pendingCapture = Capture(text: "", sourceURL: "https://fly.io/blog/sprites/")
        library.captureAnswer = "an older answer"

        await library.readPage("https://fly.io/blog/sprites/")

        XCTAssertNil(library.pendingCapture)
        XCTAssertNil(library.captureAnswer)
    }

    /// A link that is not a page to read leaves the shelf as it was and
    /// says why.
    func testALinkThatCannotBeReadSaysSo() async {
        let fake = FakeService()
        fake.onSubjects = { SubjectsResponse(subjects: [], active: nil, shelves: []) }
        fake.onReadPage = { _ in throw URLError(.cannotConnectToHost) }
        let library = library(fake)

        await library.readPage("https://example.com/nothing/")

        XCTAssertNil(library.session)
        XCTAssertNotNil(library.shelfError)
    }

    /// Anything that is not an address is not sent at all.
    func testSomethingThatIsNotALinkIsNotSent() async {
        let fake = FakeService()
        fake.onSubjects = { SubjectsResponse(subjects: [], active: nil, shelves: []) }
        let library = library(fake)

        await library.readPage("what even is this")

        XCTAssertTrue(fake.pagesRead.isEmpty)
        XCTAssertNotNil(library.shelfError)
    }
}
