import XCTest

/// The shell as typed at: a hardware keyboard's keys reach the far
/// shell, arrows included, and what comes back is shown as sent. Needs a
/// gate with an sshd behind it (CHIRON_GATE and CHIRON_GATE_KEY in the
/// runner's environment); without one the test is skipped.
final class ShellUITests: XCTestCase {
    private let harness = URL(string: "http://localhost:8087")!

    private func state() throws -> [String: Any] {
        let data = try Data(contentsOf: harness.appendingPathComponent("state"))
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func get(_ path: String) throws -> [String: Any] {
        let data = try Data(contentsOf: harness.appendingPathComponent(path))
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func post(_ path: String, _ body: [String: Any]) throws {
        var req = URLRequest(url: harness.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        let done = expectation(description: path)
        URLSession.shared.dataTask(with: req) { _, _, _ in done.fulfill() }.resume()
        wait(for: [done], timeout: 60)
    }

    private func screen() throws -> [String] {
        (try get("shell/screen")["lines"] as? [String]) ?? []
    }

    func testArrowKeysAndUnicodeReachTheShell() throws {
        let env = ProcessInfo.processInfo.environment
        guard let gate = env["CHIRON_GATE"] else { throw XCTSkip("no CHIRON_GATE in the environment") }
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["harness"]
        app.launchEnvironment["CHIRON_SERVER"] = env["CHIRON_SERVER"] ?? "http://localhost:8084"
        // The sshd behind a Mac gate signs the Mac's user in, not "sprite".
        if let user = env["CHIRON_SHELL_USER"] { app.launchEnvironment["CHIRON_SHELL_USER"] = user }
        app.launch()
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, (try? state()) == nil { Thread.sleep(forTimeInterval: 0.5) }

        try post("server", ["url": gate, "key": env["CHIRON_GATE_KEY"] ?? "", "name": "gate"])
        try post("shell", [:])
        let connected = Date().addingTimeInterval(60)
        var phase = ""
        while Date() < connected {
            phase = (try? get("shell/screen")["phase"] as? String) ?? ""
            if phase == "connected" || phase.hasPrefix("closed") { break }
            Thread.sleep(forTimeInterval: 0.5)
        }
        XCTAssertEqual(phase, "connected", "the shell connected through the gate")
        Thread.sleep(forTimeInterval: 1.5)

        // A command typed, its output seen. Typed through the harness:
        // XCUI keystrokes wait for the terminal's caret to stop moving,
        // which it never does, and take minutes.
        let tag = "up-test-\(Int.random(in: 100...999))"
        try post("shell/type", ["text": "echo \(tag)\n"])
        Thread.sleep(forTimeInterval: 1.5)
        var lines = try screen()
        XCTAssertTrue(lines.contains(where: { $0.hasSuffix(tag) && !$0.contains("echo") }),
                      "the command ran: \(lines.filter { !$0.isEmpty })")

        // The up arrow from the keyboard recalls it: the same line, at the
        // prompt again.
        app.typeKey(.upArrow, modifierFlags: [])
        Thread.sleep(forTimeInterval: 1)
        lines = try screen()
        let recalled = lines.filter { $0.contains("echo \(tag)") }.count
        XCTAssertEqual(recalled, 2, "the up arrow recalled the line: \(lines.filter { !$0.isEmpty })")
        // Control-C drops the recalled line.
        try post("shell/type", ["text": "\u{03}"])
        Thread.sleep(forTimeInterval: 0.5)

        // Bytes beyond ASCII come back as the glyphs they are.
        try post("shell/type", ["text": "printf '\\xe2\\x8f\\xb5\\xe2\\x8f\\xb5 glyphs\\n'\n"])
        Thread.sleep(forTimeInterval: 1.5)
        lines = try screen()
        XCTAssertTrue(lines.contains(where: { $0.hasPrefix("⏵⏵ glyphs") }),
                      "the terminal shows the glyphs: \(lines.filter { !$0.isEmpty })")
        try post("shell/close", [:])
    }
}
