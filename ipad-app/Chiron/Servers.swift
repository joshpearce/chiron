import Foundation

/// A saved server. The key lives in the Keychain under the id, never here -
/// this struct is written to UserDefaults, which is a plist in the app
/// container and ends up in backups.
struct SavedServer: Identifiable, Codable, Equatable {
    let id: UUID
    var name: String
    var url: String

    init(id: UUID = UUID(), name: String, url: String) {
        self.id = id
        self.name = name
        self.url = url
    }

    /// What to call a server the learner did not name.
    static func suggestedName(for url: String) -> String {
        guard let host = URLComponents(string: url)?.host, !host.isEmpty else { return "Server" }
        // A bare LAN address is more recognisable as "the Mac" than as digits.
        if host.hasPrefix("192.168.") || host.hasPrefix("10.") || host.hasPrefix("172.") {
            return "Mac (\(host))"
        }
        return host.split(separator: ".").first.map(String.init) ?? host
    }
}

/// The list of servers and which one is in use.
///
/// Addresses move - a laptop is on home Wi-Fi, then a hotspot, then its own
/// soft AP in the air, and each is a different address for the same server.
/// Retyping a URL and a key at every hop is how you end up mistyping one.
@MainActor
final class ServerStore: ObservableObject {
    @Published private(set) var servers: [SavedServer] = []
    @Published private(set) var selectedID: UUID?

    private let listKey = "savedServers"
    private let selectedKey = "selectedServerID"
    private let defaults: UserDefaults

    var selected: SavedServer? {
        servers.first { $0.id == selectedID }
    }

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        load()
    }

    private func load() {
        if let data = defaults.data(forKey: listKey),
           let list = try? JSONDecoder().decode([SavedServer].self, from: data) {
            servers = list
        }
        if let raw = defaults.string(forKey: selectedKey), let id = UUID(uuidString: raw) {
            selectedID = id
        }
        if selectedID == nil || !servers.contains(where: { $0.id == selectedID }) {
            selectedID = servers.first?.id
        }
    }

    private func save() {
        if let data = try? JSONEncoder().encode(servers) {
            defaults.set(data, forKey: listKey)
        }
        defaults.set(selectedID?.uuidString, forKey: selectedKey)
    }

    /// Bring forward whatever a previous version stored as the single server,
    /// so upgrading does not look like losing the configuration.
    func migrateIfNeeded(currentURL: String) {
        guard servers.isEmpty else { return }
        let url = currentURL.trimmingCharacters(in: .whitespaces)
        guard !url.isEmpty else { return }
        let server = SavedServer(name: SavedServer.suggestedName(for: url), url: url)
        servers = [server]
        selectedID = server.id
        if let legacy = Credentials.takeLegacyToken() {
            Credentials.setToken(legacy, for: server.id)
        }
        save()
    }

    @discardableResult
    func add(name: String, url: String, key: String?) -> SavedServer {
        let trimmed = url.trimmingCharacters(in: .whitespaces)
        let label = name.trimmingCharacters(in: .whitespaces)
        let server = SavedServer(
            name: label.isEmpty ? SavedServer.suggestedName(for: trimmed) : label,
            url: trimmed)
        servers.append(server)
        Credentials.setToken(key, for: server.id)
        selectedID = server.id
        save()
        return server
    }

    func update(_ server: SavedServer, name: String, url: String, key: String?) {
        guard let idx = servers.firstIndex(where: { $0.id == server.id }) else { return }
        let label = name.trimmingCharacters(in: .whitespaces)
        servers[idx].name = label.isEmpty
            ? SavedServer.suggestedName(for: url) : label
        servers[idx].url = url.trimmingCharacters(in: .whitespaces)
        Credentials.setToken(key, for: server.id)
        save()
    }

    func delete(_ server: SavedServer) {
        servers.removeAll { $0.id == server.id }
        // The key goes with it. Leaving an orphaned secret in the keychain for a
        // server nobody can select is the kind of thing nobody cleans up later.
        Credentials.forget(server.id)
        DeviceKey.forget(server.id)
        if selectedID == server.id { selectedID = servers.first?.id }
        save()
    }

    func select(_ server: SavedServer) {
        selectedID = server.id
        save()
    }

    /// A server set up from another device's code: added and selected, or,
    /// when its address is already here, brought up to date and selected.
    @discardableResult
    func adopt(_ link: ServerLink) -> SavedServer {
        let url = link.url.trimmingCharacters(in: .whitespaces)
        if let existing = servers.first(where: { $0.url == url }) {
            let name = link.name.trimmingCharacters(in: .whitespaces)
            update(existing, name: name.isEmpty ? existing.name : name, url: url, key: link.key)
            select(existing)
            return existing
        }
        return add(name: link.name, url: url, key: link.key)
    }
}
