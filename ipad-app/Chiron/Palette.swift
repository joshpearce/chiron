import SwiftUI

/// The reader's tools, on the page's trailing edge: pen and eraser for ink,
/// the highlighter for runs of text, and the ask tool, which highlights a
/// run and opens the card for a question. Tapping the active tool puts it
/// down. The Pencil Pro's squeeze and double-tap do what Settings says
/// they do, with "show the palette" read as "next tool".
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
                // The pen's colours, right under the pen while it is up.
                if tool == .pen && session.tool == .pen {
                    ForEach(BookSession.PenColor.allCases, id: \.self) { c in
                        Button {
                            session.penColor = c
                        } label: {
                            Circle()
                                .fill(Color(c.uiColor))
                                .frame(width: 18, height: 18)
                                .overlay(Circle().stroke(Color.primary.opacity(session.penColor == c ? 0.9 : 0), lineWidth: 2))
                                .frame(width: 40, height: 26)
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel("\(c.rawValue) pen")
                        .accessibilityAddTraits(session.penColor == c ? .isSelected : [])
                    }
                }
            }
        }
        .padding(6)
        // Material, not glass: glassEffect, plain or interactive, swallowed
        // every tap on the palette's buttons (the UI test proves it).
        .background(.thinMaterial, in: Capsule())
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Tools")
    }
}
