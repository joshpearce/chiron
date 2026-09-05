import XCTest
@testable import Chiron

/// Primers on the shelf and in a session: the wire shape a server sends,
/// and the margin-note loop against a fake server.
@MainActor
final class PrimerTests: XCTestCase {
    func testShelfRowsDecodeKindStatusAndSource() throws {
        let json = """
        {"subjects":[
          {"id":"ai","title":"How AI Works","kind":"book","units_total":11,"units_cleared":1,"current_unit":"u1","debt":0},
          {"id":"primer-robots","title":"Robots and money","kind":"primer","units_total":1,"units_cleared":0,"current_unit":null,"debt":0,
           "status":"ready","source":{"text":"User-agent: *","url":"https://lexweekly.example/robots.txt","app":"Safari"},"captured_at":"2026-09-03T12:00:00Z"},
          {"id":"primer-x","title":"What is this","kind":"primer","units_total":0,"units_cleared":0,"current_unit":null,"debt":0,
           "status":"authoring","source":{"text":"words"},"captured_at":"2026-09-03T12:00:00Z"},
          {"id":"primer-y","title":"Why","kind":"primer","units_total":0,"units_cleared":0,"current_unit":null,"debt":0,
           "status":"failed","error":"no model","captured_at":"2026-09-03T12:00:00Z"}
        ],"active":"ai"}
        """
        let shelf = try JSONDecoder().decode(SubjectsResponse.self, from: Data(json.utf8))
        XCTAssertEqual(shelf.subjects.count, 4)
        let book = shelf.subjects[0], ready = shelf.subjects[1], authoring = shelf.subjects[2], failed = shelf.subjects[3]
        XCTAssertFalse(book.isPrimer)
        XCTAssertEqual(book.progressLine, "1/11 units · reading u1")
        XCTAssertTrue(ready.isPrimer)
        XCTAssertFalse(ready.authoring)
        XCTAssertEqual(ready.sourceLine, "from Safari · Sep 3")
        XCTAssertEqual(ready.progressLine, ready.sourceLine)
        XCTAssertTrue(authoring.authoring)
        XCTAssertEqual(authoring.sourceLine, "captured · Sep 3")
        XCTAssertTrue(failed.failed)
        XCTAssertEqual(failed.error, "no model")
    }

    func testAnOldServerRowStillDecodes() throws {
        let json = #"{"subjects":[{"id":"ai","title":"AI","units_total":3,"units_cleared":0,"current_unit":null,"debt":0}],"active":""}"#
        let shelf = try JSONDecoder().decode(SubjectsResponse.self, from: Data(json.utf8))
        XCTAssertFalse(shelf.subjects[0].isPrimer)
        XCTAssertNil(shelf.activeID)
    }

    func testCaptureRequestWireNames() throws {
        var req = CaptureRequest(prompt: "why?")
        req.text = "words"
        req.sourceApp = "Safari"
        req.imagePngB64 = "AAAA"
        let obj = try JSONSerialization.jsonObject(with: JSONEncoder().encode(req)) as! [String: Any]
        XCTAssertEqual(obj["source_app"] as? String, "Safari")
        XCTAssertEqual(obj["image_png_b64"] as? String, "AAAA")
        XCTAssertEqual(obj["prompt"] as? String, "why?")
        XCTAssertEqual(obj["scale"] as? String, "primer", "a primer unless the reader picks otherwise")
    }

    func testCaptureInboxRoundTrip() throws {
        var c = Capture(text: "captured words", sourceApp: "Notes")
        c.imagePNG = Data([1, 2, 3])
        try CaptureInbox.write(c)
        let url = CaptureInbox.url(for: c.id)
        XCTAssertEqual(CaptureInbox.captureID(in: url), c.id)
        XCTAssertNil(CaptureInbox.captureID(in: URL(string: "https://example.com/capture/x")!))
        let taken = CaptureInbox.take(c.id)
        XCTAssertEqual(taken, c)
        XCTAssertNil(CaptureInbox.take(c.id), "a capture is delivered once")
    }

    private func primerSession(_ fake: FakeService) -> BookSession {
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let s = BookSession(subjectID: "primer-robots", title: "Robots and money", service: fake, storage: storage)
        s.kind = "primer"
        return s
    }

    private func chapter(html: String) -> ChapterPayload {
        let json = """
        {"unit":"p1","title":"Robots and money","minutes":10,"html":"\(html)","beats":[],"pretest":[],"check":[],"next_action":"read"}
        """
        return try! JSONDecoder().decode(ChapterPayload.self, from: Data(json.utf8))
    }

    func testAMarginNoteExtendsThePrimer() async throws {
        let fake = FakeService()
        let s = primerSession(fake)
        XCTAssertTrue(s.isPrimer)
        XCTAssertNil(s.chapter)
        s.setChapterForTesting(chapter(html: "<p>Cheques get signed.</p>"))
        fake.onExtend = { _, quote, note in
            XCTAssertEqual(quote, "Cheques get signed.")
            XCTAssertEqual(note, "why 402?")
            return ExtendResponse(chapter: self.chapter(html: "<p>Cheques get signed.</p><h2>Why 402</h2>"), heading: "Why 402", entries: 1)
        }
        // The note tool marks the passage and opens the card.
        let mark = s.addMark(kind: .note, start: 3, end: 22, text: "Cheques get signed.")
        XCTAssertEqual(s.asking?.mark.id, mark.id)
        await s.extend("why 402?")
        XCTAssertEqual(fake.extends.count, 1)
        XCTAssertEqual(fake.extends[0].subject, "primer-robots")
        XCTAssertEqual(s.asking?.mark.answer, "Why 402")
        XCTAssertTrue(s.chapter?.html.contains("Why 402") == true, "the document came back whole")
        XCTAssertEqual(s.marks.first?.question, "why 402?")

        // Only the note tool appears for primers, and the squeeze cycle knows it.
        s.tool = .ask
        s.cycleTool()
        XCTAssertEqual(s.tool, .note)
        s.cycleTool()
        XCTAssertEqual(s.tool, .capture)
    }

    func testAFailedExtendKeepsTheNoteAndSaysSo() async {
        let fake = FakeService()
        let s = primerSession(fake)
        s.setChapterForTesting(chapter(html: "<p>Words.</p>"))
        _ = s.addMark(kind: .note, start: 0, end: 6, text: "Words.")
        await s.extend("more")
        XCTAssertNotNil(s.asking?.error)
        XCTAssertEqual(s.asking?.mark.question, "more")
        XCTAssertNil(s.asking?.mark.answer)
    }
}
