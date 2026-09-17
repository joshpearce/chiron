import XCTest
@testable import Chiron

/// The secrets live in the data-protection keychain on every device, the
/// Mac included, where the file keychain would refuse the `ThisDeviceOnly`
/// accessibility and prompt for a password besides.
final class KeychainTests: XCTestCase {
    private let service = "com.mjbraun.chiron.test"

    func testASecretRoundTripsAndForgetsCleanly() {
        let account = UUID().uuidString
        addTeardownBlock { Keychain.write(nil, service: self.service, account: account) }
        XCTAssertNil(Keychain.read(service: service, account: account))
        Keychain.write(Data("first".utf8), service: service, account: account)
        XCTAssertEqual(Keychain.read(service: service, account: account), Data("first".utf8))
        Keychain.write(Data("second".utf8), service: service, account: account)
        XCTAssertEqual(Keychain.read(service: service, account: account), Data("second".utf8), "a write replaces")
        Keychain.write(nil, service: service, account: account)
        XCTAssertNil(Keychain.read(service: service, account: account))
    }

    func testAccountsDoNotBleed() {
        let a = UUID().uuidString, b = UUID().uuidString
        addTeardownBlock {
            Keychain.write(nil, service: self.service, account: a)
            Keychain.write(nil, service: self.service, account: b)
        }
        Keychain.write(Data("a".utf8), service: service, account: a)
        XCTAssertNil(Keychain.read(service: service, account: b))
    }
}
