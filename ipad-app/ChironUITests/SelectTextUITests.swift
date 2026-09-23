import XCTest

/// Taking a passage out of the planning conversation. Selection is a
/// gesture, so nothing below the UI can prove it works: a unit test sees a
/// text view that says it is selectable, and a SwiftUI gesture laid over the
/// top can still swallow the long press that starts a selection. Only a
/// finger finds that out.
final class SelectTextUITests: HarnessTestCase {
    func testAPassageOfTheTutorsWordsCanBeSelected() throws {
        let app = self.app
        try waitUntil("the server to answer") { (try? self.state()["connected"] as? Bool) == true }

        try post("capture", ["prompt": "How do macaroons and actor-chain JWTs differ?",
                             "scale": "primer",
                             "text": "Agent identity is the hard part: a chain of actors, each attenuating what the next may do."])
        try waitUntil("the planning card") { (try? self.state()["plan_card"]) as? [String: Any] != nil }

        // The tutor's turn is a text view; a double tap takes a word of it.
        let turn = app.textViews.element(boundBy: 0)
        XCTAssertTrue(turn.waitForExistence(timeout: 10), "the conversation is on screen")
        turn.doubleTap()

        // A real selection brings the system's own edit menu. Select All is
        // the tell: it comes from the text view, not from anything we drew.
        let copy = app.menuItems["Copy"]
        XCTAssertTrue(copy.waitForExistence(timeout: 5), "a double tap selects a word and offers the edit menu")
        // Look Up and Share come with a text selection and with nothing
        // else: a menu of our own drawing would offer Copy alone.
        let offered = app.menuItems.allElementsBoundByIndex.map(\.label)
        XCTAssertTrue(offered.contains("Look Up"),
                      "the text view owns the selection, so its own menu appears; got \(offered)")
    }
}
