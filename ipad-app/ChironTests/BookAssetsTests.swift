import XCTest
@testable import Chiron

/// An imported book's pictures: fetched once from the server, then kept,
/// so a chapter turned back to costs nothing and a book read where there
/// is no server still has its exhibits.
@MainActor
final class BookAssetsTests: XCTestCase {
    private func store(_ fake: FakeService) -> BookAssets {
        let dir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        return BookAssets(service: fake, storage: dir)
    }

    func testAPictureIsFetchedOnceAndThenKept() async throws {
        let fake = FakeService()
        let assets = store(fake)
        let first = try await assets.data(subject: "read-1", name: "c07f001.jpg")
        XCTAssertEqual(String(data: first, encoding: .utf8), "bytes of c07f001.jpg")
        XCTAssertEqual(fake.assetFetches.map(\.name), ["c07f001.jpg"])

        // Again from this device: the server is not asked twice.
        let again = try await assets.data(subject: "read-1", name: "c07f001.jpg")
        XCTAssertEqual(again, first)
        XCTAssertEqual(fake.assetFetches.count, 1, "the second read came from the device")

        // And with no server to ask, what is kept still reads.
        fake.onBookAsset = { _, _ in throw ServiceError.status(0) }
        let offline = try await assets.data(subject: "read-1", name: "c07f001.jpg")
        XCTAssertEqual(offline, first)

        // One that was never fetched cannot be conjured.
        do {
            _ = try await assets.data(subject: "read-1", name: "never-seen.jpg")
            XCTFail("a picture this device does not have needs the server")
        } catch {}
    }

    func testABooksPicturesGoWhenTheBookDoes() async throws {
        let fake = FakeService()
        let assets = store(fake)
        _ = try await assets.data(subject: "read-1", name: "fig.jpg")
        assets.forget(subject: "read-1")
        fake.assetFetches.removeAll()
        _ = try await assets.data(subject: "read-1", name: "fig.jpg")
        XCTAssertEqual(fake.assetFetches.count, 1, "the picture was fetched again")
    }

    /// The page asks for a picture under the book's own scheme.
    func testTheSchemeNamesTheBookAndThePicture() {
        XCTAssertEqual(BookAssets.base(subject: "read-338e"), "chiron-asset://book/read-338e/")
        XCTAssertEqual(BookAssetScheme.mime(for: "c01f002.jpg"), "image/jpeg")
        XCTAssertEqual(BookAssetScheme.mime(for: "logo.png"), "image/png")
    }
}

/// Taking a book: every chapter and every picture, so it reads with no
/// server to ask.
@MainActor
final class KeepOnDeviceTests: XCTestCase {
    private func session(_ fake: FakeService, storage: URL) -> BookSession {
        let s = BookSession(subjectID: "read-1", title: "A Borrowed Book", service: fake, storage: storage)
        s.kind = "reading"
        s.assets = BookAssets(service: fake, storage: storage)
        return s
    }

    private func chapterStatus(_ unit: String) -> ChapterStatus {
        let json = """
        {"chapter":{"unit":"\(unit)","title":"Chapter \(unit)","minutes":5,"html":"<p>the prose of \(unit)</p>",
         "beats":[],"pretest":[],"check":[],"calibration":false,"next_action":"read"},
         "authoring":false,"authoring_error":""}
        """
        return try! JSONDecoder().decode(ChapterStatus.self, from: Data(json.utf8))
    }

    func testTheWholeBookIsTakenAndThenReadsWithNoServer() async throws {
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let fake = FakeService()
        let spine = [
            SpineEntry(unit: "u1", title: "One", status: "active", score: nil, inFringe: true),
            SpineEntry(unit: "u2", title: "Two", status: "available", score: nil, inFringe: true),
        ]
        fake.onChapterByUnit = { [self] _, unit in chapterStatus(unit) }
        fake.onBookAssetNames = { _ in ["fig1.jpg", "fig2.jpg"] }
        let s = session(fake, storage: storage)
        s.setBookStateForTesting(BookState(spine: spine, fringe: ["u1", "u2"], debt: [],
                                           activeMisconceptions: [], summary: "", sessionMinutes: 0))
        XCTAssertEqual(s.kept, .no)

        await s.keepOnDevice()
        XCTAssertEqual(s.kept, .yes)
        XCTAssertEqual(fake.chaptersByUnit, ["u1", "u2"], "every chapter was fetched")
        XCTAssertEqual(fake.assetFetches.map(\.name), ["fig1.jpg", "fig2.jpg"], "and every picture")
        XCTAssertEqual(s.storedChapter("u2")?.title, "Chapter u2")

        // With the server gone, turning to a chapter reads what is held.
        fake.onExchange = { _ in throw ServiceError.status(0) }
        await s.turnTo(spine[1])
        XCTAssertEqual(s.chapter?.unit, "u2", "the held chapter was read")
        guard case .reading = s.screen else { return XCTFail("the reader should be open: \(s.screen)") }

        // A chapter never taken still needs the server.
        await s.turnTo(SpineEntry(unit: "u9", title: "Nine", status: "available", score: nil, inFringe: true))
        guard case .error = s.screen else { return XCTFail("a chapter not taken needs the server: \(s.screen)") }
    }
}
