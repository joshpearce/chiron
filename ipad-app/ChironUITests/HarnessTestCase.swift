import XCTest

/// What every UI test needs: the app, the harness's verbs against it, and
/// waits that end the moment the app has caught up.
///
/// Three things here are about the clock. The app is launched once for a
/// class and handed to each of its tests, since a launch is seconds and a
/// test puts the app where it wants it through the harness anyway. Waits
/// poll the app instead of sleeping for a guessed interval. And the
/// harness port and dev server are picked from the simulator's clone
/// number, so classes can run in parallel: clones share the Mac's
/// loopback, and two apps on one port would answer for each other.
@MainActor
class HarnessTestCase: XCTestCase {
    /// Under parallel testing each worker is "Clone N of <device>"; a
    /// simulator under its own name is the only worker.
    static let worker: Int = {
        let name = ProcessInfo.processInfo.environment["SIMULATOR_DEVICE_NAME"] ?? ""
        guard name.hasPrefix("Clone "),
              let n = Int(name.dropFirst("Clone ".count).prefix(while: \.isNumber)), n > 0 else { return 0 }
        return n - 1
    }()

    /// The dev server for this worker: the one the runner was given, moved
    /// up by the clone number, since each worker needs its own learner
    /// state to mark up and answer.
    static var serverURL: String {
        let given = ProcessInfo.processInfo.environment["CHIRON_SERVER"] ?? "http://localhost:8084"
        guard worker > 0, let url = URL(string: given), let port = url.port else { return given }
        var parts = URLComponents(url: url, resolvingAgainstBaseURL: false)
        parts?.port = port + worker
        return parts?.url?.absoluteString ?? given
    }

    var harness: URL { URL(string: "http://localhost:\(8087 + Self.worker)")! }

    /// Anything a class needs beyond `harness` and the server, such as the
    /// shell's user.
    var launchEnvironment: [String: String] { [:] }

    private nonisolated(unsafe) static var apps: [String: XCUIApplication] = [:]
    private nonisolated(unsafe) static var nonces: [String: String] = [:]

    /// The class's app, launched the first time a test asks for it and
    /// reused by the rest, unless it is no longer running.
    var app: XCUIApplication {
        let key = String(describing: type(of: self))
        if let running = Self.apps[key], running.state == .runningForeground { return running }
        let app = XCUIApplication()
        // Simulators share the Mac's loopback, so an app left running by an
        // earlier run can hold this port and answer for the one just
        // launched. Every launch carries a nonce and the state is only
        // this app's once it comes back.
        let nonce = UUID().uuidString
        Self.nonces[key] = nonce
        app.launchArguments = ["harness", "harness_port=\(8087 + Self.worker)", "harness_nonce=\(nonce)"]
        app.launchEnvironment["CHIRON_SERVER"] = Self.serverURL
        for (k, v) in launchEnvironment { app.launchEnvironment[k] = v }
        app.launch()
        Self.apps[key] = app
        let deadline = Date().addingTimeInterval(30)
        while Date() < deadline, (try? state())?["nonce"] as? String != nonce { Thread.sleep(forTimeInterval: 0.2) }
        XCTAssertEqual((try? state())?["nonce"] as? String, nonce,
                       "port \(8087 + Self.worker) answers for the app this test launched, not one left running")
        // A container with nothing in it, which is what a parallel clone
        // and a machine's first run both have, reaches no server: the
        // launch override names one but saves none, and the shelf and the
        // exchange both go to the saved server. The settings screen would
        // do this; here the harness does.
        if (try? state())?["server_url"] as? String != Self.serverURL {
            try? post("server", ["url": Self.serverURL, "name": "dev"])
        }
        return app
    }

    override func setUp() {
        super.setUp()
        continueAfterFailure = false
    }

    // MARK: the harness

