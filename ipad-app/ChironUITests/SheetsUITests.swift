import XCTest

/// Every sheet the app puts up, opened. A sheet is its own presentation -
/// on the Mac, its own window - so what the screen behind it holds does
/// not reach it: a view that asks the environment for an object nobody
/// handed it stops the app on the spot. Opening each one is the only way
/// to find that out, and the Mac is the only place it happens.
final class SheetsUITests: HarnessTestCase {
    func testEverySheetOpensWithWhatItsViewAsksFor() throws {
        let app = self.app
        for sheet in ["settings", "requests/card", "server/setup"] {
            try post(sheet, ["open": true, "shown": true])
            XCTAssertEqual(app.state, .runningForeground, "the app went down opening \(sheet)")
            try post(sheet, ["open": false, "shown": false])
        }

        // The reader's own: the contents, and the capture card a passage
        // sent on puts up.
        try post("open", ["subject": "ai"])
        try waitUntil("the book to be open") { (try? self.state()["subject"] as? String) == "ai" }
        try post("contents")
        XCTAssertEqual(app.state, .runningForeground, "the app went down opening the contents")
        try post("contents")
        try post("capture/card", ["text": "A passage to ask about"])
        XCTAssertEqual(app.state, .runningForeground, "the app went down opening the capture card")
        try post("capture/close")
        XCTAssertEqual(app.state, .runningForeground)
    }
}
