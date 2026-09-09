import XCTest

/// A PDF on the shelf under real touches: with the pen up a drag lands as
/// ink on the page (a finger stands in for the Pencil in the Simulator),
/// and with the select tool a long press selects text. Both go through
/// PDFKit's hit-testing, which is what the harness's own stroke verb
/// bypasses.
final class DocumentUITests: HarnessTestCase {
    func testInkAndSelectionOnAnImportedPDF() throws {
        _ = app
        let fixture = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent()
            .appendingPathComponent("fixtures/prose.pdf").path
        try post("import", ["path": fixture])
        let rows = try state()["shelf_rows"] as? [[String: Any]] ?? []
        let doc = try XCTUnwrap(rows.last { $0["kind"] as? String == "pdf" }?["id"] as? String,
                                "the imported PDF is on the shelf: \(rows)")
        try post("open", ["subject": doc])
        let select = app.buttons["Select"]
        XCTAssertTrue(select.waitForExistence(timeout: 15), "the document's toolbar")
        try waitUntil("the page's ink overlay") { !((try self.document()["overlaid_pages"] as? [Int] ?? []).isEmpty) }
        XCTAssertEqual(try document()["tool"] as? String, "pen", "a document opens ready to mark")

        // Pen up: a drag across the page is a stroke of ink, not a scroll.
        let page = app.windows.firstMatch
        let before = try document()["ink_strokes"] as? Int ?? 0
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.3, dy: 0.5))
            .press(forDuration: 0.1, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.6, dy: 0.55)),
                   withVelocity: .slow, thenHoldForDuration: 0.1)
        try waitUntil("one stroke of ink") { (try self.document()["ink_strokes"] as? Int) == before + 1 }
        app.buttons["Undo"].tap()
        try waitUntil("undo to take the stroke back") { (try self.document()["ink_strokes"] as? Int) == before }
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.3, dy: 0.6))
            .press(forDuration: 0.1, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.6, dy: 0.65)),
                   withVelocity: .slow, thenHoldForDuration: 0.1)
        try waitUntil("the stroke drawn again") { (try self.document()["ink_strokes"] as? Int) == before + 1 }

        // Select: a long press on the text selects a word, and draws nothing.
        select.tap()
        try waitUntil("the select tool") { (try self.document()["tool"] as? String) == "select" }
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.4, dy: 0.45)).press(forDuration: 1.2)
        try waitUntil("a long press to select text") { !((try self.document()["selection"] as? String ?? "").isEmpty) }
        XCTAssertEqual(try document()["ink_strokes"] as? Int, before + 1, "the select tool does not draw")

        // Back to the pen from the toolbar; the eraser too.
        app.buttons["Eraser"].tap()
        try waitUntil("the eraser") { (try self.document()["tool"] as? String) == "eraser" }
    }
}
