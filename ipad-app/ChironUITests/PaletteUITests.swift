import XCTest

/// Taps, as a finger or Pencil would make them, against the running app.
/// State is read back through the debug harness on localhost:8087, so an
/// assertion is about what the reader actually holds, not what the screen
/// suggests.
final class PaletteUITests: XCTestCase {
    private let harness = URL(string: "http://localhost:8087")!

    override func setUp() {
        continueAfterFailure = false
    }

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

    func testTappingAColourDotChangesThePen() throws {
        let app = XCUIApplication()
        app.launchArguments = ["harness"]
        app.launchEnvironment["CHIRON_SERVER"] = ProcessInfo.processInfo.environment["CHIRON_SERVER"] ?? "http://localhost:8084"
        app.launch()
        // The harness comes up with the app; open a book the way the shelf does.
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, (try? state()) == nil { Thread.sleep(forTimeInterval: 0.5) }
        try post("open", ["subject": "ai"])
        let pen = app.buttons["Pen"]
        XCTAssertTrue(pen.waitForExistence(timeout: 15), "the palette's pen button")
        let contents = app.buttons["Contents"]
        XCTAssertTrue(contents.waitForExistence(timeout: 5))
        contents.tap()
        Thread.sleep(forTimeInterval: 0.7)
        let afterContents = try state()
        contents.tap()
        Thread.sleep(forTimeInterval: 0.5)
        pen.tap()
        Thread.sleep(forTimeInterval: 0.7)
        var afterPen = try state()
        if afterPen["tool"] as? String != "pen" {
            pen.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.5)).tap()
            Thread.sleep(forTimeInterval: 0.7)
            afterPen = try state()
        }
        XCTAssertEqual(afterContents["contents"] as? Bool, true,
                       "the Contents chrome button toggles; pen frame \(pen.frame) hittable=\(pen.isHittable) enabled=\(pen.isEnabled)")
        let blue = app.buttons["blue pen"]
        let anyBlue = app.descendants(matching: .any)["blue pen"]
        XCTAssertTrue(blue.waitForExistence(timeout: 5),
                      "the colour row appears under the pen; tool=\(afterPen["tool"] ?? "?") anyElementNamedBluePen=\(anyBlue.exists) toolsElementExists=\(app.descendants(matching: .any)["Tools"].exists) penButtons=\(app.buttons.matching(NSPredicate(format: "label CONTAINS[c] 'pen'")).count)")
        blue.tap()
        Thread.sleep(forTimeInterval: 0.7)
        let s = try state()
        XCTAssertEqual(s["tool"] as? String, "pen")
        XCTAssertEqual(s["pen_color"] as? String, "blue")
        XCTAssertEqual(s["canvas_pen"] as? String, "pen rgb(0,136,255) w2.5", "the canvas holds the tapped colour")
        app.buttons["black pen"].tap()
        Thread.sleep(forTimeInterval: 0.7)
        XCTAssertEqual(try state()["canvas_pen"] as? String, "pen rgb(0,0,0) w2.5")

        // The ask card's own buttons take taps too.
        try post("tool", ["tool": "ask"])
        try post("mark", ["text": "predict the next token", "kind": "question"])
        let close = app.buttons["Close"]
        XCTAssertTrue(close.waitForExistence(timeout: 5), "the ask card opened on the question mark")
        close.tap()
        Thread.sleep(forTimeInterval: 0.7)
        XCTAssertNil(try state()["asking"], "the card closed on its own button")
    }
}

/// The ink layer over the page: a finger scrolls the page while the pen is
/// up, a stroke lands as ink, and the highlighter's drag still selects.
final class InkUITests: XCTestCase {
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

    func testFingersScrollDrawAndHighlightOverThePage() throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["harness"]
        app.launchEnvironment["CHIRON_SERVER"] = ProcessInfo.processInfo.environment["CHIRON_SERVER"] ?? "http://localhost:8084"
        app.launch()
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, (try? state()) == nil { Thread.sleep(forTimeInterval: 0.5) }
        try post("open", ["subject": "ai"])
        XCTAssertTrue(app.buttons["Pen"].waitForExistence(timeout: 15))
        Thread.sleep(forTimeInterval: 2)

        // With the pen up, a finger drag draws (no Pencil in the Simulator).
        try post("tool", ["tool": "pen"])
        Thread.sleep(forTimeInterval: 0.5)
        let page = app.webViews.firstMatch
        XCTAssertTrue(page.waitForExistence(timeout: 10))
        let strokesBefore = try state()["ink_strokes"] as? Int ?? 0
        let from = page.coordinate(withNormalizedOffset: CGVector(dx: 0.3, dy: 0.5))
        let to = page.coordinate(withNormalizedOffset: CGVector(dx: 0.6, dy: 0.55))
        from.press(forDuration: 0.1, thenDragTo: to, withVelocity: .slow, thenHoldForDuration: 0.1)
        Thread.sleep(forTimeInterval: 1.0)
        let drawn = try state()
        XCTAssertEqual(drawn["ink_strokes"] as? Int, strokesBefore + 1,
                       "one stroke of ink; canvas touches=\(drawn["canvas_touches"] ?? "?") frame=\(drawn["canvas_frame"] ?? "?") pen=\(drawn["canvas_pen"] ?? "?")")

        // The highlighter's drag selects a run of text.
        try post("tool", ["tool": "highlighter"])
        Thread.sleep(forTimeInterval: 0.5)
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.2, dy: 0.4))
            .press(forDuration: 0.05, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.4)))
        Thread.sleep(forTimeInterval: 1.0)
        let marks = try state()["marks"] as? [[String: Any]] ?? []
        XCTAssertTrue(marks.contains { $0["kind"] as? String == "highlight" }, "a highlight from the drag: \(marks)")
        // With no tool, a finger swipe scrolls the page.
        try post("tool", ["tool": "none"])
        Thread.sleep(forTimeInterval: 0.5)
        let before = try state()["position"] as? Double ?? 0
        page.swipeUp()
        Thread.sleep(forTimeInterval: 1.5)
        let after = try state()["position"] as? Double ?? 0
        XCTAssertGreaterThan(after, before, "the page scrolled")

    }
}
