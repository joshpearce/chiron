import AppIntents
import Combine
import Foundation

/// Where an intent's capture lands: the app is already running when an
/// intent performs, so the id goes straight to the library.
enum CaptureRouter {
    static let arrivals = PassthroughSubject<String, Never>()
}

/// "Make a Chiron primer" in Shortcuts, Spotlight, the Pencil Pro squeeze
/// menu and the text-selection bar: whatever is selected or handed over
/// becomes a capture, and the app opens on the question.
struct CapturePrimerIntent: AppIntent {
    static var title: LocalizedStringResource = "Make a Chiron primer"
    static var description = IntentDescription("Capture text or an image and ask Chiron to write a primer about it.")
    static var openAppWhenRun = true

    @Parameter(title: "Text", inputOptions: String.IntentInputOptions(multiline: true))
    var text: String?

    @Parameter(title: "Image", supportedContentTypes: [.png, .jpeg, .image])
    var image: IntentFile?

    @Parameter(title: "Source")
    var source: String?

    static var parameterSummary: some ParameterSummary {
        Summary("Make a primer from \(\.$text)") {
            \.$image
            \.$source
        }
    }

    @MainActor
    func perform() async throws -> some IntentResult {
        var c = Capture(text: text ?? "", sourceApp: source ?? "Shortcuts")
        if let image {
            c.imagePNG = image.data
        }
        guard !c.isEmpty else {
            throw $text.needsValueError("What should the primer be about?")
        }
        try CaptureInbox.write(c)
        CaptureRouter.arrivals.send(c.id)
        return .result()
    }
}

struct ChironShortcuts: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(
            intent: CapturePrimerIntent(),
            phrases: ["Make a \(.applicationName) primer", "Ask \(.applicationName) about this"],
            shortTitle: "Make a primer",
            systemImageName: "doc.text.badge.plus")
    }
}
