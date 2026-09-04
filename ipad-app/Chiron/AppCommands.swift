import Foundation
import PencilKit
import UIKit

/// The verbs a script or the sprite's agent can run against the app. They
/// drive the same session calls the buttons do, so every screen is reached
/// the way a learner reaches it. The debug harness and the agent link both
/// dispatch here.
@MainActor
enum AppCommands {
    enum Failure: Error, CustomStringConvertible {
        case unknownVerb(String), noBook, badArguments(String)
        var description: String {
            switch self {
            case .unknownVerb(let v): return "unknown verb \(v)"
            case .noBook: return "no book is open"
            case .badArguments(let why): return why
            }
        }
    }

    /// Run one verb; the result is the app's state afterwards, or for
    /// verbs that answer a question, that answer.
    static func run(_ verb: String, args: [String: Any], library: Library) async throws -> [String: Any] {
        switch verb {
        case "state":
            break
        case "open":
            guard let id = args["subject"] as? String else { throw Failure.badArguments("open needs subject") }
            await library.open(id)
        case "shelf":
            library.closeBook()
            await library.refresh()
        case "screenshot":
            return ["png_b64": try screenshot().base64EncodedString()]
        case "server":
            guard let url = args["url"] as? String else { throw Failure.badArguments("server needs url") }
            library.sync.servers.add(name: args["name"] as? String ?? "", url: url, key: args["key"] as? String)
            library.sync.baseURL = url
            await library.sync.probe()
            // As the settings sheet does on save: a key means enrol.
            if let key = args["key"] as? String, !key.isEmpty {
                _ = try? await library.sync.enrolDeviceKey(name: DeviceKeySection.deviceName)
            }
        case "agent":
            let on = args["on"] as? Bool ?? true
            library.agent.enabled = on
            if on { library.agent.start() } else { library.agent.stop() }
            try? await Task.sleep(nanoseconds: 500_000_000)
        case "capture":
            guard let prompt = args["prompt"] as? String else { throw Failure.badArguments("capture needs prompt") }
            let c = Capture(text: args["text"] as? String ?? "", sourceURL: args["url"] as? String, sourceApp: args["app"] as? String)
            let id = try await library.submitCapture(c, prompt: prompt)
            var out = state(library)
            out["subject"] = id
            return out
        case "capture/card":
            // The card as the share extension would open it.
            let c = Capture(text: args["text"] as? String ?? "", sourceURL: args["url"] as? String, sourceApp: args["app"] as? String)
            try CaptureInbox.write(c)
            library.receiveCapture(id: c.id)
        case "capture/close":
            library.pendingCapture = nil
        case "note":
            guard let s = library.session else { throw Failure.noBook }
            guard let text = args["text"] as? String, let note = args["note"] as? String else {
                throw Failure.badArguments("note needs text and note")
            }
            if await s.mark(text: text, kind: .note) != nil {
                await s.extend(note)
            }
        case "shell":
            library.shellShown = true
            try? await Task.sleep(nanoseconds: 300_000_000)
        case "shell/close":
            library.shell.close()
            library.shellShown = false
        case "shell/type":
            guard let text = args["text"] as? String else { throw Failure.badArguments("shell/type needs text") }
            library.shell.send(text: text)
            try? await Task.sleep(nanoseconds: 300_000_000)
        case "mark/rect":
            guard let s = library.session, let id = args["id"] as? String else { throw Failure.badArguments("mark/rect needs id") }
            guard let r = await s.page?.rect(of: id) else { return ["found": false] }
            return ["found": true, "x": r.minX, "y": r.minY, "width": r.width, "height": r.height]
        case "shell/screen":
            var out: [String: Any] = ["phase": shellPhase(library.shell.phase)]
            #if DEBUG
            if let lines = ShellScreen.shared.lines() { out["lines"] = lines }
            #endif
            return out
        default:
            guard let session = library.session else { throw Failure.noBook }
            try await run(verb, args: args, session: session, llm: library.sync.llmConnected)
        }
        return state(library)
    }

    /// The verbs that act on the open book.
    static func run(_ verb: String, args: [String: Any], session: BookSession, llm: Bool) async throws {
        switch verb {
        case "start":
            await session.start()
        case "place":
            await session.place(level: args["level"] as? Int ?? 3)
        case "check":
            session.beginCheck()
        case "contents":
            session.contentsShown.toggle()
        case "chrome":
            session.toggleChrome()
        case "tool":
            guard let name = args["tool"] as? String, let t = BookSession.Tool(rawValue: name) else {
                throw Failure.badArguments("tool must be none | pen | highlighter | ask | eraser")
            }
            session.tool = t
        case "pen":
            guard let name = args["color"] as? String, let c = BookSession.PenColor(rawValue: name) else {
                throw Failure.badArguments("pen color must be red | blue | green | black")
            }
            session.penColor = c
        case "mark":
            guard let text = args["text"] as? String else { throw Failure.badArguments("mark needs text") }
            let kind: Mark.Kind = (args["kind"] as? String) == "question" ? .question : .highlight
            await session.mark(text: text, kind: kind)
        case "ask":
            guard let text = args["text"] as? String, let q = args["question"] as? String else {
                throw Failure.badArguments("ask needs text and question")
            }
            if await session.mark(text: text, kind: .question) != nil {
                await session.ask(q)
            }
        case "close":
            session.closeAsking()
        case "delete":
            session.deleteAsking()
        case "unmark":
            guard let id = args["id"] as? String else { throw Failure.badArguments("unmark needs id") }
            session.removeMark(id)
        case "answer":
            guard let ch = session.chapter else { throw Failure.noBook }
            await session.submitCheck(answers(for: ch, mode: args["mode"] as? String ?? "correct", llm: llm))
        case "proceed":
            if case .takingBreak = session.screen {
                await session.breakFinished(minutes: 0.1)
            } else {
                await session.proceed()
            }
        case "override":
            await session.override()
        case "reset":
            await session.startOver()
        case "retry":
            await session.retry()
        default:
            throw Failure.unknownVerb(verb)
        }
    }

