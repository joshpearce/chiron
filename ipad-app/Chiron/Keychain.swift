import Foundation
import Security

/// Generic-password items in the data-protection keychain, `ThisDeviceOnly`:
/// out of backups and off every other device. The data-protection keychain
/// is the only one on iOS; on the Mac it is named, or the item lands in the
/// file keychain, which refuses the accessibility class and asks for a
/// password when the app reads it back.
enum Keychain {
    /// This is also the app target's `keychain-access-groups` entitlement.
    /// Naming it explicitly keeps Catalyst on the provisioned data-protection
    /// access group instead of relying on the macOS file-keychain default.
    static var accessGroup: String? {
        Bundle.main.object(forInfoDictionaryKey: "ChironKeychainAccessGroup") as? String
    }

    static func read(service: String, account: String) -> Data? {
        var query = base(service: service, account: account)
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var out: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &out) == errSecSuccess else { return nil }
        return out as? Data
    }

    /// Replaces the item; nil removes it.
    static func write(_ value: Data?, service: String, account: String) {
        let base = base(service: service, account: account)
        SecItemDelete(base as CFDictionary)
        guard let value else { return }
        var add = base
        add[kSecValueData as String] = value
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        SecItemAdd(add as CFDictionary, nil)
    }

    private static func base(service: String, account: String) -> [String: Any] {
        var query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecUseDataProtectionKeychain as String: true,
        ]
        if let accessGroup, !accessGroup.isEmpty {
            query[kSecAttrAccessGroup as String] = accessGroup
        }
        return query
    }
}
