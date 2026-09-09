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
