import SwiftUI

/// The book's faces: Source Serif 4 for anything read, Source Sans 3 for
/// labels and controls, the same files the e-ink client typesets with, so
/// the two clients read as one book. System fonts stay for system chrome.
/// Every size is relative to a text style, so Dynamic Type applies.
enum Typography {
    static func serif(_ size: CGFloat, weight: Font.Weight = .regular, relativeTo style: Font.TextStyle = .body) -> Font {
        let name: String
        switch weight {
        case .bold, .heavy, .black: name = "SourceSerif4-Bold"
        case .semibold, .medium: name = "SourceSerif4-Semibold"
        default: name = "SourceSerif4-Regular"
        }
        return .custom(name, size: size, relativeTo: style)
    }

    static func serifItalic(_ size: CGFloat, relativeTo style: Font.TextStyle = .body) -> Font {
        .custom("SourceSerif4-It", size: size, relativeTo: style)
    }

    static func sans(_ size: CGFloat, weight: Font.Weight = .regular, relativeTo style: Font.TextStyle = .body) -> Font {
        let name: String
        switch weight {
        case .bold, .heavy, .black: name = "SourceSans3-Bold"
        case .semibold: name = "SourceSans3-Semibold"
        case .medium: name = "SourceSans3-Medium"
        default: name = "SourceSans3-Regular"
        }
        return .custom(name, size: size, relativeTo: style)
    }

    /// Display-size headings.
    static func display(_ size: CGFloat) -> Font { serif(size, weight: .semibold, relativeTo: .largeTitle) }

    /// The CSS font stack the web views use, matching the faces above.
    static let webSerif = "\"Source Serif 4\", -apple-system-ui-serif, ui-serif, Georgia, serif"
}
