import PencilKit
import UIKit

/// Ink and the system's appearance.
///
/// PencilKit stores the colour a stroke was drawn with, not a colour that
/// follows the page. A chapter written on in black in the afternoon is
/// invisible when evening turns the page black, which is what the reader
/// sees as their handwriting disappearing.
///
/// So ink is kept as it looks on a light page, and converted to whatever
/// the page is now on the way to the canvas and back. A PDF's page is
/// white whatever the system is doing, so its ink is always the light one.
enum Ink {
    /// The same colour, meant for a different appearance: black on white
    /// becomes white on black.
    static func color(_ color: UIColor, from: UIUserInterfaceStyle, to: UIUserInterfaceStyle) -> UIColor {
        guard from != to else { return color }
        return PKInkingTool.convertColor(color, from: from, to: to)
    }

    /// A stored drawing as it should look on a page in this style.
    static func shown(_ drawing: PKDrawing, in style: UIUserInterfaceStyle) -> PKDrawing {
        mapped(drawing) { color(  $0, from: .light, to: style) }
    }

    /// A drawing off a page in this style, as it is kept.
    static func canonical(_ drawing: PKDrawing, drawnIn style: UIUserInterfaceStyle) -> PKDrawing {
        mapped(drawing) { color($0, from: style, to: .light) }
    }

    /// A palette colour as the pen should lay it down on a page in this
    /// style; what it leaves is converted back when it is kept.
    static func pen(_ chosen: UIColor, on style: UIUserInterfaceStyle) -> UIColor {
        color(resolved(chosen, .light), from: .light, to: style)
    }

    /// Ink for paper: a PDF page is white in either appearance.
    static var onPaper: UIColor { resolved(.label, .light) }

    private static func resolved(_ color: UIColor, _ style: UIUserInterfaceStyle) -> UIColor {
        color.resolvedColor(with: UITraitCollection(userInterfaceStyle: style))
    }

    private static func mapped(_ drawing: PKDrawing, _ transform: (UIColor) -> UIColor) -> PKDrawing {
        PKDrawing(strokes: drawing.strokes.map { stroke in
            var out = stroke
            out.ink = PKInk(stroke.ink.inkType, color: transform(stroke.ink.color))
            return out
        })
    }
}
