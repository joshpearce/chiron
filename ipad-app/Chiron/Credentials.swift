import Foundation
import Security

/// The shared secret the sprite requires on every request.
///
/// Keychain rather than UserDefaults: UserDefaults is a plist in the app
/// container, readable from a backup, and this token is the only thing standing
/// between a public endpoint and someone else's model budget. `ThisDeviceOnly`
/// keeps it out of encrypted backups and off any other device.
enum Credentials {
    private static let service = "com.mjbraun.chiron"
    private static let account = "server-token"

    static var token: String? {
        get {
            let query: [String: Any] = [
                kSecClass as String: kSecClassGenericPassword,
                kSecAttrService as String: service,
                kSecAttrAccount as String: account,
                kSecReturnData as String: true,
                kSecMatchLimit as String: kSecMatchLimitOne,
            ]
            var out: CFTypeRef?
            guard SecItemCopyMatching(query as CFDictionary, &out) == errSecSuccess,
                  let data = out as? Data, let s = String(data: data, encoding: .utf8),
                  !s.isEmpty else { return nil }
            return s
        }
        set {
            let base: [String: Any] = [
                kSecClass as String: kSecClassGenericPassword,
                kSecAttrService as String: service,
                kSecAttrAccount as String: account,
            ]
            SecItemDelete(base as CFDictionary)
            guard let value = newValue?.trimmingCharacters(in: .whitespacesAndNewlines),
                  !value.isEmpty, let data = value.data(using: .utf8) else { return }
            var add = base
            add[kSecValueData as String] = data
            add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
            SecItemAdd(add as CFDictionary, nil)
        }
    }

    /// Attach the secret if there is one. A blank token means the LAN case,
    /// where the server is open and the header would be meaningless.
    static func authorize(_ request: inout URLRequest) {
        if let t = token {
            request.setValue("Bearer \(t)", forHTTPHeaderField: "Authorization")
        }
    }
}
