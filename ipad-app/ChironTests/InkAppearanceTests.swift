import PencilKit
import XCTest
@testable import Chiron

/// PencilKit keeps the colour a stroke was drawn in, and the page follows
/// the system's appearance: ink written in black on a white page is gone
/// when evening turns the page black. Ink is kept as it looks in light and
/// converted for whatever the page is now.
final class InkAppearanceTests: XCTestCase {
    private func stroke(_ color: UIColor) -> PKDrawing {
        let path = PKStrokePath(controlPoints: [0, 40].map {
            PKStrokePoint(location: CGPoint(x: $0, y: 10), timeOffset: 0, size: CGSize(width: 3, height: 3),
                          opacity: 1, force: 1, azimuth: 0, altitude: .pi / 2)
        }, creationDate: Date())
        return PKDrawing(strokes: [PKStroke(ink: PKInk(.pen, color: color), path: path)])
    }

    private func brightness(_ color: UIColor) -> CGFloat {
        var white: CGFloat = 0, alpha: CGFloat = 0
        color.getWhite(&white, alpha: &alpha)
        return white
    }

    func testDarkShowsLightInkAndLightShowsItBack() {
        let black = UIColor.black
        let inDark = Ink.color(black, from: .light, to: .dark)
        XCTAssertGreaterThan(brightness(inDark), 0.5, "black ink shows light on a dark page")
        let back = Ink.color(inDark, from: .dark, to: .light)
        XCTAssertLessThan(brightness(back), 0.5, "and dark again on a light one")
    }

    func testADrawingIsShownForThePageAndKeptAsItLooksInLight() {
        let drawing = stroke(.black)
        let shown = Ink.shown(drawing, in: .dark)
        XCTAssertEqual(shown.strokes.count, 1)
        XCTAssertGreaterThan(brightness(shown.strokes[0].ink.color), 0.5, "the stroke is visible on the dark page")

        let kept = Ink.canonical(shown, drawnIn: .dark)
        XCTAssertLessThan(brightness(kept.strokes[0].ink.color), 0.5, "what is stored is how it looks in light")
        // (compared by brightness: PencilKit keeps the colour in its own
        // space, so the same black is not the same object.)
        XCTAssertEqual(brightness(Ink.shown(drawing, in: .light).strokes[0].ink.color), 0, accuracy: 0.01,
                       "a light page shows what was stored")
    }

    /// A PDF page is white whatever the system is doing, so its ink never
    /// turns with the appearance.
    func testPaperInkStaysDark() {
        XCTAssertLessThan(brightness(Ink.onPaper), 0.5)
    }
}
