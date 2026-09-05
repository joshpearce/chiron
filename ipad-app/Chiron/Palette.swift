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
        list.append((.capture, "square.and.arrow.up", "Send to Chiron: a passage into a capture"))
        list.append((.eraser, "eraser", "Eraser"))
        return list
    }

    var body: some View {
        // One glass control per tool, sharing a container so the colours
        // grow out of the pen rather than appearing beside it.
        GlassEffectContainer(spacing: 8) {
            VStack(alignment: .trailing, spacing: 8) {
                ForEach(tools, id: \.0) { tool, symbol, label in
                    HStack(spacing: 8) {
                        // The pen's colours beside the pen while it is up,
                        // so the tools below keep their places.
                        if tool == .pen && session.tool == .pen {
                            ForEach(BookSession.PenColor.allCases, id: \.self) { c in
                                Button {
                                    session.penColor = c
                                } label: {
                                    Circle()
                                        .fill(Color(c.uiColor))
                                        .frame(width: 16, height: 16)
                                        .overlay(Circle().stroke(Color.primary.opacity(session.penColor == c ? 0.9 : 0), lineWidth: 2))
                                        .frame(width: 22, height: 22)
                                }
                                .buttonStyle(.glass)
                                .glassEffectID("colour-\(c.rawValue)", in: palette)
                                .accessibilityLabel("\(c.rawValue) pen")
                                .accessibilityAddTraits(session.penColor == c ? .isSelected : [])
                            }
                        }
                        toolButton(tool, symbol, label)
                    }
                }
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Tools")
    }

    @Namespace private var palette

    @ViewBuilder
    private func toolButton(_ tool: BookSession.Tool, _ symbol: String, _ label: String) -> some View {
        let button = Button {
            session.tool = session.tool == tool ? .none : tool
        } label: {
            Image(systemName: symbol)
                .font(.system(size: 18, weight: .medium))
                .frame(width: 26, height: 26)
        }
        .glassEffectID(tool.rawValue, in: palette)
        .accessibilityLabel(label)
        .accessibilityAddTraits(session.tool == tool ? .isSelected : [])
        if session.tool == tool {
            button.buttonStyle(.glassProminent)
        } else {
            button.buttonStyle(.glass)
        }
    }
}
