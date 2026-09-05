import XCTest

/// A card dragged onto a shelf is filed there; dragged onto the library
/// row on the shelf's screen, it comes back. Read back through the
/// harness, so the assertion is about where the server says it is.
final class ShelvesUITests: XCTestCase {
    private let harness = URL(string: "http://localhost:8087")!

    private var made: String?

    override func setUp() {
        continueAfterFailure = false
    }

    /// A failed assertion skips Swift's defer, so the shelf goes here.
    override func tearDown() {
        if let made { try? post("shelf/delete", ["id": made]) }
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

    private func shelfOf(_ subject: String) throws -> String {
        let rows = try XCTUnwrap(try state()["shelf_rows"] as? [[String: Any]])
        return rows.first { $0["id"] as? String == subject }?["shelf"] as? String ?? ""
    }

    func testDraggingACardOntoAShelfFilesItAndBack() throws {
        let app = XCUIApplication()
        app.launchArguments = ["harness"]
        app.launchEnvironment["CHIRON_SERVER"] = ProcessInfo.processInfo.environment["CHIRON_SERVER"] ?? "http://localhost:8084"
        app.launch()
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, (try? state()) == nil { Thread.sleep(forTimeInterval: 0.5) }
        try post("shelf", [:])
        // Shelves a broken run left behind go first.
        for old in try XCTUnwrap(try state()["shelves"] as? [[String: Any]]) where old["name"] as? String == "Dragged" {
            try post("shelf/delete", ["id": old["id"] as? String ?? ""])
        }
        try post("shelf/create", ["name": "Dragged"])
        let shelves = try XCTUnwrap(try state()["shelves"] as? [[String: Any]])
        let shelf = try XCTUnwrap(shelves.first { $0["name"] as? String == "Dragged" })
        let id = try XCTUnwrap(shelf["id"] as? String)
        made = id

        let folder = app.buttons["Shelf: Dragged"].firstMatch
        XCTAssertTrue(folder.waitForExistence(timeout: 10), "the shelf's folder card")
        let card = app.buttons["Where the Words Come From"].firstMatch
        XCTAssertTrue(card.waitForExistence(timeout: 5), "a book card in the library")
        card.press(forDuration: 0.6, thenDragTo: folder, withVelocity: .slow, thenHoldForDuration: 0.8)
        let filed = Date().addingTimeInterval(10)
        while Date() < filed, (try shelfOf("data")) != id { Thread.sleep(forTimeInterval: 0.5) }
        XCTAssertEqual(try shelfOf("data"), id, "the book is on the shelf")

        // Onto the shelf's screen, and back off it by the library row.
        try post("shelf/open", ["id": id])
        let library = app.descendants(matching: .any)["Back to the library"].firstMatch
        XCTAssertTrue(library.waitForExistence(timeout: 10), "the library drop row")
        let onShelf = app.buttons["Where the Words Come From"].firstMatch
        XCTAssertTrue(onShelf.waitForExistence(timeout: 5))
        onShelf.press(forDuration: 0.6, thenDragTo: library, withVelocity: .slow, thenHoldForDuration: 0.8)
        let unfiled = Date().addingTimeInterval(10)
        while Date() < unfiled, (try shelfOf("data")) != "" { Thread.sleep(forTimeInterval: 0.5) }
        XCTAssertEqual(try shelfOf("data"), "", "the book is back in the library")
        try post("shelf/close", [:])
    }

    /// The shelf's own menu renames it and deletes it, through the real
    /// menu, alert and confirmation, which a dialog inside a menu never
    /// reached.
    func testTheShelfMenuRenamesAndDeletes() throws {
        let app = XCUIApplication()
        app.launchArguments = ["harness"]
        app.launchEnvironment["CHIRON_SERVER"] = ProcessInfo.processInfo.environment["CHIRON_SERVER"] ?? "http://localhost:8084"
        app.launch()
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, (try? state()) == nil { Thread.sleep(forTimeInterval: 0.5) }
        try post("shelf", [:])
        for old in try XCTUnwrap(try state()["shelves"] as? [[String: Any]]) where (old["name"] as? String)?.hasPrefix("Menu") == true {
            try post("shelf/delete", ["id": old["id"] as? String ?? ""])
        }
        try post("shelf/create", ["name": "Menu test"])
        let shelves = try XCTUnwrap(try state()["shelves"] as? [[String: Any]])
        let id = try XCTUnwrap(shelves.first { $0["name"] as? String == "Menu test" }?["id"] as? String)
        made = id
        try post("shelf/open", ["id": id])

        let menu = app.buttons["Shelf menu"]
        XCTAssertTrue(menu.waitForExistence(timeout: 10), "the shelf screen's menu")
        menu.tap()
        let rename = app.buttons["Rename"]
        XCTAssertTrue(rename.waitForExistence(timeout: 5))
        rename.tap()
        let field = app.textFields["Name"]
        XCTAssertTrue(field.waitForExistence(timeout: 5), "the rename alert's field")
        field.tap()
        field.typeText(" renamed")
        app.buttons["Save"].tap()
        let renamed = Date().addingTimeInterval(10)
        func name() throws -> String {
            let list = try XCTUnwrap(try state()["shelves"] as? [[String: Any]])
            return list.first { $0["id"] as? String == id }?["name"] as? String ?? ""
        }
        while Date() < renamed, (try name()) != "Menu test renamed" { Thread.sleep(forTimeInterval: 0.5) }
        XCTAssertEqual(try name(), "Menu test renamed")

        menu.tap()
        let delete = app.buttons["Delete shelf"]
        XCTAssertTrue(delete.waitForExistence(timeout: 5))
        delete.tap()
        let confirm = app.buttons["Delete the shelf"]
        XCTAssertTrue(confirm.waitForExistence(timeout: 5), "the confirmation")
        confirm.tap()
        let gone = Date().addingTimeInterval(10)
        while Date() < gone, (try name()) != "" { Thread.sleep(forTimeInterval: 0.5) }
        XCTAssertEqual(try name(), "", "the shelf is deleted")
        XCTAssertEqual(try state()["open_shelf"] as? String, "", "and no longer open")
        made = nil
    }
}
