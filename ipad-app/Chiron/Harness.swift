#if DEBUG
import Foundation
import Network

/// A script's hands on the app: a tiny HTTP server on localhost (the
/// simulator shares the Mac's loopback) that reports where the app is and
/// drives the same session calls the buttons do. Screens are reached the
/// way a learner reaches them; nothing is faked into view. Launch with the
/// `harness` argument (`harness_port=8088` to pick a port).
///
///   GET  /state              {screen, subject, unit, items, wait, error, shelf}
///   POST /open   {subject}   open a book from the shelf
///   POST /shelf              close the book
///   POST /start              start exchange from the top
///   POST /place  {level}     answer the placement screener
///   POST /check              enter the check from the reader
///   POST /contents           toggle the contents (sidebar or sheet)
///   POST /chrome             toggle the reader chrome, as a tap on the page does
///   POST /tool {tool}        pick a palette tool: none | pen | highlighter | ask | eraser
///   POST /mark {text, kind}  highlight the first occurrence of text (kind: highlight | question)
///   POST /ask {text, question}  mark text as a question and ask it
///   POST /close              close the ask card
///   POST /answer {mode}      answer every item of the current chapter:
///                            correct | idk | wrong | mixed (default correct;
///                            mixed inks one item, passes on one, types the rest)
///   POST /proceed            leave the results (or the break)
///   POST /override           override a failed gate
///   POST /reset              start the book over
///   POST /retry              the error screen's Try again
///
/// Every POST waits for the action to settle and returns the new state.
@MainActor
final class Harness {
    static let shared = Harness()
    static var requested: Bool { CommandLine.arguments.contains("harness") }
    static var port: UInt16 { UInt16(SelfTest.argument("harness_port=") ?? "") ?? 8087 }

    private var listener: NWListener?
    private weak var library: Library?

    func start(_ library: Library) {
        self.library = library
        guard let l = try? NWListener(using: .tcp, on: NWEndpoint.Port(rawValue: Self.port)!) else {
            SelfTest.say("HARNESS could not listen on \(Self.port)")
            return
        }
        listener = l
        l.newConnectionHandler = { [weak self] conn in
            conn.start(queue: .global())
            self?.receive(on: conn, buffer: Data())
        }
        l.start(queue: .global())
        SelfTest.say("HARNESS listening on \(Self.port)")
    }

    private nonisolated func receive(on conn: NWConnection, buffer: Data) {
        conn.receive(minimumIncompleteLength: 1, maximumLength: 1 << 20) { [weak self] data, _, done, err in
            guard let self, err == nil else { conn.cancel(); return }
            var buf = buffer
            if let data { buf.append(data) }
            if let request = HTTPRequest(raw: buf) {
                Task { @MainActor in await self.route(request, conn) }
            } else if !done {
                self.receive(on: conn, buffer: buf)
            } else {
                conn.cancel()
            }
        }
    }

