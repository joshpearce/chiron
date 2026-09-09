import XCTest

/// Taps, as a finger or Pencil would make them, against the running app.
/// State is read back through the debug harness, so an assertion is about
/// what the reader actually holds, not what the screen suggests.
final class PaletteUITests: HarnessTestCase {
    func testTappingAColourDotChangesThePen() throws {
        try openBook()
        let pen = app.buttons["Pen"]
        let contents = app.buttons["Contents"]
        XCTAssertTrue(contents.waitForExistence(timeout: 5))
        contents.tap()
        try waitUntil("the contents to open") { (try self.state()["contents"] as? Bool) == true }
        contents.tap()
        try waitUntil("the contents to close") { (try self.state()["contents"] as? Bool) == false }

        // A tap that lands on the glass edge does nothing; if the pen did
        // not come up, aim at the middle of it.
        pen.tap()
        if try !settles(4, { (try? self.state()["tool"] as? String) == "pen" }) {
            pen.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.5)).tap()
            try waitFor("tool", "pen")
        }

        // One dot of the pen's colour beside the pen; the row of colours
        // opens from it and folds back once a colour is picked.
        let colour = app.buttons["Pen color"]
        XCTAssertTrue(colour.waitForExistence(timeout: 5),
                      "the colour dot appears beside the pen; penButtons=\(app.buttons.matching(NSPredicate(format: "label CONTAINS[c] 'pen'")).count)")
        XCTAssertFalse(app.buttons["blue pen"].exists, "the row is folded until the dot is tapped")
        colour.tap()
        let blue = app.buttons["blue pen"]
        XCTAssertTrue(blue.waitForExistence(timeout: 5), "the colour row opens from the dot")
        blue.tap()
        try waitFor("pen_color", "blue")
        let s = try state()
        XCTAssertEqual(s["tool"] as? String, "pen")
        XCTAssertEqual(s["canvas_pen"] as? String, "pen rgb(0,136,255) w2.5", "the canvas holds the tapped colour")
        XCTAssertTrue(colour.waitForExistence(timeout: 5), "the row folded back into the dot")
        XCTAssertFalse(app.buttons["blue pen"].exists)
        XCTAssertEqual(colour.value as? String, "blue", "the dot shows the picked colour")
        colour.tap()
        let black = app.buttons["black pen"]
        XCTAssertTrue(black.waitForExistence(timeout: 5))
        black.tap()
        try waitFor("canvas_pen", "pen rgb(0,0,0) w2.5")

        // The ask card's own buttons take taps too.
        try post("tool", ["tool": "ask"])
        try post("mark", ["text": "predict the next token", "kind": "question"])
        let close = app.buttons["Close"]
        XCTAssertTrue(close.waitForExistence(timeout: 5), "the ask card opened on the question mark")
        close.tap()
        try waitUntil("the card to close") { try self.state()["asking"] == nil }
    }
}

