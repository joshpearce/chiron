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

    /// A UITextView hands SwiftUI a size of its own choosing, and left to
    /// itself it says one line however much text it holds - which silently
    /// cuts the tutor off mid-sentence. It has to measure the text against
    /// the width it is given.
    func testAPassageIsAsTallAsItNeedsToBe() {
        let width: CGFloat = 320
        let sentence = "A macaroon carries a bearer token plus a chain of caveats, each one narrowing what the holder may do: which service, which method, which resource, until when."

        let one = SelectableText(text: "One line.").height(forWidth: width)
        let many = SelectableText(text: sentence).height(forWidth: width)
        let more = SelectableText(text: sentence + " " + sentence).height(forWidth: width)

        XCTAssertGreaterThan(one, 0)
        XCTAssertGreaterThan(many, one * 2, "a passage that wraps is taller than a single line")
        XCTAssertGreaterThan(more, many, "twice the words, more of the height")
    }

    /// Narrower means taller: the measurement is of this text at this width,
    /// not a number it remembered from somewhere else.
    func testANarrowerColumnIsTaller() {
        let text = SelectableText(text: String(repeating: "caveats narrow what the holder may do. ", count: 6))
        XCTAssertGreaterThan(text.height(forWidth: 200), text.height(forWidth: 600))
    }

    func testItIsSetInTheBooksFace() {
        XCTAssertEqual(Typography.uiSerif(16).familyName, "Source Serif 4")
        XCTAssertEqual(Typography.uiSans(14).familyName, "Source Sans 3")
    }

}
