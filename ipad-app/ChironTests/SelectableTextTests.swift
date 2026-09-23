import XCTest
@testable import Chiron

/// The tutor's words have to be takeable in pieces. SwiftUI's own text
/// selection copies all or nothing, so these views are UIKit text views
/// underneath; what matters is that they stay selectable and stay
/// read-only, and that they carry the book's face rather than the system's.
@MainActor
final class SelectableTextTests: XCTestCase {
    func testTheTextIsSelectableButNotEditable() {
        let ui = SelectableText.textView()
        SelectableText(text: "What does your macaroon carry?").show(in: ui)

        XCTAssertEqual(ui.text, "What does your macaroon carry?")
        XCTAssertTrue(ui.isSelectable, "without this there are no handles to drag")
        XCTAssertFalse(ui.isEditable, "the tutor's words are not the reader's to edit")
        XCTAssertFalse(ui.isScrollEnabled, "it sizes to its content inside the card's own scroll")
    }

    func testItIsSetInTheBooksFace() {
        XCTAssertEqual(Typography.uiSerif(16).familyName, "Source Serif 4")
        XCTAssertEqual(Typography.uiSans(14).familyName, "Source Sans 3")
    }

}
