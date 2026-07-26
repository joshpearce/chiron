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

    static var requested: Bool {
        CommandLine.arguments.contains("selftest")
    }

    static func run(_ model: AppModel) async {
        log.notice("START transport=\(model.sync.transport, privacy: .public) base=\(model.sync.baseURL, privacy: .public)")

        await model.sync.probe()
        log.notice("PROBE connected=\(model.sync.connected, privacy: .public) transport=\(model.sync.transport, privacy: .public)")

        await model.refreshSubjects()
        log.notice("SUBJECTS n=\(model.subjects.count, privacy: .public)")
        guard let subject = model.subjects.first else {
            log.error("FAIL no subjects available")
            return
        }

        await model.openSubject(subject.id)
        await model.start()

        guard let chapter = model.chapter else {
            log.error("FAIL no chapter after start: \(model.errorMessage ?? "no error reported", privacy: .public)")
            return
        }
        log.notice("CHAPTER unit=\(chapter.unit, privacy: .public) html=\(chapter.html.count, privacy: .public) beats=\(chapter.beats.count, privacy: .public) pretest=\(chapter.pretest.count, privacy: .public) check=\(chapter.check.count, privacy: .public)")

        // Answer the terminal check from the shipped reference answers - this
        // is the strong-learner path, so the gate should pass.
        let responses: [ItemResponse] = chapter.check.map { item in
            if item.kind == "mcq" {
                let correct = item.reveal?.options?.firstIndex(where: { $0.correct }) ?? 0
                return ItemResponse(itemId: item.id, response: nil,
                                    selectedIndex: correct, confidence: 4)
            }
            return ItemResponse(itemId: item.id,
                                response: item.reveal?.answer ?? "see reference",
                                selectedIndex: nil, confidence: 4)
        }
        log.notice("CHECK submitting n=\(responses.count, privacy: .public)")
        await model.submitCheck(responses)

        if case .gate(let gate, let results) = model.screen {
            let pct = Int((gate.score ?? 0) * 100)
            let passed = results.filter { $0.passed }.count
            log.notice("GATE score=\(pct, privacy: .public)% passed=\(gate.passed, privacy: .public) items=\(passed, privacy: .public)/\(results.count, privacy: .public)")
            log.notice("NEXT unit=\(model.chapter?.unit ?? "none", privacy: .public)")
            log.notice("PASS self-test completed the full loop")
        } else {
            log.error("FAIL no gate after check: \(model.errorMessage ?? "no error reported", privacy: .public)")
        }
    }
}
