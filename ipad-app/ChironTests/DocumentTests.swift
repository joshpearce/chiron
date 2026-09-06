import XCTest
@testable import Chiron

/// A PDF on the shelf opens in its own reader, not as a book: the file
/// is fetched once and kept, the page turned to is the server's page, and
/// a page turned here reaches the server so the other device opens there.
@MainActor
final class DocumentTests: XCTestCase {
    private var pdf: Data {
        let dir = Bundle(for: DocumentTests.self).url(forResource: "fixtures", withExtension: nil)!
        return try! Data(contentsOf: dir.appendingPathComponent("sample.pdf"))
    }

    private func library(_ fake: FakeService) -> Library {
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let l = Library(storage: storage, service: fake)
        l.documentPositionDelay = 0
        return l
    }

    private func shelf(page: Int = 0) -> SubjectsResponse {
        SubjectsResponse(subjects: [
            SubjectInfo(id: "ai", title: "How AI Works", kind: "book"),
            SubjectInfo(id: "doc-1", title: "A paper", kind: "pdf", pages: 2, page: page),
        ], active: "ai", shelves: [])
    }

    func testAPDFRowOpensAsADocumentAtTheServersPage() async {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.shelf(page: 1) }
        fake.onDocumentData = { [unowned self] _ in self.pdf }
        let library = library(fake)
        await library.refresh()
        await library.open("doc-1")
        XCTAssertNil(library.session, "a PDF is not a book session")
        XCTAssertEqual(library.document?.id, "doc-1")
        XCTAssertEqual(library.document?.title, "A paper")
        XCTAssertEqual(library.document?.page, 1)
        XCTAssertEqual(library.document?.pages, 2)
        XCTAssertEqual(try? Data(contentsOf: library.document!.fileURL), pdf, "the file is kept beside the app")

        library.closeBook()
        XCTAssertNil(library.document)
        await library.open("doc-1")
        XCTAssertEqual(fake.documentFetches, ["doc-1"], "the second open reads the kept file")
    }

    func testTheKeptFileOpensWhenTheServerIsAway() async {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.shelf() }
        fake.onDocumentData = { [unowned self] _ in self.pdf }
        let library = library(fake)
        await library.refresh()
        await library.open("doc-1")
        library.closeBook()
        fake.onDocumentData = { _ in throw URLError(.cannotConnectToHost) }
        await library.open("doc-1")
        XCTAssertEqual(library.document?.id, "doc-1")
    }

    func testTurningAPageReachesTheServer() async {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.shelf() }
        fake.onDocumentData = { [unowned self] _ in self.pdf }
        let library = library(fake)
        await library.refresh()
        await library.open("doc-1")
        library.document?.turned(to: 1, position: 0.25)
        try? await Task.sleep(nanoseconds: 100_000_000)
        XCTAssertEqual(fake.positions.map(\.page), [1])
        XCTAssertEqual(fake.positions.last?.position, 0.25)
        library.closeBook()
        try? await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertEqual(fake.positions.count, 1, "closing without a change sends nothing more")
    }

    func testImportingAPDFUploadsItWithItsPageCountAndRefreshes() async throws {
        let fake = FakeService()
        var rows = shelf()
        fake.onSubjects = { rows }
        let library = library(fake)
        await library.refresh()
        let url = FileManager.default.temporaryDirectory.appendingPathComponent("An essay.pdf")
        try pdf.write(to: url)
        rows = SubjectsResponse(subjects: rows.subjects + [SubjectInfo(id: "doc-An essay", title: "An essay", kind: "pdf", pages: 2)], active: "ai", shelves: [])
        await library.importPDF(at: url)
        XCTAssertEqual(fake.uploads.map(\.title), ["An essay"])
        XCTAssertEqual(fake.uploads.first?.pages, 2, "PDFKit counted the pages")
        XCTAssertEqual(fake.uploads.first?.data, pdf)
        XCTAssertTrue(library.subjects.contains { $0.id == "doc-An essay" }, "the shelf was refreshed")
        XCTAssertNil(library.shelfError)

        let text = FileManager.default.temporaryDirectory.appendingPathComponent("notes.txt")
        try Data("hello".utf8).write(to: text)
        await library.importPDF(at: text)
        XCTAssertEqual(fake.uploads.count, 1, "not a PDF, not sent")
        XCTAssertNotNil(library.shelfError)
    }
}
