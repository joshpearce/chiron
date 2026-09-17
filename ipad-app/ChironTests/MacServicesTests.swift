#if targetEnvironment(macCatalyst)
import XCTest
@testable import Chiron

/// "Send to Chiron" in a Mac app's Services menu hands the selection over
/// on a pasteboard; the service keeps it as a capture in the inbox.
final class MacServicesTests: XCTestCase {
    private func pasteboard(holding text: String?) throws -> NSObject {
        let cls = try XCTUnwrap(NSClassFromString("NSPasteboard") as? NSObject.Type)
        let pb = try XCTUnwrap(cls.perform(NSSelectorFromString("pasteboardWithUniqueName"))?.takeUnretainedValue() as? NSObject)
        _ = pb.perform(NSSelectorFromString("clearContents"))
        if let text {
            _ = pb.perform(NSSelectorFromString("setString:forType:"), with: text, with: "public.utf8-plain-text")
        }
        return pb
    }

    func testTheSelectionBecomesACapture() throws {
        let id = try XCTUnwrap(MacServices.Provider().capture(from: pasteboard(holding: "  a passage from Chrome\n")))
        addTeardownBlock { _ = CaptureInbox.take(id) }
        let c = try XCTUnwrap(CaptureInbox.take(id))
        XCTAssertEqual(c.text, "a passage from Chrome")
        XCTAssertNil(c.pdfFile)
    }

    func testAnEmptySelectionIsNotACapture() throws {
        XCTAssertNil(try MacServices.Provider().capture(from: pasteboard(holding: "   ")))
        XCTAssertNil(try MacServices.Provider().capture(from: pasteboard(holding: nil)))
    }
}
#endif
