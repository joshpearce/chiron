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
    let nextAction: String?

    var id: String { unit }

    enum CodingKeys: String, CodingKey {
        case unit, title, minutes, html, beats, pretest, check
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

    enum CodingKeys: String, CodingKey {
        case itemId = "item_id"
        case response
        case selectedIndex = "selected_index"
        case confidence
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

    enum CodingKeys: String, CodingKey {
        case subject, phase, unit, override, choice
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

    enum CodingKeys: String, CodingKey {
        case results, gate, chapter, state
        case breakSuggestion = "break_suggestion"
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

    enum CodingKeys: String, CodingKey {
        case score, passed, gate
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
