import SwiftUI
import UIKit

/// Text the reader can take a piece of.
///
/// SwiftUI's `.textSelection(.enabled)` offers the whole string and nothing
/// less: long-press and the only choice is Copy, with no handles to drag in
/// from either end. A passage worth handing to someone else is usually a
/// passage, not everything that was said, so anywhere that holds the tutor's
/// words uses a UIKit text view instead - not editable, but selectable, which
/// is what brings the handles.
struct SelectableText: UIViewRepresentable {
    let text: String
    var font: UIFont = Typography.uiSerif(16)
    var colour: UIColor = .label

    func makeUIView(context: Context) -> UITextView { Self.textView() }

    /// The view itself, apart from SwiftUI, so its settings can be checked.
    static func textView() -> UITextView {
        let view = UITextView()
        view.isEditable = false
        view.isSelectable = true
        view.isScrollEnabled = false          // so it sizes to its content
        view.backgroundColor = .clear
        view.textContainerInset = .zero
        view.textContainer.lineFragmentPadding = 0
        view.adjustsFontForContentSizeCategory = true
        view.dataDetectorTypes = [.link]
        view.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        view.setContentHuggingPriority(.defaultHigh, for: .vertical)
        return view
    }

    func updateUIView(_ view: UITextView, context: Context) { show(in: view) }

    func show(in view: UITextView) {
        if view.text != text { view.text = text }
        view.font = font
        view.textColor = colour
    }
}

extension Typography {
    /// The same faces, as UIKit fonts, for the places that cannot take a
    /// SwiftUI one. Dynamic Type still applies: the metrics scale them.
    static func uiSerif(_ size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
        let name: String
        switch weight {
        case .bold, .heavy, .black: name = "SourceSerif4-Bold"
        case .semibold, .medium: name = "SourceSerif4-Semibold"
        default: name = "SourceSerif4-Regular"
        }
        let base = UIFont(name: name, size: size) ?? .systemFont(ofSize: size)
        return UIFontMetrics(forTextStyle: .body).scaledFont(for: base)
    }

    static func uiSans(_ size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
        let name: String
        switch weight {
        case .bold, .heavy, .black: name = "SourceSans3-Bold"
        case .semibold: name = "SourceSans3-Semibold"
        case .medium: name = "SourceSans3-Medium"
        default: name = "SourceSans3-Regular"
        }
        let base = UIFont(name: name, size: size) ?? .systemFont(ofSize: size)
        return UIFontMetrics(forTextStyle: .body).scaledFont(for: base)
    }
}
