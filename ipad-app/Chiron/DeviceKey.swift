import CryptoKit
import Foundation
import Security

/// This iPad's ssh identity for a server: an Ed25519 key made on the device,
/// kept in the Keychain, and enrolled on the server with the shared key. One
/// key per saved server, so revoking one on the sprite revokes only this
/// device's access to that server.
enum DeviceKey {
    private static let service = "com.mjbraun.chiron.devicekey"

    /// The key for a server, created on first use.
    static func privateKey(for serverID: UUID) -> Curve25519.Signing.PrivateKey {
        if let raw = read(account: serverID.uuidString),
           let key = try? Curve25519.Signing.PrivateKey(rawRepresentation: raw) {
            return key
        }
        let key = Curve25519.Signing.PrivateKey()
        write(key.rawRepresentation, account: serverID.uuidString)
        return key
    }

    static func forget(_ serverID: UUID) {
        write(nil, account: serverID.uuidString)
    }

    /// The public half as an authorized_keys line: "ssh-ed25519 <base64>".
    /// The base64 wraps two length-prefixed strings, the algorithm name and
    /// the 32 raw key bytes (RFC 4253 section 6.6).
    static func authorizedKeysLine(_ publicKey: Curve25519.Signing.PublicKey) -> String {
        var blob = Data()
        for part in [Data("ssh-ed25519".utf8), publicKey.rawRepresentation] {
            var n = UInt32(part.count).bigEndian
            blob.append(Data(bytes: &n, count: 4))
            blob.append(part)
        }
        return "ssh-ed25519 " + blob.base64EncodedString()
    }

    static func authorizedKeysLine(for serverID: UUID) -> String {
        authorizedKeysLine(privateKey(for: serverID).publicKey)
    }

    // MARK: - keychain

    private static func read(account: String) -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var out: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &out) == errSecSuccess else { return nil }
        return out as? Data
    }

    private static func write(_ value: Data?, account: String) {
        let base: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(base as CFDictionary)
        guard let value else { return }
        var add = base
        add[kSecValueData as String] = value
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        SecItemAdd(add as CFDictionary, nil)
    }
}
