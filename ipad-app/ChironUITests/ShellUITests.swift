import XCTest

/// The shell as typed at: a hardware keyboard's keys reach the far
/// shell, arrows included, and what comes back is shown as sent. Needs a
/// gate with an sshd behind it (CHIRON_GATE and CHIRON_GATE_KEY in the
/// runner's environment); without one the test is skipped.
final class ShellUITests: HarnessTestCase {
    /// The sshd behind a Mac gate signs the Mac's user in, not "sprite".
    override var launchEnvironment: [String: String] {
        guard let user = ProcessInfo.processInfo.environment["CHIRON_SHELL_USER"] else { return [:] }
        return ["CHIRON_SHELL_USER": user]
    }

    private func screen() throws -> [String] {
        (try get("shell/screen")["lines"] as? [String]) ?? []
    }

    func testArrowKeysAndUnicodeReachTheShell() throws {
        let env = ProcessInfo.processInfo.environment
        guard let gate = env["CHIRON_GATE"] else { throw XCTSkip("no CHIRON_GATE in the environment") }
        _ = app
        try post("server", ["url": gate, "key": env["CHIRON_GATE_KEY"] ?? "", "name": "gate"])
        try post("shell")
        try waitUntil("the shell to settle", timeout: 60) {
            let phase = (try? self.get("shell/screen")["phase"] as? String) ?? ""
            return phase == "connected" || phase.hasPrefix("closed")
        }
        XCTAssertEqual(try get("shell/screen")["phase"] as? String, "connected", "the shell connected through the gate")
        try waitUntil("a prompt") { try !self.screen().allSatisfy(\.isEmpty) }

        // A command typed, its output seen. Typed through the harness:
        // XCUI keystrokes wait for the terminal's caret to stop moving,
        // which it never does, and take minutes.
        let tag = "up-test-\(Int.random(in: 100...999))"
        try post("shell/type", ["text": "echo \(tag)\n"])
        try waitUntil("the command to run") {
            try self.screen().contains { $0.hasSuffix(tag) && !$0.contains("echo") }
        }

        // The up arrow from the keyboard recalls it: the same line, at the
        // prompt again.
        app.typeKey(.upArrow, modifierFlags: [])
        try waitUntil("the up arrow to recall the line") {
            try self.screen().filter { $0.contains("echo \(tag)") }.count == 2
        }
        // Control-C drops the recalled line.
        try post("shell/type", ["text": "\u{03}"])
        try waitUntil("the recalled line to be dropped") {
            try self.screen().filter { $0.contains("echo \(tag)") }.count < 3
        }

        // Bytes beyond ASCII come back as the glyphs they are.
        try post("shell/type", ["text": "printf '\\xe2\\x8f\\xb5\\xe2\\x8f\\xb5 glyphs\\n'\n"])
        try waitUntil("the terminal to show the glyphs") {
            try self.screen().contains { $0.hasPrefix("⏵⏵ glyphs") }
        }
        try post("shell/close")
    }
}
