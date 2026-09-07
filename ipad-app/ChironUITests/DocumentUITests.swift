import XCTest

/// A PDF on the shelf under real touches: with the pen up a drag lands as
/// ink on the page (a finger stands in for the Pencil in the Simulator),
/// and with the select tool a long press selects text. Both go through
/// PDFKit's hit-testing, which is what the harness's own stroke verb
/// bypasses.
final class DocumentUITests: XCTestCase {
    private let harness = URL(string: "http://localhost:8087")!

    private func state() throws -> [String: Any] {
        let data = try Data(contentsOf: harness.appendingPathComponent("state"))
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func post(_ path: String, _ body: [String: Any]) throws {
        var req = URLRequest(url: harness.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        let done = expectation(description: path)
        URLSession.shared.dataTask(with: req) { _, _, _ in done.fulfill() }.resume()
        wait(for: [done], timeout: 30)
    }

    private func document() throws -> [String: Any]? { try state()["document"] as? [String: Any] }

    func testInkAndSelectionOnAnImportedPDF() throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["harness"]
        app.launchEnvironment["CHIRON_SERVER"] = ProcessInfo.processInfo.environment["CHIRON_SERVER"] ?? "http://localhost:8084"
        app.launch()
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, (try? state()) == nil { Thread.sleep(forTimeInterval: 0.5) }

        let fixture = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent()
            .appendingPathComponent("fixtures/prose.pdf").path
        try post("import", ["path": fixture])
        let rows = try state()["shelf_rows"] as? [[String: Any]] ?? []
        let doc = try XCTUnwrap(rows.last { $0["kind"] as? String == "pdf" }?["id"] as? String,
                                "the imported PDF is on the shelf: \(rows)")
        try post("open", ["subject": doc])
        let select = app.buttons["Select"]
        XCTAssertTrue(select.waitForExistence(timeout: 15), "the document's toolbar")
        let opened = Date().addingTimeInterval(10)
        while Date() < opened, ((try? document())?["overlaid_pages"] as? [Int] ?? []).isEmpty { Thread.sleep(forTimeInterval: 0.5) }
        XCTAssertEqual(try document()?["tool"] as? String, "pen", "a document opens ready to mark")

        // Pen up: a drag across the page is a stroke of ink, not a scroll.
        let page = app.windows.firstMatch
        let before = try document()?["ink_strokes"] as? Int ?? 0
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.3, dy: 0.5))
            .press(forDuration: 0.1, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.6, dy: 0.55)),
                   withVelocity: .slow, thenHoldForDuration: 0.1)
        Thread.sleep(forTimeInterval: 1.0)
        let drawn = try document()
        XCTAssertEqual(drawn?["ink_strokes"] as? Int, before + 1, "one stroke of ink: \(drawn ?? [:])")

        // Select: a long press on the text selects a word, and draws nothing.
        select.tap()
        Thread.sleep(forTimeInterval: 0.5)
        XCTAssertEqual(try document()?["tool"] as? String, "select")
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.4, dy: 0.45)).press(forDuration: 1.2)
        Thread.sleep(forTimeInterval: 1.0)
        let selected = try document()
        XCTAssertFalse((selected?["selection"] as? String ?? "").isEmpty, "a long press selected text: \(selected ?? [:])")
        XCTAssertEqual(selected?["ink_strokes"] as? Int, before + 1, "the select tool does not draw")

        // Back to the pen from the toolbar; the eraser too.
        app.buttons["Eraser"].tap()
        Thread.sleep(forTimeInterval: 0.5)
        XCTAssertEqual(try document()?["tool"] as? String, "eraser")
    }
}
