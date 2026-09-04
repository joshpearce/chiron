import XCTest
@testable import Chiron

/// The tools as the Pencil's gestures drive them.
@MainActor
final class ToolTests: XCTestCase {
    private func session() -> BookSession {
        let storage = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        return BookSession(subjectID: "ai", title: "How AI Works", service: FakeService(), storage: storage)
    }

    func testSqueezeCyclesTheTools() {
        let s = session()
        XCTAssertEqual(s.tool, .none)
        s.cycleTool(); XCTAssertEqual(s.tool, .pen)
        s.cycleTool(); XCTAssertEqual(s.tool, .highlighter)
        s.cycleTool(); XCTAssertEqual(s.tool, .ask)
        s.cycleTool(); XCTAssertEqual(s.tool, .none, "a book has no note tool")
        s.kind = "primer"
        s.tool = .ask
        s.cycleTool(); XCTAssertEqual(s.tool, .note)
    }

    func testDoubleTapFlipsTheEraser() {
        let s = session()
        s.tool = .pen
        s.flipEraser(); XCTAssertEqual(s.tool, .eraser)
        s.flipEraser(); XCTAssertEqual(s.tool, .pen)
        s.tool = .highlighter
        s.flipEraser(); XCTAssertEqual(s.tool, .eraser)
        s.flipEraser(); XCTAssertEqual(s.tool, .highlighter, "back to the tool that was up, not always the pen")
    }

    /// "Switch to the last used tool" in the Pencil settings.
    func testSwitchPreviousTradesTheLastTwoTools() {
        let s = session()
        s.switchPrevious(); XCTAssertEqual(s.tool, .pen, "nothing was up before: the pen is the first tool")
        s.tool = .highlighter
        s.switchPrevious(); XCTAssertEqual(s.tool, .pen)
        s.switchPrevious(); XCTAssertEqual(s.tool, .highlighter)
        s.tool = .none
        s.switchPrevious(); XCTAssertEqual(s.tool, .highlighter, "putting a tool down and switching back picks it up again")
    }
}
