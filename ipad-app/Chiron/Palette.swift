import SwiftUI

/// The reader's tools, on the page's trailing edge: pen and eraser for ink,
/// the highlighter for runs of text, and the ask tool, which highlights a
/// run and opens the card for a question. Tapping the active tool puts it
/// down. A Pencil Pro squeeze cycles the tools; a double-tap flips pen and
/// eraser.
struct Palette: View {
    @EnvironmentObject var session: BookSession

    private var tools: [(BookSession.Tool, String, String)] {
        var list: [(BookSession.Tool, String, String)] = [
            (.pen, "pencil.tip", "Pen"),
            (.highlighter, "highlighter", "Highlighter"),
            (.ask, "questionmark.bubble", "Ask about a passage"),
        ]
        if session.isPrimer {
            list.append((.note, "note.text.badge.plus", "Margin note: extend the primer"))
        }
        list.append((.eraser, "eraser", "Eraser"))
        return list
    }

    var body: some View {
        VStack(spacing: 6) {
            ForEach(tools, id: \.0) { tool, symbol, label in
                Button {
                    session.tool = session.tool == tool ? .none : tool
                } label: {
                    Image(systemName: symbol)
                        .font(.system(size: 18, weight: .medium))
                        .frame(width: 40, height: 40)
                        .foregroundStyle(session.tool == tool ? Color.white : Color.primary)
                        .background(
                            Circle().fill(session.tool == tool ? Color.accentColor : Color.clear))
                }
                .buttonStyle(.plain)
                .hoverEffect()
                .accessibilityLabel(label)
                .accessibilityAddTraits(session.tool == tool ? .isSelected : [])
            }
        }
        .padding(6)
        .glassEffect(.regular, in: Capsule())
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Tools")
    }
}
