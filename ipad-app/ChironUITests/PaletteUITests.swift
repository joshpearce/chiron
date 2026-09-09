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
        // One dot of the pen's colour beside the pen; the row of colours
        // opens from it and folds back once a colour is picked.
        let colour = app.buttons["Pen colour"]
        XCTAssertTrue(colour.waitForExistence(timeout: 5),
                      "the colour dot appears beside the pen; tool=\(afterPen["tool"] ?? "?") toolsElementExists=\(app.descendants(matching: .any)["Tools"].exists) penButtons=\(app.buttons.matching(NSPredicate(format: "label CONTAINS[c] 'pen'")).count)")
        XCTAssertFalse(app.buttons["blue pen"].exists, "the row is folded until the dot is tapped")
        colour.tap()
        let blue = app.buttons["blue pen"]
        XCTAssertTrue(blue.waitForExistence(timeout: 5), "the colour row opens from the dot")
        blue.tap()
        Thread.sleep(forTimeInterval: 0.7)
        let s = try state()
        XCTAssertEqual(s["tool"] as? String, "pen")
        XCTAssertEqual(s["pen_color"] as? String, "blue")
        XCTAssertEqual(s["canvas_pen"] as? String, "pen rgb(0,136,255) w2.5", "the canvas holds the tapped colour")
        XCTAssertTrue(colour.waitForExistence(timeout: 5), "the row folded back into the dot")
        XCTAssertFalse(app.buttons["blue pen"].exists)
        XCTAssertEqual(colour.value as? String, "blue", "the dot shows the picked colour")
        colour.tap()
        let black = app.buttons["black pen"]
        XCTAssertTrue(black.waitForExistence(timeout: 5))
        black.tap()
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

    /// A line of JavaScript against the page, and what it came to.
    private func eval(_ js: String) throws -> String {
        try call("reader/eval", ["js": js])["result"] as? String ?? ""
    }

    private func rectOf(selector: String) throws -> CGRect {
        let out = try call("reader/rect", ["selector": selector])
        XCTAssertEqual(out["found"] as? Bool, true, "\(selector) is on the page: \(out)")
        return CGRect(x: out["x"] as? Double ?? 0, y: out["y"] as? Double ?? 0,
                      width: out["width"] as? Double ?? 0, height: out["height"] as? Double ?? 0)
    }

    private func call(_ path: String, _ body: [String: Any]) throws -> [String: Any] {
        var req = URLRequest(url: harness.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        var out: [String: Any] = [:]
        let done = expectation(description: path)
        URLSession.shared.dataTask(with: req) { data, _, _ in
            out = data.flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] } ?? [:]
            done.fulfill()
        }.resume()
        wait(for: [done], timeout: 30)
        return out
    }

    /// Where a mark's badge is, in the page view's points.
    private func rectOf(_ id: String) throws -> CGRect {
        var req = URLRequest(url: harness.appendingPathComponent("mark/rect"))
        req.httpMethod = "POST"
        req.httpBody = try JSONSerialization.data(withJSONObject: ["id": id])
        var out: [String: Any] = [:]
        let done = expectation(description: "mark/rect")
        URLSession.shared.dataTask(with: req) { data, _, _ in
            out = data.flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] } ?? [:]
            done.fulfill()
        }.resume()
        wait(for: [done], timeout: 30)
        XCTAssertEqual(out["found"] as? Bool, true, "the mark is on the page: \(out)")
        return CGRect(x: out["x"] as? Double ?? 0, y: out["y"] as? Double ?? 0,
                      width: out["width"] as? Double ?? 0, height: out["height"] as? Double ?? 0)
    }

    /// The keyboard is the only way into a beat's answer, so a tap on the
    /// field must bring it up. (The Simulator's hardware keyboard hides
    /// this: the run turns it off.)
    func testTappingABeatsFieldBringsUpTheKeyboard() throws {
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
        let page = app.webViews.firstMatch
        let selector = "input[inputmode=decimal]"

        // The field is a beat's only way in, with no tool up and with the
        // pen up after a stroke, when the ink layer holds the keyboard.
        for tool in ["none", "pen"] {
            try post("tool", ["tool": tool])
            Thread.sleep(forTimeInterval: 0.5)
            if tool == "pen" {
                page.coordinate(withNormalizedOffset: CGVector(dx: 0.3, dy: 0.4))
                    .press(forDuration: 0.1, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.6, dy: 0.45)),
                           withVelocity: .slow, thenHoldForDuration: 0.1)
                Thread.sleep(forTimeInterval: 1.0)
            }
            _ = try eval("document.querySelector('\(selector)').scrollIntoView({block: 'center'}); 'ok'")
            Thread.sleep(forTimeInterval: 1.0)
            let field = try rectOf(selector: selector)
            page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: field.midX, dy: field.midY)).tap()
            Thread.sleep(forTimeInterval: 1.5)
            XCTAssertEqual(try eval("document.activeElement.tagName"), "INPUT", "the field took focus with \(tool) up")
            // (keyboard_hardware is not asserted: the Simulator reports the
            // Mac's keyboard whether or not it is connected to it.)
            let s = try state()
            XCTAssertEqual(s["page_focus"] as? String, "input", "the page said what it focused")
            XCTAssertFalse((s["first_responder"] as? String ?? "").contains("Canvas"),
                           "the ink layer let the keyboard go: first responder is \(s["first_responder"] ?? "?")")
            XCTAssertTrue(app.keyboards.element.waitForExistence(timeout: 5),
                          "the keyboard came up with \(tool) up; \(s["first_responder"] ?? "?") holds it")
            app.typeText("7")
            Thread.sleep(forTimeInterval: 0.5)
            XCTAssertEqual(try eval("document.querySelector('\(selector)').value"), "7", "the typing landed with \(tool) up")
            _ = try eval("var i = document.querySelector('\(selector)'); i.value = ''; i.blur(); 'ok'")
            Thread.sleep(forTimeInterval: 1.0)
        }
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
        // Start clean: any marks from an earlier run go.
        for m in (try state()["marks"] as? [[String: Any]]) ?? [] {
            if let id = m["id"] as? String { try post("unmark", ["id": id]) }
        }

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
        XCTAssertEqual(drawn["canvas_hit"] as? String, "canvas", "a touch on the prose is the canvas's")

        // With the pen still up, a beat's field takes a tap and typing:
        // the canvas hands a touch over a control through to the page.
        _ = try eval("document.querySelector('textarea, input').scrollIntoView({block: 'center'}); 'ok'")
        Thread.sleep(forTimeInterval: 1.0)
        let field = try rectOf(selector: "textarea, input")
        page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: field.midX, dy: field.midY)).tap()
        Thread.sleep(forTimeInterval: 1.0)
        XCTAssertEqual(try state()["canvas_hit"] as? String, "page", "the touch over the field went to the page")
        XCTAssertTrue(["TEXTAREA", "INPUT"].contains(try eval("document.activeElement.tagName")), "the field took focus")
        app.typeText("3")
        Thread.sleep(forTimeInterval: 0.5)
        XCTAssertEqual(try eval("document.querySelector('textarea, input').value"), "3", "and the typing")
        XCTAssertEqual(try state()["ink_strokes"] as? Int, strokesBefore + 1, "no ink from the tap")
        _ = try eval("document.activeElement.blur(); window.scrollTo(0, 0); 'ok'")
        Thread.sleep(forTimeInterval: 1.0)

        // The highlighter's drag selects a run of text.
        try post("tool", ["tool": "highlighter"])
        Thread.sleep(forTimeInterval: 0.5)
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.2, dy: 0.4))
            .press(forDuration: 0.05, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.4)))
        Thread.sleep(forTimeInterval: 1.0)
        let marks = try state()["marks"] as? [[String: Any]] ?? []
        XCTAssertTrue(marks.contains { $0["kind"] as? String == "highlight" }, "a highlight from the drag: \(marks)")

        // Undo takes the highlight back, then the stroke.
        let undo = app.buttons["Undo"]
        XCTAssertTrue(undo.waitForExistence(timeout: 5))
        undo.tap()
        Thread.sleep(forTimeInterval: 0.7)
        var s = try state()
        XCTAssertFalse((s["marks"] as? [[String: Any]] ?? []).contains { $0["kind"] as? String == "highlight" }, "the highlight came back off")
        XCTAssertEqual(s["ink_strokes"] as? Int, strokesBefore + 1, "the ink stayed")
        undo.tap()
        Thread.sleep(forTimeInterval: 0.7)
        XCTAssertEqual(try state()["ink_strokes"] as? Int, strokesBefore, "then the stroke went")

        // With no tool, a tap on a highlight offers to remove it.
        try post("tool", ["tool": "none"])
        try post("mark", ["text": "predict the next token", "kind": "highlight"])
        Thread.sleep(forTimeInterval: 0.5)
        s = try state()
        let highlight = try XCTUnwrap((s["marks"] as? [[String: Any]])?.first { $0["kind"] as? String == "highlight" }?["id"] as? String)
        var rect = try rectOf(highlight)
        page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: rect.midX, dy: rect.midY)).tap()
        let remove = app.buttons["Remove highlight"]
        if !remove.waitForExistence(timeout: 5) {
            let shot = XCTAttachment(screenshot: app.screenshot())
            shot.lifetime = .keepAlways
            add(shot)
            XCTFail("the tap at \(rect) asked about the highlight; state \(try state())")
        }
        remove.tap()
        Thread.sleep(forTimeInterval: 0.7)
        XCTAssertFalse((try state()["marks"] as? [[String: Any]] ?? []).contains { $0["id"] as? String == highlight }, "the highlight went")

        // The eraser over a highlight takes it off too.
        try post("mark", ["text": "predict the next token", "kind": "highlight"])
        Thread.sleep(forTimeInterval: 0.5)
        let again = try XCTUnwrap((try state()["marks"] as? [[String: Any]])?.first { $0["kind"] as? String == "highlight" }?["id"] as? String)
        try post("tool", ["tool": "eraser"])
        Thread.sleep(forTimeInterval: 0.5)
        rect = try rectOf(again)
        page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: rect.midX, dy: rect.midY)).tap()
        Thread.sleep(forTimeInterval: 0.7)
        XCTAssertFalse((try state()["marks"] as? [[String: Any]] ?? []).contains { $0["id"] as? String == again }, "the eraser rubbed the highlight out")

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