    private func route(_ req: HTTPRequest, _ conn: NWConnection) async {
        let body = (try? JSONSerialization.jsonObject(with: req.body)) as? [String: Any] ?? [:]
        guard let library else { respond(conn, status: "500 Internal Server Error", body: nil); return }
        let session = library.session
        switch (req.method, req.path) {
        case ("GET", "/state"):
            break
        case ("POST", "/open"):
            if let id = body["subject"] as? String { await library.open(id) }
        case ("POST", "/shelf"):
            library.closeBook()
            await library.refresh()
        case ("POST", "/start"):
            await session?.start()
        case ("POST", "/place"):
            await session?.place(level: body["level"] as? Int ?? 3)
        case ("POST", "/check"):
            session?.beginCheck()
        case ("POST", "/contents"):
            session?.contentsShown.toggle()
        case ("POST", "/chrome"):
            session?.toggleChrome()
        case ("POST", "/tool"):
            if let s = session, let name = body["tool"] as? String, let t = BookSession.Tool(rawValue: name) {
                s.tool = t
            }
        case ("POST", "/mark"):
            if let s = session, let text = body["text"] as? String {
                let kind: Mark.Kind = (body["kind"] as? String) == "question" ? .question : .highlight
                await s.mark(text: text, kind: kind)
            }
        case ("POST", "/ask"):
            if let s = session, let text = body["text"] as? String, let q = body["question"] as? String {
                if await s.mark(text: text, kind: .question) != nil {
                    await s.ask(q)
                }
            }
        case ("POST", "/close"):
            session?.closeAsking()
        case ("POST", "/answer"):
            if let s = session, let ch = s.chapter {
                let mode = body["mode"] as? String ?? "correct"
                let responses: [ItemResponse]
                switch mode {
                case "idk":
                    responses = ch.check.map {
                        ItemResponse(itemId: $0.id, response: nil, selectedIndex: nil, confidence: 1, idk: true)
                    }
                case "wrong":
                    responses = SelfTest.answers(for: ch, weak: true, llm: library.sync.llmConnected)
                case "mixed":
                    // One of each: the first constructed item inked, the
                    // second passed on, the rest typed or chosen correctly.
                    var inked = false, passed = false
                    responses = SelfTest.answers(for: ch, weak: false, llm: library.sync.llmConnected).map { r in
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
                    responses = SelfTest.answers(for: ch, weak: false, llm: library.sync.llmConnected)
                }
                await s.submitCheck(responses)
            }
        case ("POST", "/proceed"):
            if let s = session {
                if case .takingBreak = s.screen {
                    await s.breakFinished(minutes: 0.1)
                } else {
                    await s.proceed()
                }
            }
        case ("POST", "/override"):
            await session?.override()
        case ("POST", "/reset"):
            await session?.startOver()
        case ("POST", "/retry"):
            await session?.retry()
        default:
            respond(conn, status: "404 Not Found", body: nil)
            return
        }
        respond(conn, status: "200 OK", body: state(library))
    }

    private func state(_ library: Library) -> Data {
        var out: [String: Any] = [
            "shelf": library.subjects.map(\.id),
            "active": library.activeSubjectID ?? "",
            "connected": library.sync.connected,
        ]
        if let s = library.session {
            out["subject"] = s.subjectID
            out["screen"] = screenName(s.screen)
            out["unit"] = s.chapter?.unit ?? ""
            out["items"] = s.chapter?.check.count ?? 0
            out["wait"] = s.wait?.rawValue ?? ""
            out["contents"] = s.contentsShown
            out["chrome_hidden"] = s.chromeHidden
            out["tool"] = s.tool.rawValue
            out["marks"] = s.marks.map { ["kind": $0.kind.rawValue, "text": $0.text, "answered": $0.answer != nil] }
            if let a = s.asking {
                out["asking"] = ["question": a.mark.question ?? "", "busy": a.busy, "answered": a.mark.answer != nil, "error": a.error ?? ""]
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
        return (try? JSONSerialization.data(withJSONObject: out)) ?? Data("{}".utf8)
    }

    private func screenName(_ screen: BookSession.Screen) -> String {
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

    private func respond(_ conn: NWConnection, status: String, body: Data?) {
        var head = "HTTP/1.1 \(status)\r\nContent-Type: application/json\r\n"
        head += "Content-Length: \(body?.count ?? 0)\r\nConnection: close\r\n\r\n"
        var out = Data(head.utf8)
        if let body { out.append(body) }
        conn.send(content: out, completion: .contentProcessed { _ in conn.cancel() })
    }
}

/// Just enough HTTP parsing for the harness.
private struct HTTPRequest {
    let method: String
    let path: String
    let body: Data

    init?(raw: Data) {
        guard let headerEnd = raw.range(of: Data("\r\n\r\n".utf8)) else { return nil }
        guard let head = String(data: raw[..<headerEnd.lowerBound], encoding: .utf8) else { return nil }
        let lines = head.components(separatedBy: "\r\n")
        let parts = lines[0].components(separatedBy: " ")
        guard parts.count >= 2 else { return nil }
        method = parts[0]
        path = parts[1]
        var contentLength = 0
        for line in lines.dropFirst() where line.lowercased().hasPrefix("content-length:") {
            contentLength = Int(line.dropFirst("content-length:".count).trimmingCharacters(in: .whitespaces)) ?? 0
        }
        let bodyData = raw[headerEnd.upperBound...]
        guard bodyData.count >= contentLength else { return nil }  // wait for more
        body = Data(bodyData.prefix(contentLength))
    }
}
#endif
