import PencilKit
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

    func testAPassageOfThePDFOpensTheCaptureCardNamingThePage() async {
        let fake = FakeService()
        fake.onSubjects = { [unowned self] in self.shelf() }
        fake.onDocumentData = { [unowned self] _ in self.pdf }
        let library = library(fake)
        await library.refresh()
        await library.open("doc-1")
        library.document?.captured("  Two coins in a box.  ", page: 1)
        XCTAssertEqual(library.pendingCapture?.text, "Two coins in a box.")
        XCTAssertEqual(library.pendingCapture?.sourceApp, "A paper, page 2")
        library.pendingCapture = nil
        library.document?.captured("   ", page: 0)
        XCTAssertNil(library.pendingCapture, "nothing selected, no card")
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

/// Ink on a PDF: what the other device drew comes down with the document,
/// a stroke here goes up with the page's version, and a page both drew on
/// while apart ends with both drawings.
@MainActor
final class DocumentInkTests: XCTestCase {
    private var pdf: Data {
        let dir = Bundle(for: DocumentInkTests.self).url(forResource: "fixtures", withExtension: nil)!
        return try! Data(contentsOf: dir.appendingPathComponent("sample.pdf"))
    }

    private func stroke(at x: Double) -> PKDrawing {
        let path = PKStrokePath(controlPoints: [x, x + 40].map {
            PKStrokePoint(location: CGPoint(x: $0, y: 100), timeOffset: 0, size: CGSize(width: 3, height: 3), opacity: 1, force: 1, azimuth: 0, altitude: .pi / 2)
        }, creationDate: Date())
        return PKDrawing(strokes: [PKStroke(ink: PKInk(.pen, color: .black), path: path)])
    }

    private func library(_ fake: FakeService) -> Library {
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let l = Library(storage: storage, service: fake)
        l.documentPositionDelay = 0
        return l
    }

    private func open(_ fake: FakeService) async -> (Library, DocumentSession) {
        fake.onSubjects = { SubjectsResponse(subjects: [SubjectInfo(id: "doc-1", title: "A paper", kind: "pdf", pages: 2)], active: nil, shelves: []) }
        fake.onDocumentData = { [unowned self] _ in self.pdf }
        let library = library(fake)
        await library.refresh()
        await library.open("doc-1")
        return (library, library.document!)
    }

    func testTheServersInkComesDownWithTheDocument() async {
        let fake = FakeService()
        let theirs = stroke(at: 10)
        fake.onDocumentInk = { _ in [1: PageInk(version: 3, inkB64: theirs.dataRepresentation().base64EncodedString())] }
        let (_, doc) = await open(fake)
        XCTAssertEqual(doc.ink[1]?.strokes.count, 1)
        XCTAssertEqual(doc.inkVersions[1], 3)
        XCTAssertNil(doc.ink[0])
    }

    func testAStrokeGoesUpWithThePagesVersion() async {
        let fake = FakeService()
        fake.onDocumentInk = { _ in [0: PageInk(version: 2, inkB64: PKDrawing().dataRepresentation().base64EncodedString())] }
        let (library, doc) = await open(fake)
        doc.drew(on: 0, stroke(at: 10))
        try? await Task.sleep(nanoseconds: 150_000_000)
        XCTAssertEqual(fake.inkPuts.map(\.page), [0])
        XCTAssertEqual(fake.inkPuts.first?.base, 2)
        XCTAssertEqual(doc.inkVersions[0], 3, "the stored version is kept for the next put")
        let sent = fake.inkPuts.first.flatMap { Data(base64Encoded: $0.inkB64) }.flatMap { try? PKDrawing(data: $0) }
        XCTAssertEqual(sent?.strokes.count, 1)
        XCTAssertNotNil(library.document, "the library lives as long as the reader")
    }

    func testAPageBothDrewOnKeepsBothDrawings() async {
        let fake = FakeService()
        let theirs = stroke(at: 200)
        var conflicted = false
        fake.onPutDocumentInk = { _, ink, base in
            if !conflicted {
                conflicted = true
                return .conflict(server: PageInk(version: 5, inkB64: theirs.dataRepresentation().base64EncodedString()))
            }
            return .stored(PageInk(version: base + 1, inkB64: ink))
        }
        let (library, doc) = await open(fake)
        doc.drew(on: 1, stroke(at: 10))
        try? await Task.sleep(nanoseconds: 150_000_000)
        XCTAssertEqual(fake.inkPuts.count, 2, "the merge is put back on top of the server's version")
        XCTAssertEqual(fake.inkPuts.last?.base, 5)
        XCTAssertEqual(doc.ink[1]?.strokes.count, 2)
        XCTAssertEqual(doc.inkVersions[1], 6)
        XCTAssertNotNil(library.document)
    }
}
