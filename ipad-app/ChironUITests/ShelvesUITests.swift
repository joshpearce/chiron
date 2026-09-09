import XCTest

/// A card dragged onto a shelf is filed there; dragged onto the library
/// row on the shelf's screen, it comes back. Read back through the
/// harness, so the assertion is about where the server says it is.
final class ShelvesUITests: HarnessTestCase {
    private var made: String?

    /// A failed assertion skips Swift's defer, so the shelf goes here.
    override func tearDown() {
        if let made { try? post("shelf/delete", ["id": made]) }
        super.tearDown()
    }

    private func shelfOf(_ subject: String) throws -> String {
        let rows = try XCTUnwrap(try state()["shelf_rows"] as? [[String: Any]])
        return rows.first { $0["id"] as? String == subject }?["shelf"] as? String ?? ""
    }

    private func shelves() throws -> [[String: Any]] {
        try XCTUnwrap(try state()["shelves"] as? [[String: Any]])
    }

    /// The shelf screen, with any shelf an earlier run left behind removed.
    private func freshShelf(named name: String, clearing prefix: String) throws -> String {
        _ = app
        try post("shelf")
        for old in try shelves() where (old["name"] as? String)?.hasPrefix(prefix) == true {
            try post("shelf/delete", ["id": old["id"] as? String ?? ""])
        }
        try post("shelf/create", ["name": name])
        let id = try XCTUnwrap(try shelves().first { $0["name"] as? String == name }?["id"] as? String)
        made = id
        return id
    }

    func testDraggingACardOntoAShelfFilesItAndBack() throws {
        let id = try freshShelf(named: "Dragged", clearing: "Dragged")
        let folder = app.buttons["Shelf: Dragged"].firstMatch
        XCTAssertTrue(folder.waitForExistence(timeout: 10), "the shelf's folder card")
        let card = app.buttons["Where the Words Come From"].firstMatch
        XCTAssertTrue(card.waitForExistence(timeout: 5), "a book card in the library")
        card.press(forDuration: 0.6, thenDragTo: folder, withVelocity: .slow, thenHoldForDuration: 0.8)
        try waitUntil("the book to land on the shelf") { (try self.shelfOf("data")) == id }

        // Onto the shelf's screen, and back off it by the library row.
        try post("shelf/open", ["id": id])
        let library = app.descendants(matching: .any)["Back to the library"].firstMatch
        XCTAssertTrue(library.waitForExistence(timeout: 10), "the library drop row")
        let onShelf = app.buttons["Where the Words Come From"].firstMatch
        XCTAssertTrue(onShelf.waitForExistence(timeout: 5))
        onShelf.press(forDuration: 0.6, thenDragTo: library, withVelocity: .slow, thenHoldForDuration: 0.8)
        try waitUntil("the book back in the library") { (try self.shelfOf("data")) == "" }
        try post("shelf/close")
    }

    /// The shelf's own menu renames it and deletes it, through the real
    /// menu, alert and confirmation, which a dialog inside a menu never
    /// reached.
    func testTheShelfMenuRenamesAndDeletes() throws {
        let id = try freshShelf(named: "Menu test", clearing: "Menu")
        try post("shelf/open", ["id": id])
        func name() throws -> String {
            try shelves().first { $0["id"] as? String == id }?["name"] as? String ?? ""
        }

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
        try waitUntil("the rename to reach the server") { try name() == "Menu test renamed" }

        menu.tap()
        let delete = app.buttons["Delete shelf"]
        XCTAssertTrue(delete.waitForExistence(timeout: 5))
        delete.tap()
        let confirm = app.buttons["Delete the shelf"]
        XCTAssertTrue(confirm.waitForExistence(timeout: 5), "the confirmation")
        confirm.tap()
        try waitUntil("the shelf to be deleted") { try name() == "" }
        XCTAssertEqual(try state()["open_shelf"] as? String, "", "and no longer open")
        made = nil
    }
}