/// A question's life on the page: asked, closed with its badge left
/// behind, reopened by tapping the badge, and deleted from the card.
final class AskCardUITests: XCTestCase {
    private let harness = URL(string: "http://localhost:8087")!

    private func state() throws -> [String: Any] {
        let data = try Data(contentsOf: harness.appendingPathComponent("state"))
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func post(_ path: String, _ body: [String: Any]) throws -> [String: Any] {
        var req = URLRequest(url: harness.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        var out: [String: Any] = [:]
        let done = expectation(description: path)
        URLSession.shared.dataTask(with: req) { data, _, _ in
            out = data.flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] } ?? [:]
            done.fulfill()
        }.resume()
        wait(for: [done], timeout: 60)
        return out
    }

    func testBadgeReopensAndDeleteRemoves() throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["harness"]
        app.launchEnvironment["CHIRON_SERVER"] = ProcessInfo.processInfo.environment["CHIRON_SERVER"] ?? "http://localhost:8084"
        app.launch()
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, (try? state()) == nil { Thread.sleep(forTimeInterval: 0.5) }
        _ = try post("open", ["subject": "ai"])
        XCTAssertTrue(app.buttons["Pen"].waitForExistence(timeout: 15))
        Thread.sleep(forTimeInterval: 2)
        // Start clean: any marks from an earlier run go.
        for m in (try state()["marks"] as? [[String: Any]]) ?? [] {
            if let id = m["id"] as? String { _ = try post("unmark", ["id": id]) }
        }

