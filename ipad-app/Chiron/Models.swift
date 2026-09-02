import Foundation

// Chapter payloads carry pieces the webview owns (html, beats) as opaque JSON
// and pieces native UI owns (check items) as typed models.

struct ChapterPayload: Codable, Identifiable {
    let unit: String
    let title: String
    let minutes: Int
    let html: String
    let beats: [JSONValue]
    let pretest: [CheckItem]
    let check: [CheckItem]
    // True for the placement screener and the calibration series: no reveal,
    // no gate verdict - the score is measurement.
    let calibration: Bool?
    let nextAction: String?

    var id: String { unit }
    var isCalibration: Bool { calibration == true }
    /// The placement screener travels as a one-item calibration chapter.
    var screener: CheckItem? {
        check.count == 1 && check[0].check == "screener" ? check[0] : nil
    }

    enum CodingKeys: String, CodingKey {
        case unit, title, minutes, html, beats, pretest, check, calibration
        case nextAction = "next_action"
    }
}

struct CheckItem: Codable, Identifiable {
    let id: String
    let unit: String?
    let concept: String?
    let kind: String            // constructed | mcq
    let prompt: String
    let check: String
    let difficulty: String?
    let options: [MCQOption]?
    let reveal: Reveal?

    struct MCQOption: Codable { let text: String }
    struct Reveal: Codable {
        let answer: String?
        let rubric: String?
        let options: [OptionReveal]?
        struct OptionReveal: Codable {
            let explain: String
            let correct: Bool
        }

        // Compute items answer with a bare number (`answer: 6`). Strict String
        // decoding turns one such item into a total decode failure for the
        // whole book, so accept either shape and normalise to text. The server
        // also stringifies these; this is the belt to that suspenders.
        init(from decoder: Decoder) throws {
            let c = try decoder.container(keyedBy: CodingKeys.self)
            answer = try c.decodeLenientString(forKey: .answer)
            rubric = try c.decodeLenientString(forKey: .rubric)
            options = try c.decodeIfPresent([OptionReveal].self, forKey: .options)
        }
    }
}

struct ItemResponse: Codable {
    let itemId: String
    var response: String?
    var selectedIndex: Int?
    var confidence: Int
    // An explicit "I don't know - move on". The server grades it as a fail
    // without a model call.
    var idk: Bool?

    enum CodingKeys: String, CodingKey {
        case itemId = "item_id"
        case response
        case selectedIndex = "selected_index"
        case confidence
        case idk
    }
}

struct BeatResponse: Codable {
    let beatId: String
    let response: String
    var selfVerdict: String?
    var mechanicalVerdict: String?

    enum CodingKeys: String, CodingKey {
        case beatId = "beat_id"
        case response
        case selfVerdict = "self_verdict"
        case mechanicalVerdict = "mechanical_verdict"
    }
}

struct SubjectInfo: Codable, Identifiable {
    let id: String
    let title: String
    let unitsTotal: Int?
    let unitsCleared: Int?
    let currentUnit: String?
    let debt: Int?

    var progressLine: String {
        var parts: [String] = []
        if let c = unitsCleared, let t = unitsTotal { parts.append("\(c)/\(t) units") }
        if let u = currentUnit { parts.append("reading \(u)") }
        if let d = debt, d > 0 { parts.append("\(d) in debt") }
        return parts.isEmpty ? "not started" : parts.joined(separator: " · ")
    }

    enum CodingKeys: String, CodingKey {
        case id, title, debt
        case unitsTotal = "units_total"
        case unitsCleared = "units_cleared"
        case currentUnit = "current_unit"
    }
}

/// GET /subjects: the shelf, and which book the reader last had open.
struct SubjectsResponse: Codable {
    let subjects: [SubjectInfo]
    let active: String?

    /// The server sends "" before any book has been opened.
    var activeID: String? { (active?.isEmpty ?? true) ? nil : active }
}

struct ExchangeRequest: Codable {
    var subject: String = "ai"
    var phase: String = "boundary"
    var unit: String?
    var beatResponses: [BeatResponse] = []
    var pretestResponses: [ItemResponse] = []
    var checkResponses: [ItemResponse] = []
    var override: Bool = false
    var skippedCheck: Bool = false
    var catchMeUp: Bool = false
    var choice: String?
    var chunkMinutes: Double?
    var breakMinutes: Double?
    // Grades come back at once; the next chapter is authored in the
    // background and fetched from /chapter. A request that has to outlive
    // the app being backgrounded is a request that gets lost.
    var async: Bool = true

    enum CodingKeys: String, CodingKey {
        case subject, phase, unit, override, choice, async
        case beatResponses = "beat_responses"
        case pretestResponses = "pretest_responses"
        case checkResponses = "check_responses"
        case skippedCheck = "skipped_check"
        case catchMeUp = "catch_me_up"
        case chunkMinutes = "chunk_minutes"
        case breakMinutes = "break_minutes"
    }
}

struct ExchangeResponse: Codable {
    let results: [GradeResult]
    let gate: Gate?
    let chapter: ChapterPayload?
    let state: BookState
    let breakSuggestion: BreakSuggestion?
    /// The unit being authored in the background, when no chapter came.
    let authoring: String?
    /// The graded check as the typeset results pages say it.
    let resultsDoc: ResultsDoc?