/// The ink layer over the page: a finger scrolls the page while the pen is
/// up, a stroke lands as ink, and the highlighter's drag still selects.
final class InkUITests: HarnessTestCase {
    /// The keyboard is the only way into a beat's answer, so a tap on the
    /// field must bring it up. (The Simulator's hardware keyboard hides
    /// this: the run turns it off.)
    func testTappingABeatsFieldBringsUpTheKeyboard() throws {
        try openBook()
        let page = app.webViews.firstMatch
        let selector = "input[inputmode=decimal]"

        // The field is a beat's only way in, with no tool up and with the
        // pen up after a stroke, when the ink layer holds the keyboard.
        for tool in ["none", "pen"] {
            try post("tool", ["tool": tool])
            if tool == "pen" {
                page.coordinate(withNormalizedOffset: CGVector(dx: 0.3, dy: 0.4))
                    .press(forDuration: 0.1, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.6, dy: 0.45)),
                           withVelocity: .slow, thenHoldForDuration: 0.1)
                try waitUntil("the stroke to land") { (try self.state()["ink_strokes"] as? Int ?? 0) > 0 }
            }
            try eval("document.querySelector('\(selector)').scrollIntoView({block: 'center'}); 'ok'")
            let field = try rect(selector: selector)
            page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: field.midX, dy: field.midY)).tap()
            try waitUntil("the field to take focus with \(tool) up") { try self.eval("document.activeElement.tagName") == "INPUT" }
            // (keyboard_hardware is not asserted: the Simulator reports the
            // Mac's keyboard whether or not it is connected to it.)
            let s = try state()
            XCTAssertEqual(s["page_focus"] as? String, "input", "the page said what it focused")
            XCTAssertFalse((s["first_responder"] as? String ?? "").contains("Canvas"),
                           "the ink layer let the keyboard go: first responder is \(s["first_responder"] ?? "?")")
            XCTAssertTrue(app.keyboards.element.waitForExistence(timeout: 5),
                          "the keyboard came up with \(tool) up; \(s["first_responder"] ?? "?") holds it")
            app.typeText("7")
            try waitUntil("the typing to land with \(tool) up") { try self.eval("document.querySelector('\(selector)').value") == "7" }
            try eval("var i = document.querySelector('\(selector)'); i.value = ''; i.blur(); 'ok'")
        }
    }

    func testFingersScrollDrawAndHighlightOverThePage() throws {
        try openBook()
        // With the pen up, a finger drag draws (no Pencil in the Simulator).
        try post("tool", ["tool": "pen"])
        let page = app.webViews.firstMatch
        XCTAssertTrue(page.waitForExistence(timeout: 10))
        let strokesBefore = try state()["ink_strokes"] as? Int ?? 0
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.3, dy: 0.5))
            .press(forDuration: 0.1, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.6, dy: 0.55)),
                   withVelocity: .slow, thenHoldForDuration: 0.1)
        try waitUntil("one stroke of ink") { (try self.state()["ink_strokes"] as? Int) == strokesBefore + 1 }
        XCTAssertEqual(try state()["canvas_hit"] as? String, "canvas", "a touch on the prose is the canvas's")

        // With the pen still up, a beat's field takes a tap and typing:
        // the canvas hands a touch over a control through to the page.
        try eval("document.querySelector('textarea, input').scrollIntoView({block: 'center'}); 'ok'")
        let field = try rect(selector: "textarea, input")
        page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: field.midX, dy: field.midY)).tap()
        try waitUntil("the touch over the field to go to the page") { (try self.state()["canvas_hit"] as? String) == "page" }
        XCTAssertTrue(["TEXTAREA", "INPUT"].contains(try eval("document.activeElement.tagName")), "the field took focus")
        app.typeText("3")
        try waitUntil("the typing to land") { try self.eval("document.querySelector('textarea, input').value") == "3" }
        XCTAssertEqual(try state()["ink_strokes"] as? Int, strokesBefore + 1, "no ink from the tap")
        try eval("document.activeElement.blur(); window.scrollTo(0, 0); 'ok'")

        // The highlighter's drag selects a run of text.
        try post("tool", ["tool": "highlighter"])
        page.coordinate(withNormalizedOffset: CGVector(dx: 0.2, dy: 0.4))
            .press(forDuration: 0.05, thenDragTo: page.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.4)))
        try waitUntil("a highlight from the drag") { try self.marks().contains { $0["kind"] as? String == "highlight" } }

        // Undo takes the highlight back, then the stroke.
        let undo = app.buttons["Undo"]
        XCTAssertTrue(undo.waitForExistence(timeout: 5))
        undo.tap()
        try waitUntil("the highlight to come back off") { try !self.marks().contains { $0["kind"] as? String == "highlight" } }
        XCTAssertEqual(try state()["ink_strokes"] as? Int, strokesBefore + 1, "the ink stayed")
        undo.tap()
        try waitUntil("the stroke to go") { (try self.state()["ink_strokes"] as? Int) == strokesBefore }

        // With no tool, a tap on a highlight offers to remove it.
        try post("tool", ["tool": "none"])
        try post("mark", ["text": "predict the next token", "kind": "highlight"])
        let highlight = try XCTUnwrap(try marks().first { $0["kind"] as? String == "highlight" }?["id"] as? String)
        var box = try rect(markID: highlight)
        page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: box.midX, dy: box.midY)).tap()
        let remove = app.buttons["Remove highlight"]
        if !remove.waitForExistence(timeout: 5) {
            let shot = XCTAttachment(screenshot: app.screenshot())
            shot.lifetime = .keepAlways
            add(shot)
            XCTFail("the tap at \(box) asked about the highlight; state \(try state())")
        }
        remove.tap()
        try waitUntil("the highlight to go") { try !self.marks().contains { $0["id"] as? String == highlight } }

        // The eraser over a highlight takes it off too.
        try post("mark", ["text": "predict the next token", "kind": "highlight"])
        let again = try XCTUnwrap(try marks().first { $0["kind"] as? String == "highlight" }?["id"] as? String)
        try post("tool", ["tool": "eraser"])
        box = try rect(markID: again)
        page.coordinate(withNormalizedOffset: .zero).withOffset(CGVector(dx: box.midX, dy: box.midY)).tap()
        try waitUntil("the eraser to rub the highlight out") { try !self.marks().contains { $0["id"] as? String == again } }

        // With no tool, a finger swipe scrolls the page.
        try post("tool", ["tool": "none"])
        let before = try state()["position"] as? Double ?? 0
        page.swipeUp()
        try waitUntil("the page to scroll") { (try self.state()["position"] as? Double ?? 0) > before }
    }
}

