import Foundation
import os

/// Hands-free verification of the full exchange loop, for use over USB when
/// nobody is present to tap through the UI. Launch with:
///
///     ios-deploy --bundle Chiron.app --args selftest --justlaunch
///
/// Progress is written to the system log (subsystem com.mjbraun.chiron), so
/// the Mac can watch it with `idevicesyslog | grep chiron-selftest`. Every step
/// drives the same code paths the real UI drives - no shortcuts around Sync.
@MainActor
enum SelfTest {
    static let log = Logger(subsystem: "com.mjbraun.chiron", category: "chiron-selftest")

    /// os.Logger output does not reach `idevicesyslog`, which reads the legacy
    /// syslog stream - so a run over USB looked like it had produced nothing at
    /// all. Everything goes to stdout as well, where ios-deploy relays it.
    static func say(_ line: String) {
        print("chiron-selftest \(line)")
        fflush(stdout)
        log.notice("\(line)")
    }

    static var requested: Bool {
        CommandLine.arguments.contains("selftest") || weakRequested
    }

    /// Answer everything wrong, to exercise the path that actually matters
    /// pedagogically: gate failure, grader diagnosis, and the remediation
    /// chapter. `simctl launch <dev> <id> selftestweak`.
    static var weakRequested: Bool {
        CommandLine.arguments.contains("selftestweak")
    }

    /// Jump straight to a screen for visual inspection, without tapping through
    /// the UI - screens like the check and the reader are otherwise several
    /// taps deep and unreachable from a script. Used from the simulator:
    /// `simctl launch <dev> <id> showreader` (also: showcheck, showpretest,
    /// showbreak). Optionally follow with a unit id, e.g. `showreader u5`.
    /// `server=http://127.0.0.1:8080` on the command line points the app at a
    /// different host for one launch. The simulator needs this: it shares the
    /// Mac's network stack, but a fresh app container has no local-network
    /// grant, so the Mac's LAN address (192.168.2.1) is unreachable while
    /// loopback is not.
    static var serverOverride: String? {
        CommandLine.arguments.first { $0.hasPrefix("server=") }?
            .replacingOccurrences(of: "server=", with: "")
    }

    /// `showteach` opens the Teach-me conversation directly. It is not part of
    /// inspectScreen because that path loads the bundled book first, and this
    /// screen has no chapter behind it.
    static var teachRequested: Bool {
        CommandLine.arguments.contains("showteach")
    }

    /// `showteach demo` seeds a transcript so the conversation layout can be
    /// inspected without typing - the same trick showreader/showcheck use to
    /// reach a screen that is otherwise several taps deep.
    static var teachDemo: Bool {
        teachRequested && CommandLine.arguments.contains("demo")
    }

    static var inspectScreen: String? {
        for name in ["showreader", "showcheck", "showpretest", "showbreak"]
        where CommandLine.arguments.contains(name) {
            return name
        }
        return nil
    }

    static func inspect(_ model: AppModel, screen: String) {
        model.startStatic()
        let chapters = model.staticChapters
        guard !chapters.isEmpty else {
            log.error("INSPECT no bundled chapters: \(model.errorMessage ?? "?")")
            return
        }
        // An explicit unit wins; otherwise pick the most math-dense chapter,
        // since that is where layout breaks first.
        let requested = CommandLine.arguments.first { $0.hasPrefix("u") && $0.count <= 3 }
        let chapter = chapters.first { $0.unit == requested }
            ?? chapters.max(by: { a, b in
                a.html.filter { $0 == "$" }.count < b.html.filter { $0 == "$" }.count
            })!
        model.loadChapterForInspection(chapter)

        switch screen {
        case "showcheck":   model.beginCheck()
        case "showpretest": model.showPretestForInspection()
        case "showbreak":   model.showBreakForInspection()
        default:            model.showReadingForInspection()
        }
        say("INSPECT \(screen) unit=\(chapter.unit) beats=\(chapter.beats.count) check=\(chapter.check.count)")
    }

    static func run(_ model: AppModel) async {
        say("START transport=\(model.sync.transport) base=\(model.sync.baseURL)")

        await model.sync.probe()
        say("PROBE connected=\(model.sync.connected) transport=\(model.sync.transport)")

        await model.refreshSubjects()
        say("SUBJECTS n=\(model.subjects.count)")
        guard let subject = model.subjects.first else {
            say("FAIL no subjects available")
            return
        }

        await model.openSubject(subject.id)
        await model.start()

        guard let chapter = model.chapter else {
            log.error("FAIL no chapter after start: \(model.errorMessage ?? "no error reported")")
            return
        }
        say("CHAPTER unit=\(chapter.unit) html=\(chapter.html.count) beats=\(chapter.beats.count) pretest=\(chapter.pretest.count) check=\(chapter.check.count)")

        // Strong path answers from the shipped reference answers; weak path
        // answers with a confident, plausible-sounding misconception, which is
        // exactly what the grader is built to catch.
        let weak = weakRequested
        let responses: [ItemResponse] = chapter.check.map { item in
            if item.kind == "mcq" {
                let options = item.reveal?.options ?? []
                let idx = weak
                    ? (options.firstIndex(where: { !$0.correct }) ?? 0)
                    : (options.firstIndex(where: { $0.correct }) ?? 0)
                return ItemResponse(itemId: item.id, response: nil,
                                    selectedIndex: idx, confidence: 4)
            }
            let answer = weak
                ? "Softmax gives the probability that each option is factually correct, so the model picks the true one."
                : (item.reveal?.answer ?? "see reference")
            return ItemResponse(itemId: item.id, response: answer,
                                selectedIndex: nil, confidence: 4)
        }
        log.notice("MODE \(weak ? "weak" : "strong")")
        say("CHECK submitting n=\(responses.count)")
        await model.submitCheck(responses)

        if case .gate(let gate, let results) = model.screen {
            let pct = Int((gate.score ?? 0) * 100)
            let passed = results.filter { $0.passed }.count
            say("GATE score=\(pct)% passed=\(gate.passed) items=\(passed)/\(results.count)")
            log.notice("NEXT unit=\(model.chapter?.unit ?? "none")")
            say("PASS self-test completed the full loop")
        } else {
            log.error("FAIL no gate after check: \(model.errorMessage ?? "no error reported")")
        }
    }
}
