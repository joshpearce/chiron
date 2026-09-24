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

    /// The reader picks the order once, and it holds in the library and
    /// on every shelf: newest first by what last changed or was opened,
    /// or by title. What has no stamp (an older server) goes by title.
    func testCardsAreOrderedByRecencyOrByNameEverywhere() async {
        let fake = FakeService()
        let filed = ["zebra": "s1", "apple": "s1"]
        let stamps = ["mango": "2026-09-20T10:00:00Z", "zebra": "2026-09-22T10:00:00Z", "apple": "2026-09-21T10:00:00Z"]
        fake.onSubjects = {
            let rows = ["mango", "zebra", "apple", "kiwi"].map { id in
                SubjectInfo(id: id, title: id.capitalized, kind: "book", shelf: filed[id], updatedAt: stamps[id])
            }
            return SubjectsResponse(subjects: rows, active: nil, shelves: [ShelfInfo(id: "s1", name: "Fruit", subjects: ["zebra", "apple"])])
        }
        let defaults = UserDefaults(suiteName: "shelves-order-\(UUID().uuidString)")!
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let library = Library(storage: storage, service: fake, defaults: defaults)
        await library.refresh()

        XCTAssertEqual(library.order, .recent, "newest first unless the reader says otherwise")
        XCTAssertEqual(library.unfiled.map(\.id), ["mango", "kiwi"], "an unstamped card comes after the stamped, by title")
        XCTAssertEqual(library.subjects(on: "s1").map(\.id), ["zebra", "apple"])

        library.order = .alphabetical
        XCTAssertEqual(library.unfiled.map(\.id), ["kiwi", "mango"])
        XCTAssertEqual(library.subjects(on: "s1").map(\.id), ["apple", "zebra"])

        let later = Library(storage: storage, service: fake, defaults: defaults)
        XCTAssertEqual(later.order, .alphabetical, "the choice is kept")
    }

    func testAServerWithoutShelvesListsAPlainLibrary() throws {
        let json = #"{"subjects":[{"id":"ai","title":"How AI Works","kind":"book"}],"active":"ai"}"#
        let r = try JSONDecoder().decode(SubjectsResponse.self, from: Data(json.utf8))
        XCTAssertNil(r.shelves)
        XCTAssertNil(r.subjects.first?.shelf)
        XCTAssertNil(r.subjects.first?.updatedAt)
    }
}