    enum CodingKeys: String, CodingKey {
        case results, gate, chapter, state, authoring
        case breakSuggestion = "break_suggestion"
        case resultsDoc = "results_doc"
    }
}

/// GET /chapter/{subject}: the persisted current chapter, if any, and
/// whether the server is still writing the next one.
struct ChapterStatus: Codable {
    let chapter: ChapterPayload?
    let authoring: Bool
    let authoringError: String?

    enum CodingKeys: String, CodingKey {
        case chapter, authoring
        case authoringError = "authoring_error"
    }
}

/// The results document: headline, framing copy, and one entry per graded
/// item with the full audit trail. The same content the tablet typesets.
struct ResultsDoc: Codable {
    let unit: String
    let headLeft: String
    let calibration: Bool?
    let score: Double
    let gate: Double
    let passed: Bool
    let extensionUnlocked: Bool?
    let action: String
    let dek: String
    let headline: String
    let tally: String?
    let entries: [ResultsEntry]

    var isCalibration: Bool { calibration == true }

    enum CodingKeys: String, CodingKey {
        case unit, calibration, score, gate, passed, action, dek, headline, tally, entries
        case headLeft = "head_left"
        case extensionUnlocked = "extension_unlocked"
    }
}

struct ResultsEntry: Codable, Identifiable {
    let n: Int
    let verdict: String
    let idk: Bool?
    let kind: String
    let prompt: String
    let confidence: Int?
    let readAs: String?
    let chose: String?
    let answer: String?
    let why: String?

    var id: Int { n }
    var isIDK: Bool { idk == true }
    var passed: Bool { verdict == "pass" || verdict == "valid_alternative_path" }

    enum CodingKeys: String, CodingKey {
        case n, verdict, idk, kind, prompt, confidence, chose, answer, why
        case readAs = "read_as"
    }
}

struct GradeResult: Codable, Identifiable {
    let itemId: String
    let verdict: String
    let misconceptions: [String]?
    let feedbackMd: String?
    let confidence: Int?

    var id: String { itemId }
    var passed: Bool { verdict == "pass" || verdict == "valid_alternative_path" }

    enum CodingKeys: String, CodingKey {
        case itemId = "item_id"
        case verdict, misconceptions, confidence
        case feedbackMd = "feedback_md"
    }
}

struct Gate: Codable {
    let score: Double?
    let passed: Bool
    let gate: Double
    let extensionUnlocked: Bool
    // A calibration result: always passed, the score is measurement.
    var calibration: Bool?

    enum CodingKeys: String, CodingKey {
        case score, passed, gate, calibration
        case extensionUnlocked = "extension_unlocked"
    }
}

struct BookState: Codable {
    let spine: [SpineEntry]
    let fringe: [String]
    let debt: [JSONValue]
    let activeMisconceptions: [String]
    let summary: String
    let sessionMinutes: Double

    enum CodingKeys: String, CodingKey {
        case spine, fringe, debt, summary
        case activeMisconceptions = "active_misconceptions"
        case sessionMinutes = "session_minutes"
    }
}

struct SpineEntry: Codable, Identifiable {
    let unit: String
    let title: String
    let status: String   // locked|available|active|passed|overridden|failed
    let score: Double?
    let inFringe: Bool

    var id: String { unit }

    enum CodingKeys: String, CodingKey {
        case unit, title, status, score
        case inFringe = "in_fringe"
    }
}

struct BreakSuggestion: Codable {
    let minutes: Int
    let kind: String
    let note: String
}

extension KeyedDecodingContainer {
    /// Decode a field that should be text but may arrive as a number or bool.
    func decodeLenientString(forKey key: Key) throws -> String? {
        if let s = try? decodeIfPresent(String.self, forKey: key) { return s }
        if let d = try? decode(Double.self, forKey: key) {
            return d == d.rounded() ? String(Int(d)) : String(d)
        }
        if let b = try? decode(Bool.self, forKey: key) { return b ? "true" : "false" }
        return nil
    }
}

// Minimal JSON passthrough for payloads the webview owns.
enum JSONValue: Codable {
    case null, bool(Bool), number(Double), string(String)
    case array([JSONValue]), object([String: JSONValue])

    init(from decoder: Decoder) throws {
        let c = try decoder.singleValueContainer()
        if c.decodeNil() { self = .null }
        else if let b = try? c.decode(Bool.self) { self = .bool(b) }
        else if let n = try? c.decode(Double.self) { self = .number(n) }
        else if let s = try? c.decode(String.self) { self = .string(s) }
        else if let a = try? c.decode([JSONValue].self) { self = .array(a) }
        else { self = .object(try c.decode([String: JSONValue].self)) }
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.singleValueContainer()
        switch self {
        case .null: try c.encodeNil()
        case .bool(let b): try c.encode(b)
        case .number(let n): try c.encode(n)
        case .string(let s): try c.encode(s)
        case .array(let a): try c.encode(a)
        case .object(let o): try c.encode(o)
        }
    }
}
