import XCTest
@testable import Chiron

/// The library's sidebar picks what the list shows: everything, one kind,
/// or one shelf. Its Continue section is what the reader is in the middle
/// of, and the list can be sorted.
@MainActor
final class LibraryLayoutTests: XCTestCase {
    private let rows = [
        SubjectInfo(id: "ai", title: "How AI Works", unitsTotal: 11, unitsCleared: 1, kind: "book", shelf: "s1"),
        SubjectInfo(id: "data", title: "Where the Words Come From", unitsTotal: 14, unitsCleared: 0, kind: "book"),
        SubjectInfo(id: "primer-why", title: "Anything at all", kind: "primer", status: "ready"),
        SubjectInfo(id: "primer-new", title: "Being written", kind: "primer", status: "authoring"),
        SubjectInfo(id: "read-simon", title: "Simon Willison's Weblog", unitsTotal: 26, kind: "feed", unread: 26),
        SubjectInfo(id: "read-fly", title: "The Fly Blog", unitsTotal: 40, kind: "feed", unread: 3),
        SubjectInfo(id: "read-quiet", title: "A quiet blog", unitsTotal: 9, kind: "feed", unread: 0),
        SubjectInfo(id: "read-pc", title: "Private Capital", unitsTotal: 88, unitsCleared: 4, kind: "reading", shelf: "s1"),
        SubjectInfo(id: "doc-a", title: "Alinea", kind: "pdf", pages: 12, page: 0),
        SubjectInfo(id: "doc-b", title: "Norm-Metrics", kind: "pdf", pages: 30, page: 7),
    ]

    private func library(active: String? = "data") async -> Library {
        let fake = FakeService()
        let rows = self.rows
        fake.onSubjects = { SubjectsResponse(subjects: rows, active: active, shelves: [ShelfInfo(id: "s1", name: "Fly.io", subjects: ["ai", "read-pc"])]) }
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let defaults = UserDefaults(suiteName: "library-layout-\(UUID().uuidString)")!
        let library = Library(storage: storage, service: fake, defaults: defaults)
        // By kind, which keeps the server's order inside each kind.
        library.order = .kind
        await library.refresh()
        return library
    }

    func testEachSubjectHasOneKind() {
        XCTAssertEqual(rows.map(\.libraryKind), [.book, .book, .primer, .primer, .feed, .feed, .feed, .reading, .pdf, .pdf])
        XCTAssertEqual(SubjectInfo(id: "old", title: "A server before kinds").libraryKind, .book)
    }

    func testTheSidebarPicksWhatIsListed() async {
        let library = await library()
        XCTAssertEqual(library.listed(in: .all).count, rows.count, "all is everything, shelved or not")
        XCTAssertEqual(library.listed(in: .kind(.feed)).map(\.id), ["read-simon", "read-fly", "read-quiet"])
        XCTAssertEqual(library.listed(in: .kind(.pdf)).map(\.id), ["doc-a", "doc-b"])
        XCTAssertEqual(library.listed(in: .shelf("s1")).map(\.id), ["ai", "read-pc"])
        XCTAssertEqual(library.listed(in: .shelf("gone")).map(\.id), [])
    }

    func testFeedsCountWhatIsUnread() async {
        let library = await library()
        XCTAssertEqual(library.unreadTotal, 29)
    }

    /// The open one first, then what is started and not finished: a book
    /// with units cleared, a reading, a PDF past its first page, a primer
    /// still being written. Not what is untouched, and not a feed unless
    /// it is the one open.
    func testContinueIsWhatTheReaderIsInTheMiddleOf() async {
        var library = await library(active: "data")
        XCTAssertEqual(library.continuing.map(\.id), ["data", "ai", "primer-new", "read-pc", "doc-b"])
        library = await self.library(active: "read-fly")
        XCTAssertEqual(library.continuing.first?.id, "read-fly", "a feed is there while it is open")
        XCTAssertFalse(library.continuing.contains { $0.id == "read-simon" })
    }

    func testTheListSorts() {
        XCTAssertEqual(LibrarySort.recent.apply(rows).map(\.id), LibrarySort.title.apply(rows).map(\.id),
                       "with no stamps from the server, most recent first is by title")
        XCTAssertEqual(LibrarySort.title.apply(rows).prefix(3).map(\.id), ["read-quiet", "doc-a", "primer-why"],
                       "by title, ignoring case")
        XCTAssertEqual(LibrarySort.unread.apply(rows).prefix(2).map(\.id), ["read-simon", "read-fly"],
                       "most unread first, the rest in the server's order")
        XCTAssertEqual(LibrarySort.unread.apply(rows)[2].id, "ai")
        XCTAssertEqual(LibrarySort.kind.apply(rows).map(\.libraryKind),
                       [.book, .book, .primer, .primer, .feed, .feed, .feed, .reading, .pdf, .pdf])
    }

    /// A shelf deleted here or elsewhere no longer holds the sidebar.
    func testADeletedShelfIsNoLongerShown() async {
        let fake = FakeService()
        var shelves = [ShelfInfo(id: "s1", name: "Fly.io")]
        let rows = self.rows
        fake.onSubjects = { SubjectsResponse(subjects: rows, active: nil, shelves: shelves) }
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let library = Library(storage: storage, service: fake, defaults: UserDefaults(suiteName: "library-layout-\(UUID().uuidString)")!)
        await library.refresh()
        library.scope = .shelf("s1")
        await library.refresh()
        XCTAssertEqual(library.scope, .shelf("s1"))
        shelves = []
        await library.refresh()
        XCTAssertEqual(library.scope, .all)
    }
}
