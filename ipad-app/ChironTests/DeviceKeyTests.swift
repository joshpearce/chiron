import CryptoKit
import XCTest
@testable import Chiron

final class DeviceKeyTests: XCTestCase {
    /// The line must be byte-for-byte what OpenSSH writes: the base64 wraps
    /// "ssh-ed25519" and the 32 key bytes, each length-prefixed. This vector
    /// is the same key the server's tests enrol.
    func testAuthorizedKeysLineMatchesOpenSSH() throws {
        let blob = try XCTUnwrap(Data(base64Encoded: "AAAAC3NzaC1lZDI1NTE5AAAAIN3qA5o3Z8lHwlZ0m+XW2bbz4ZFf6wTqjV1p8QqYxuZk"))
        let raw = blob.suffix(32)
        let key = try Curve25519.Signing.PublicKey(rawRepresentation: raw)
        XCTAssertEqual(DeviceKey.authorizedKeysLine(key),
                       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN3qA5o3Z8lHwlZ0m+XW2bbz4ZFf6wTqjV1p8QqYxuZk")
    }

    func testLineRoundTripsAFreshKey() throws {
        let key = Curve25519.Signing.PrivateKey()
        let line = DeviceKey.authorizedKeysLine(key.publicKey)
        let parts = line.split(separator: " ")
        XCTAssertEqual(parts.count, 2)
        XCTAssertEqual(parts[0], "ssh-ed25519")
        let blob = try XCTUnwrap(Data(base64Encoded: String(parts[1])))
        XCTAssertEqual(blob.count, 4 + 11 + 4 + 32)
        XCTAssertEqual(blob.suffix(32), key.publicKey.rawRepresentation)
    }
}
