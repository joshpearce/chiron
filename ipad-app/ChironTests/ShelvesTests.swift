import XCTest
@testable import Chiron

/// The library is organised into shelves the server keeps: the app
/// groups what it lists, and every change goes to the server and comes
/// back through a refresh.
@MainActor
final class ShelvesTests: XCTestCase {
    private func response(shelves: [ShelfInfo], filed: [String: String]) -> SubjectsResponse {
        let rows = ["ai", "data", "primer-why"].map { id in
            SubjectInfo(id: id, title: id, kind: id.hasPrefix("primer") ? "primer" : "book", status: id.hasPrefix("primer") ? "ready" : nil, shelf: filed[id])
        }
        return SubjectsResponse(subjects: rows, active: "ai", shelves: shelves)
    }

    private func library(_ fake: FakeService) -> Library {
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        return Library(storage: storage, service: fake)
    }

    func testTheLibraryIsWhatIsOnNoShelf() async {
        let fake = FakeService()
        let systems = ShelfInfo(id: "s1", name: "Systems", subjects: ["ai"])
        fake.onSubjects = { [unowned self] in self.response(shelves: [systems], filed: ["ai": "s1"]) }
        let library = library(fake)
        await library.refresh()
        XCTAssertEqual(library.shelves, [systems])
        XCTAssertEqual(library.unfiled.map(\.id), ["data", "primer-why"])
        XCTAssertEqual(library.subjects(on: "s1").map(\.id), ["ai"])
    }

    func testMakingRenamingAndDeletingAShelfGoToTheServer() async {
        let fake = FakeService()
        var shelves: [ShelfInfo] = []
        fake.onSubjects = { [unowned self] in self.response(shelves: shelves, filed: [:]) }
        let library = library(fake)
        shelves = [ShelfInfo(id: "s1", name: "Systems")]
        await library.createShelf(named: "  Systems ")
        XCTAssertEqual(fake.shelvesMade, ["Systems"])
        XCTAssertEqual(library.shelves.map(\.name), ["Systems"], "the refresh after shows it")

        await library.createShelf(named: "   ")
        XCTAssertEqual(fake.shelvesMade.count, 1, "a blank name is not sent")

        shelves = [ShelfInfo(id: "s1", name: "Models")]
        await library.renameShelf("s1", to: "Models")
        XCTAssertEqual(fake.renames.map(\.name), ["Models"])
        XCTAssertEqual(library.shelves.first?.name, "Models")

        library.shelfPath = ["s1"]
        shelves = []
        await library.deleteShelf("s1")
        XCTAssertEqual(fake.shelvesDeleted, ["s1"])
        XCTAssertTrue(library.shelves.isEmpty)
        XCTAssertTrue(library.shelfPath.isEmpty, "a deleted shelf is no longer open")
    }

    func testMovingFilesTheSubjectAndBack() async {
        let fake = FakeService()
        var filed: [String: String] = [:]
        fake.onSubjects = { [unowned self] in self.response(shelves: [ShelfInfo(id: "s1", name: "Systems")], filed: filed) }
        let library = library(fake)
        filed = ["ai": "s1"]
        await library.move("ai", to: "s1")
        XCTAssertEqual(fake.moves.last?.shelf, "s1")
        XCTAssertEqual(library.subjects(on: "s1").map(\.id), ["ai"])
        filed = [:]
        await library.move("ai", to: nil)
        XCTAssertNil(fake.moves.last?.shelf)
        XCTAssertEqual(library.unfiled.count, 3)
    }

    func testAServerWithoutShelvesListsAPlainLibrary() throws {
        let json = #"{"subjects":[{"id":"ai","title":"How AI Works","kind":"book"}],"active":"ai"}"#
        let r = try JSONDecoder().decode(SubjectsResponse.self, from: Data(json.utf8))
        XCTAssertNil(r.shelves)
        XCTAssertNil(r.subjects.first?.shelf)
    }
}