        let asked = try post("ask", ["text": "at sufficient scale", "question": "What counts as sufficient scale?"])
        let asking = try XCTUnwrap(asked["asking"] as? [String: Any])
        XCTAssertEqual(asking["answered"] as? Bool, true)
        XCTAssertTrue(app.buttons["Close"].waitForExistence(timeout: 5))
        app.buttons["Close"].tap()
        Thread.sleep(forTimeInterval: 0.5)
        var s = try state()
        XCTAssertNil(s["asking"], "the card closed")
        let marks = try XCTUnwrap(s["marks"] as? [[String: Any]])
        XCTAssertEqual(marks.count, 1, "the highlight stayed")
        let id = try XCTUnwrap(marks[0]["id"] as? String)

        // Tap the badge on the page as a finger would.
        let rect = try post("mark/rect", ["id": id])
        XCTAssertEqual(rect["found"] as? Bool, true, "the mark is on the page: \(rect)")
        let page = app.webViews.firstMatch
        let x = (rect["x"] as? Double ?? 0) + (rect["width"] as? Double ?? 0) - 6
        let y = (rect["y"] as? Double ?? 0) + (rect["height"] as? Double ?? 0) / 2
        page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: x, dy: y)).tap()
        Thread.sleep(forTimeInterval: 0.8)
        s = try state()
        XCTAssertNotNil(s["asking"], "tapping the badge reopened the card")
        XCTAssertTrue(app.buttons["Delete"].waitForExistence(timeout: 5))
        app.buttons["Delete"].tap()
        // The bin asks first, in a popover; on iPad a tap outside keeps it.
        let confirm = app.buttons["Delete the question and its highlight"]
        XCTAssertTrue(confirm.waitForExistence(timeout: 5), "the confirmation appeared")
        app.otherElements["PopoverDismissRegion"].tap()
        Thread.sleep(forTimeInterval: 0.7)
        s = try state()
        XCTAssertNotNil(s["asking"], "kept: the card is still open")
        XCTAssertEqual((s["marks"] as? [[String: Any]])?.count, 1)
        app.buttons["Delete"].tap()
        XCTAssertTrue(confirm.waitForExistence(timeout: 5))
        confirm.tap()
        Thread.sleep(forTimeInterval: 0.5)
        s = try state()
        XCTAssertNil(s["asking"])
        XCTAssertEqual((s["marks"] as? [[String: Any]])?.count, 0, "delete removed the question and its highlight")
    }
}
