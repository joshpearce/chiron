import Foundation

/// The server as the book sees it. `Sync` talks HTTP; tests hand a
/// `BookSession` a fake.
protocol ChironService: AnyObject {
    func subjects() async throws -> SubjectsResponse
    func state(subject: String) async throws -> BookState
    func chapter(subject: String) async throws -> ChapterStatus
    func exchange(_ request: ExchangeRequest) async throws -> ExchangeResponse
    func ink(subject: String, _ submission: InkSubmission) async throws -> ExchangeResponse
    func ask(subject: String, unit: String, quote: String, question: String) async throws -> AskResponse
    func capture(_ request: CaptureRequest) async throws -> CaptureResponse
    func extend(subject: String, quote: String, note: String) async throws -> ExtendResponse
    func reset(subject: String) async throws -> BookState
}

enum ServiceError: Error {
    case badURL
    case status(Int)
}

/// HTTP client for the book server, with bearer auth from the saved server's
/// key and a launch-environment override for development.
@MainActor
final class Sync: ObservableObject, ChironService {
    @Published var baseURL: String {
        didSet { UserDefaults.standard.set(baseURL, forKey: "baseURL") }
    }
    /// Saved servers. The selected one drives baseURL and supplies its own key.
    let servers = ServerStore()
    var serverID: UUID? { servers.selected?.id }
    @Published var connected = false
    /// Whether the server has a model behind it (/health llm.connected).
    /// Without one, mechanical grading and pre-authored chapters still work;
    /// free-text grading and adaptive authoring do not.
    @Published var llmConnected = false

    init() {
        baseURL = UserDefaults.standard.string(forKey: "baseURL") ?? "http://192.168.2.1:8080"
        // Anything configured before servers were a list becomes the first
        // saved entry, so upgrading does not read as losing the setup.
        servers.migrateIfNeeded(currentURL: baseURL)
        if let selected = servers.selected { baseURL = selected.url }
        // Development override: point the app at a server for this launch only,
        // without touching saved servers. Simulator runs use
        //   SIMCTL_CHILD_CHIRON_SERVER=http://localhost:8084 xcrun simctl launch ...
        if let override = ProcessInfo.processInfo.environment["CHIRON_SERVER"],
           !override.isEmpty {
            baseURL = override
        }
        Task { await probe() }
    }

    func probe() async {
        if let health: Health = try? await get("/health", timeout: 3) {
            connected = true
            llmConnected = health.llm.connected
            return
        }
        connected = false
    }

    // MARK: - ChironService

    func subjects() async throws -> SubjectsResponse {
        try await get("/subjects", timeout: 5)
    }

    func state(subject: String) async throws -> BookState {
        try await get("/state?subject=\(subject)", timeout: 10)
    }

    func chapter(subject: String) async throws -> ChapterStatus {
        try await get("/chapter/\(subject)", timeout: 30)
    }

    /// Grading a check is a model call; the timeout is generous but finite
    /// so a server that hangs surfaces as an error rather than a spinner.
    func exchange(_ request: ExchangeRequest) async throws -> ExchangeResponse {
        try await post("/exchange", body: try JSONEncoder().encode(request), timeout: 600)
    }

    /// The ink check-in transcribes before it grades; same budget as an
    /// exchange.
    func ink(subject: String, _ submission: InkSubmission) async throws -> ExchangeResponse {
        try await post("/ink/\(subject)", body: try JSONEncoder().encode(submission), timeout: 600)
    }

    /// One model call; the answer comes back in the same request.
    func ask(subject: String, unit: String, quote: String, question: String) async throws -> AskResponse {
        struct Body: Encodable { let unit, quote, question: String }
        return try await post("/ask/\(subject)",
                              body: try JSONEncoder().encode(Body(unit: unit, quote: quote, question: question)),
                              timeout: 120)
    }

    // MARK: - device keys

    /// Put this device's ssh key on the server. Possessing the shared key
    /// is what authorizes it; a server without key enrolment says so.
    func enrolDeviceKey(name: String) async throws -> EnrolResponse {
        guard let id = serverID else { throw ServiceError.badURL }
        struct Body: Encodable { let pubkey, name: String }
        return try await post("/agent/pubkey",
                              body: try JSONEncoder().encode(Body(pubkey: DeviceKey.authorizedKeysLine(for: id), name: name)),
                              timeout: 30)
    }

    func serverKeys() async throws -> [EnrolledKey] {
        struct Reply: Decodable { let keys: [EnrolledKey] }
        let r: Reply = try await get("/agent/keys", timeout: 15)
        return r.keys
    }

    func revokeKey(fingerprint: String) async throws {
        struct Body: Encodable { let fingerprint: String }
        struct Reply: Decodable { let removed: Int }
        let _: Reply = try await post("/agent/keys/revoke", body: try JSONEncoder().encode(Body(fingerprint: fingerprint)), timeout: 15)
    }

    /// A capture becomes a primer on the shelf; authoring runs on the
    /// server after this returns.
    func capture(_ request: CaptureRequest) async throws -> CaptureResponse {
        try await post("/primer/capture", body: try JSONEncoder().encode(request), timeout: 60)
    }

    /// A margin note extends the primer; the whole document comes back.
    func extend(subject: String, quote: String, note: String) async throws -> ExtendResponse {
        struct Body: Encodable { let quote, note: String }
        return try await post("/primer/\(subject)/extend", body: try JSONEncoder().encode(Body(quote: quote, note: note)), timeout: 180)
    }

    func reset(subject: String) async throws -> BookState {
        // Hand-encoded: confirm is a bool on the wire, and a [String: String]
        // dictionary would send it as the string "true", which the server
        // rejects - deliberately, since this discards everything.
        try await post("/reset", body: Data("{\"subject\":\"\(subject)\",\"confirm\":true}".utf8), timeout: 30)
    }

    // MARK: - transport

    private func request(_ path: String, timeout: TimeInterval) throws -> URLRequest {
        guard let url = URL(string: "\(baseURL)\(path)") else { throw ServiceError.badURL }
        var req = URLRequest(url: url, timeoutInterval: timeout)
        Credentials.authorize(&req, serverID: serverID)
        return req
    }

    private func perform<T: Decodable>(_ req: URLRequest) async throws -> T {
        let (data, resp) = try await URLSession.shared.data(for: req)
        let code = (resp as? HTTPURLResponse)?.statusCode ?? 0
        guard code == 200 else {
            connected = false
            throw ServiceError.status(code)
        }
        connected = true
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func get<T: Decodable>(_ path: String, timeout: TimeInterval) async throws -> T {
        try await perform(try request(path, timeout: timeout))
    }

    private func post<T: Decodable>(_ path: String, body: Data, timeout: TimeInterval) async throws -> T {
        var req = try request(path, timeout: timeout)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = body
        return try await perform(req)
    }
}

private struct Health: Decodable {
    struct LLM: Decodable { let connected: Bool }
    let llm: LLM
}
