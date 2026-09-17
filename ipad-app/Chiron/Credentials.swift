import Foundation

/// Shared secrets for saved servers, one per server.
///
/// Keychain rather than UserDefaults: UserDefaults is a plist in the app
/// container, readable from a backup, and this token is the only thing standing
/// between a public endpoint and someone else's model budget. `ThisDeviceOnly`
/// keeps it out of encrypted backups and off any other device.
///
/// Keyed by the server's id rather than stored as one global token, so moving
/// between a LAN server with no key and a public one with a key does not mean
/// retyping - or worse, silently sending one server's key to another.
enum Credentials {
    private static let service = "com.mjbraun.chiron"
    private static let legacyAccount = "server-token"

    static func token(for serverID: UUID) -> String? {
        read(account: serverID.uuidString)
    }

    static func setToken(_ value: String?, for serverID: UUID) {
        write(value, account: serverID.uuidString)
    }

    /// The token stored before servers were a list. Read once during migration
    /// and then cleared, so it does not linger in the keychain unused.
    static func takeLegacyToken() -> String? {
        let existing = read(account: legacyAccount)
        if existing != nil { write(nil, account: legacyAccount) }
        return existing
    }

    static func forget(_ serverID: UUID) {
        write(nil, account: serverID.uuidString)
    }

    /// Attach the secret if there is one. A blank token means the LAN case,
    /// where the server is open and the header would be meaningless.
    static func authorize(_ request: inout URLRequest, serverID: UUID?) {
        guard let id = serverID, let t = token(for: id) else { return }
        request.setValue("Bearer \(t)", forHTTPHeaderField: "Authorization")
    }

    private static func read(account: String) -> String? {
        guard let data = Keychain.read(service: service, account: account),
              let s = String(data: data, encoding: .utf8), !s.isEmpty else { return nil }
        return s
    }

    private static func write(_ value: String?, account: String) {
        let v = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        Keychain.write(v.isEmpty ? nil : Data(v.utf8), service: service, account: account)
    }
}
