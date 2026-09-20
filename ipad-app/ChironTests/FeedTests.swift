import XCTest
@testable import Chiron

/// A blog the reader follows: the feed goes up once, and from then on the
/// shelf card carries its posts as chapters and counts what has not been
/// read. Nothing polls - the server sleeps between readers - so opening
/// the app is what asks the blogs what is new.
@MainActor
final class FeedTests: XCTestCase {
    private func library(_ fake: FakeService) -> Library {
        Library(storage: FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString),
                service: fake)
    }

    /// One post as the server sends it: read as it is, nothing to answer.
    private func post(_ unit: String) -> ChapterStatus {
        let json = """
        {"chapter":{"unit":"\(unit)","title":"Prompt injection in 2026","minutes":4,
         "html":"<p>An agent that reads the web reads whatever an attacker wrote.</p>",
         "beats":[],"pretest":[],"check":[],"calibration":false,"next_action":"read"},
         "authoring":false,"authoring_error":""}
        """
        return try! JSONDecoder().decode(ChapterStatus.self, from: Data(json.utf8))
    }

    private func shelf(_ rows: [SubjectInfo]) -> SubjectsResponse {
        SubjectsResponse(subjects: rows, active: nil, shelves: [])
    }

    func testFollowingABlogPutsItOnTheShelfAndOpensIt() async {
        let fake = FakeService()
        var rows: [SubjectInfo] = []
        fake.onSubjects = { [rows] in SubjectsResponse(subjects: rows, active: nil, shelves: []) }
        fake.onFollowFeed = { url in
            XCTAssertEqual(url, "https://simonwillison.net/atom/everything/")
            rows = [SubjectInfo(id: "read-7", title: "Simon Willison's Weblog", kind: "feed", unread: 17)]
            return ImportedBook(id: "read-7", title: "Simon Willison's Weblog", chapters: 17)
        }
        let library = library(fake)

        await library.followFeed("https://simonwillison.net/atom/everything/")

        XCTAssertEqual(fake.feedsFollowed.count, 1)
        XCTAssertEqual(library.session?.subjectID, "read-7")
        XCTAssertNil(library.shelfError)
    }

    /// The shelf asks the blogs what is new when it refreshes, and what
    /// came back lands without the reader asking twice.
    func testTheShelfAsksTheBlogsWhatIsNew() async {
        let fake = FakeService()
        var unread = 2
        fake.onSubjects = { [unowned self] in
            self.shelf([SubjectInfo(id: "read-7", title: "A Weblog", unitsTotal: 9, kind: "feed", unread: unread)])
        }
        fake.onCheckFeeds = {
            unread = 3
            return FeedCheck(checked: 1, added: 1)
        }
        let library = library(fake)

        await library.refresh()
        // The check runs beside the shelf; wait for it to land.
        for _ in 0..<40 where library.subjects.first?.unread != 3 {
            try? await Task.sleep(nanoseconds: 25_000_000)
        }
        XCTAssertEqual(fake.feedChecks, 1)
        XCTAssertEqual(library.subjects.first?.unread, 3, "what came back is on the shelf")
    }

    /// A shelf with no blog on it has nothing to ask.
    func testAShelfWithoutBlogsAsksNothing() async {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in
            self.shelf([SubjectInfo(id: "ai", title: "How AI Works", kind: "book")])
        }
        let library = library(fake)
        await library.refresh()
        try? await Task.sleep(nanoseconds: 100_000_000)
        XCTAssertEqual(fake.feedChecks, 0)
    }

    /// Opening a post is reading it, on every device.
    func testOpeningAPostMarksItRead() async {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in
            self.shelf([SubjectInfo(id: "read-7", title: "A Weblog", kind: "feed", unread: 2)])
        }
        fake.onCheckFeeds = { FeedCheck(checked: 1, added: 0) }
        fake.onState = { _ in
            BookState(spine: [SpineEntry(unit: "u3", title: "Prompt injection in 2026", status: "active", score: nil, inFringe: true)],
                      fringe: ["u3"], debt: [], activeMisconceptions: [], summary: "", sessionMinutes: 0)
        }
        fake.onChapter = { [unowned self] _ in self.post("u3") }
        let library = library(fake)
        await library.refresh()

        await library.open("read-7")
        // The mark goes up beside the reading, not in front of it.
        for _ in 0..<40 where fake.readMarks.isEmpty {
            try? await Task.sleep(nanoseconds: 25_000_000)
        }

        XCTAssertEqual(fake.readMarks.map(\.unit), ["u3"])
        XCTAssertEqual(fake.readMarks.first?.subject, "read-7")
    }

    /// Something that is not an address is not sent at all.
    func testSomethingThatIsNotALinkIsNotFollowed() async {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.shelf([]) }
        let library = library(fake)
        await library.followFeed("my favourite blog")
        XCTAssertTrue(fake.feedsFollowed.isEmpty)
        XCTAssertNotNil(library.shelfError)
    }
}