    /// Answers for every item of the chapter: correct | idk | wrong | mixed
    /// (mixed inks one item, passes on one, types the rest).
    static func answers(for ch: ChapterPayload, mode: String, llm: Bool) -> [ItemResponse] {
        switch mode {
        case "idk":
            return ch.check.map {
                ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
            }
        case "wrong":
            return SelfTest.answers(for: ch, weak: true, llm: llm)
        case "mixed":
            var inked = false, passed = false
            return SelfTest.answers(for: ch, weak: false, llm: llm).map { r in
                var r = r
                guard r.selectedIndex == nil else { return r }
                if !inked {
                    inked = true
                    r.response = nil
                    r.idk = nil
                    r.ink = InkAnswer(strokes: [
                        [InkAnswer.Point(x: 0.1, y: 0.3), InkAnswer.Point(x: 0.3, y: 0.7), InkAnswer.Point(x: 0.5, y: 0.3)],
                        [InkAnswer.Point(x: 0.6, y: 0.2), InkAnswer.Point(x: 0.6, y: 0.8)],
                    ], aspect: InkBox.aspect)
                } else if !passed {
                    passed = true
                    r.response = nil
                    r.idk = true
                    r.confidence = 1
                }
                return r
            }
        default:
            return SelfTest.answers(for: ch, weak: false, llm: llm)
        }
    }

    static func state(_ library: Library) -> [String: Any] {
        var out: [String: Any] = [
            "shelf": library.subjects.map(\.id),
            "active": library.activeSubjectID ?? "",
            "connected": library.sync.connected,
            "agent": library.agent.connected,
            "capture_card": library.pendingCapture != nil,
            "shelf_rows": library.subjects.map { ["id": $0.id, "kind": $0.kind ?? "book", "status": $0.status ?? ""] },
        ]
        if let s = library.session {
            out["subject"] = s.subjectID
            out["kind"] = s.kind
            out["screen"] = screenName(s.screen)
            out["unit"] = s.chapter?.unit ?? ""
            out["items"] = s.chapter?.check.count ?? 0
            out["wait"] = s.wait?.rawValue ?? ""
            out["contents"] = s.contentsShown
            out["chrome_hidden"] = s.chromeHidden
            out["tool"] = s.tool.rawValue
            out["pen_color"] = s.penColor.rawValue
            out["ink_strokes"] = s.inkData.flatMap { try? PKDrawing(data: $0) }?.strokes.count ?? 0
            out["position"] = s.chapter.map { s.position(for: $0.unit) } ?? 0
            #if DEBUG
            out["canvas_pen"] = ReaderView.Coordinator.probe?.canvasPen ?? ""
            out["canvas_touches"] = ReaderView.Coordinator.probe?.canvasTouches ?? -1
            out["pencil"] = s.lastPencil
            out["canvas_frame"] = ReaderView.Coordinator.probe?.canvasFrame ?? ""
            #endif
            out["marks"] = s.marks.map { ["id": $0.id, "kind": $0.kind.rawValue, "text": $0.text, "answered": $0.answer != nil, "turns": $0.history.count] }
            if let a = s.asking {
                out["asking"] = ["question": a.mark.question ?? "", "busy": a.busy, "answered": a.mark.answer != nil, "error": a.error ?? "", "turns": a.mark.history.count]
            }
            out["error"] = s.errorMessage ?? ""
            if case .results(let doc, _) = s.screen {
                out["headline"] = doc.headline
                out["entries"] = doc.entries.count
            }
            if case .error(let message) = s.screen { out["error"] = message }
        } else {
            out["screen"] = library.teaching ? "teach" : "bookshelf"
        }
        return out
    }

    static func screenName(_ screen: BookSession.Screen) -> String {
        switch screen {
        case .empty: return "empty"
        case .placement: return "placement"
        case .series: return "series"
        case .reading: return "reading"
        case .pretest: return "pretest"
        case .check: return "check"
        case .results: return "results"
        case .authoring: return "authoring"
        case .takingBreak: return "break"
        case .error: return "error"
        }
    }

    static func shellPhase(_ phase: ShellSession.Phase) -> String {
        switch phase {
        case .idle: return "idle"
        case .connecting: return "connecting"
        case .connected: return "connected"
        case .closed(let err): return "closed" + (err.map { ": " + $0 } ?? "")
        }
    }

    /// The window as the reader sees it, as PNG.
    static func screenshot() throws -> Data {
        guard let window = UIApplication.shared.connectedScenes
            .compactMap({ $0 as? UIWindowScene }).flatMap(\.windows).first(where: \.isKeyWindow) else {
            throw Failure.badArguments("no window to capture")
        }
        let renderer = UIGraphicsImageRenderer(bounds: window.bounds)
        let image = renderer.image { _ in
            window.drawHierarchy(in: window.bounds, afterScreenUpdates: true)
        }
        guard let png = image.pngData() else { throw Failure.badArguments("could not encode the screenshot") }
        return png
    }
}
