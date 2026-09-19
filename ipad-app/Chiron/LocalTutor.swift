import Foundation
#if canImport(FoundationModels)
import FoundationModels
#endif

/// Something that can answer a reader's question about a passage, given
/// the chapter it sits in.
protocol PassageAnswerer: AnyObject {
    func answer(question: String, about quote: String, in chapter: String, history: [QA]) async throws -> String
}

/// The model on the device, for when the tutor cannot be reached: it
/// answers from the chapter alone, and the reader is told it was the
/// device that answered. A small model reading a few thousand words,
/// so it is the fallback and never the first choice.
final class LocalTutor: PassageAnswerer {
    /// The device's model, when it has one and it is ready.
    static var ifAvailable: LocalTutor? {
        #if canImport(FoundationModels)
        if #available(iOS 26, macOS 26, *), case .available = SystemLanguageModel.default.availability {
            return LocalTutor()
        }
        #endif
        return nil
    }

    /// What the reader sees under an answer the device wrote.
    static let signature = "_Answered on this device from the chapter alone, while the tutor was out of reach._"

    private static let instructions = """
        You are helping someone read a textbook chapter. They have highlighted a passage and asked a question about it. \
        Answer in two to four sentences, plainly, using the chapter as your source and saying so when the chapter does \
        not settle the question. Write $...$ for any mathematics. Do not repeat the passage back.
        """

    func answer(question: String, about quote: String, in chapter: String, history: [QA]) async throws -> String {
        #if canImport(FoundationModels)
        guard #available(iOS 26, macOS 26, *) else { throw Unavailable() }
        let model = SystemLanguageModel.default
        let session = LanguageModelSession(model: model, instructions: Self.instructions)
        var prompt = "The chapter:\n\n\(Self.fit(chapter, within: Self.budget(of: model)))\n\nThe highlighted passage: \"\(quote)\"\n\n"
        for qa in history where qa.answer != nil {
            prompt += "Earlier question: \(qa.question)\nEarlier answer: \(qa.answer!)\n\n"
        }
        prompt += "Question: \(question)"
        return try await session.respond(to: prompt).content
        #else
        throw Unavailable()
        #endif
    }

    struct Unavailable: Error {}

    #if canImport(FoundationModels)
    /// Characters of chapter that fit beside the instructions, the passage
    /// and the answer: about three characters to a token, and a quarter
    /// of the window kept back for everything that is not the chapter.
    @available(iOS 26, macOS 26, *)
    private static func budget(of model: SystemLanguageModel) -> Int {
        let tokens: Int
        if #available(iOS 27, macOS 27, *) { tokens = model.contextSize } else { tokens = 4096 }
        return tokens * 3 * 3 / 4
    }
    #endif

    /// The chapter cut to the budget at a paragraph break, the passage's
    /// paragraph kept whichever end it falls at.
    static func fit(_ chapter: String, within budget: Int) -> String {
        guard chapter.count > budget else { return chapter }
        let head = String(chapter.prefix(budget))
        if let cut = head.range(of: "\n\n", options: .backwards) {
            return String(head[..<cut.lowerBound])
        }
        return head
    }
}