/// A question's life on the page: asked, closed with its badge left
/// behind, reopened by tapping the badge, and deleted from the card.
final class AskCardUITests: HarnessTestCase {
    func testBadgeReopensAndDeleteRemoves() throws {
        try openBook()
        let asked = try post("ask", ["text": "at sufficient scale", "question": "What counts as sufficient scale?"])
        let asking = try XCTUnwrap(asked["asking"] as? [String: Any])
        XCTAssertEqual(asking["answered"] as? Bool, true)
        XCTAssertTrue(app.buttons["Close"].waitForExistence(timeout: 5))
        app.buttons["Close"].tap()
        try waitUntil("the card to close") { try self.state()["asking"] == nil }
        let placed = try marks()
        XCTAssertEqual(placed.count, 1, "the highlight stayed")
        let id = try XCTUnwrap(placed[0]["id"] as? String)

        // Tap the badge on the page as a finger would.
        let box = try rect(markID: id)
        let page = app.webViews.firstMatch
        page.coordinate(withNormalizedOffset: .zero)
            .withOffset(CGVector(dx: box.maxX - 6, dy: box.midY)).tap()
        try waitUntil("the badge to reopen the card") { try self.state()["asking"] != nil }
        XCTAssertTrue(app.buttons["Delete"].waitForExistence(timeout: 5))
        app.buttons["Delete"].tap()
        // The bin asks first, in a popover; on iPad a tap outside keeps it.
        let confirm = app.buttons["Delete the question and its highlight"]
        XCTAssertTrue(confirm.waitForExistence(timeout: 5), "the confirmation appeared")
        app.otherElements["PopoverDismissRegion"].tap()
        try waitUntil("the popover to go") { !confirm.exists }
        XCTAssertNotNil(try state()["asking"], "kept: the card is still open")
        XCTAssertEqual(try marks().count, 1)
        app.buttons["Delete"].tap()
        XCTAssertTrue(confirm.waitForExistence(timeout: 5))
        confirm.tap()
        try waitUntil("the question and its highlight to go") { try self.marks().isEmpty }
        XCTAssertNil(try state()["asking"])
    }
}
