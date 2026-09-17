import XCTest
@testable import Chiron

/// A server's setup travels to another device as a chiron://server URL,
/// in a QR code: the name, the address and the shared key, and nothing
/// the other device has to type.
@MainActor
final class ServerLinkTests: XCTestCase {
    func testALinkRoundTripsThroughItsURL() throws {
        let link = ServerLink(name: "sprite", url: "https://chiron.example", key: "a key/with spaces+signs")
        let back = try XCTUnwrap(ServerLink(link.asURL))
        XCTAssertEqual(back, link)
        XCTAssertEqual(link.asURL.scheme, "chiron")
        XCTAssertEqual(link.asURL.host, "server")
    }

    func testALinkWithoutAKeyOrNameStillCarriesTheAddress() throws {
        let link = ServerLink(name: "", url: "http://192.168.2.1:8080", key: nil)
        let back = try XCTUnwrap(ServerLink(link.asURL))
        XCTAssertEqual(back.url, "http://192.168.2.1:8080")
        XCTAssertNil(back.key)
        XCTAssertEqual(back.name, "")
    }

    func testOtherURLsAreNotLinks() {
        XCTAssertNil(ServerLink(URL(string: "chiron://capture/abc")!))
        XCTAssertNil(ServerLink(URL(string: "chiron://server?name=x")!), "no address")
        XCTAssertNil(ServerLink(URL(string: "https://example.com/server?url=x")!), "wrong scheme")
    }

    /// A Mac has no camera to scan with; the link comes over as text,
    /// pasted from the other device's clipboard.
    func testAPastedLinkIsReadWithItsWhitespaceForgiven() throws {
        let link = ServerLink(name: "sprite", url: "https://chiron.example", key: "k1")
        XCTAssertEqual(ServerLink(pasted: "  \(link.asURL.absoluteString)\n"), link)
        XCTAssertNil(ServerLink(pasted: "https://chiron.example"), "an address alone is typed in, not pasted as a link")
        XCTAssertNil(ServerLink(pasted: ""))
    }

    private func store() -> ServerStore {
        let suite = "test-\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: suite)!
        addTeardownBlock { defaults.removePersistentDomain(forName: suite) }
        return ServerStore(defaults: defaults)
    }

    func testAdoptingALinkAddsAndSelectsTheServer() {
        let store = store()
        let s = store.adopt(ServerLink(name: "sprite", url: "https://chiron.example", key: "k1"))
        XCTAssertEqual(store.servers.map(\.url), ["https://chiron.example"])
        XCTAssertEqual(store.selectedID, s.id)
        XCTAssertEqual(s.name, "sprite")
        XCTAssertEqual(Credentials.token(for: s.id), "k1")
        store.delete(s)
    }

    func testAdoptingTheSameAddressAgainUpdatesRatherThanDuplicates() {
        let store = store()
        let first = store.adopt(ServerLink(name: "sprite", url: "https://chiron.example", key: "k1"))
        let again = store.adopt(ServerLink(name: "", url: "https://chiron.example", key: "k2"))
        XCTAssertEqual(again.id, first.id)
        XCTAssertEqual(store.servers.count, 1)
        XCTAssertEqual(store.servers.first?.name, "sprite", "an unnamed link keeps the name it had")
        XCTAssertEqual(Credentials.token(for: first.id), "k2")
        store.delete(first)
    }

    func testTheCodeRenders() {
        XCTAssertNotNil(QRCode.image(of: "chiron://server?url=http://x"))
    }
}