    func state() throws -> [String: Any] {
        let data = try Data(contentsOf: harness.appendingPathComponent("state"))
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    func get(_ path: String) throws -> [String: Any] {
        let data = try Data(contentsOf: harness.appendingPathComponent(path))
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    /// A verb, with the state it settled into. The harness runs the action
    /// to completion before it answers, so a post needs no wait after it.
    @discardableResult
    func post(_ path: String, _ body: [String: Any] = [:], timeout: TimeInterval = 60) throws -> [String: Any] {
        var req = URLRequest(url: harness.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        var out: [String: Any] = [:]
        let done = expectation(description: path)
        URLSession.shared.dataTask(with: req) { data, _, _ in
            out = data.flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] } ?? [:]
            done.fulfill()
        }.resume()
        wait(for: [done], timeout: timeout)
        return out
    }

    /// A line of JavaScript against the open page, and what it came to.
    @discardableResult
    func eval(_ js: String) throws -> String {
        try post("reader/eval", ["js": js])["result"] as? String ?? ""
    }

    /// Where the first element matching a selector is, in page-view points.
    func rect(selector: String, file: StaticString = #filePath, line: UInt = #line) throws -> CGRect {
        let out = try post("reader/rect", ["selector": selector])
        XCTAssertEqual(out["found"] as? Bool, true, "\(selector) is on the page: \(out)", file: file, line: line)
        return Self.rect(from: out)
    }

    /// Where a mark's badge is, in page-view points.
    func rect(markID: String, file: StaticString = #filePath, line: UInt = #line) throws -> CGRect {
        let out = try post("mark/rect", ["id": markID])
        XCTAssertEqual(out["found"] as? Bool, true, "the mark is on the page: \(out)", file: file, line: line)
        return Self.rect(from: out)
    }

    private static func rect(from out: [String: Any]) -> CGRect {
        CGRect(x: out["x"] as? Double ?? 0, y: out["y"] as? Double ?? 0,
               width: out["width"] as? Double ?? 0, height: out["height"] as? Double ?? 0)
    }

    // MARK: waiting

    /// Poll until the app agrees. A fixed sleep is either too short to be
    /// safe or too long to be quick; this is both.
    func waitUntil(_ what: String, timeout: TimeInterval = 15,
                   file: StaticString = #filePath, line: UInt = #line, _ ready: () throws -> Bool) throws {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if try ready() { return }
            Thread.sleep(forTimeInterval: 0.05)
        }
        guard try ready() else {
            XCTFail("timed out after \(Int(timeout))s waiting for \(what); state \((try? state()) ?? [:])",
                    file: file, line: line)
            return
        }
    }

    /// Poll, and say whether it came true. For a step that has a second
    /// way to go if it does not.
    func settles(_ timeout: TimeInterval = 5, _ ready: () throws -> Bool) rethrows -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if try ready() { return true }
            Thread.sleep(forTimeInterval: 0.05)
        }
        return try ready()
    }

    /// Poll until a key of the harness's state matches.
    func waitFor(_ key: String, _ want: String, timeout: TimeInterval = 15,
                 file: StaticString = #filePath, line: UInt = #line) throws {
        try waitUntil("\(key) to be \(want)", timeout: timeout, file: file, line: line) {
            (try? state()[key] as? String) == want
        }
    }

    /// The marks on the open chapter.
    func marks() throws -> [[String: Any]] {
        (try state()["marks"] as? [[String: Any]]) ?? []
    }

    /// The open document's state.
    func document() throws -> [String: Any] {
        (try state()["document"] as? [String: Any]) ?? [:]
    }

    /// A book open at its first chapter, with no marks left from an
    /// earlier run.
    func openBook(_ subject: String = "ai") throws {
        _ = app
        try waitUntil("the server to answer") { (try? self.state()["connected"] as? Bool) == true }
        try post("open", ["subject": subject])
        if (try state()["screen"] as? String) == "error" {
            // The first exchange from a container that has only just saved
            // its server can miss it; ask again before giving up.
            try post("server", ["url": Self.serverURL, "name": "dev"])
            try post("open", ["subject": subject])
        }
        // A server with fresh state starts the book at its placement
        // question and a calibration series; go through them to a chapter.
        for _ in 0..<8 {
            let screen = try state()["screen"] as? String ?? ""
            switch screen {
            case "placement": try post("place", ["level": 3])
            case "series", "pretest", "checking": try post("answer", ["mode": "correct"])
            case "results": try post("proceed")
            default: break
            }
            if screen == "reading" { break }
            Thread.sleep(forTimeInterval: 0.5)
        }
        try waitUntil("the reader's palette", timeout: 25) { self.app.buttons["Pen"].exists }
        try waitUntil("the chapter to be on the page") { try self.eval("!!document.querySelector('#chapter h1')") == "1" }
        for m in try marks() {
            if let id = m["id"] as? String { try post("unmark", ["id": id]) }
        }
        // The app is shared with the class's other tests, so the page keeps
        // what one of them typed into a beat and where it scrolled.
        try eval("""
            document.querySelectorAll('textarea, input').forEach(function (f) { f.value = ''; });
            if (document.activeElement) document.activeElement.blur();
            window.scrollTo(0, 0); 'ok'
            """)
    }
}
