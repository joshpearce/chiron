import SwiftUI

/// The reader's tools, on the page's trailing edge: pen and eraser for ink,
/// the highlighter for runs of text, and the ask tool, which highlights a
/// run and opens the card for a question. Tapping the active tool puts it
/// down. The Pencil Pro's squeeze and double-tap do what Settings says
/// they do, with "show the palette" read as "next tool".
struct Palette: View {
    @EnvironmentObject var session: BookSession
    @Environment(\.horizontalSizeClass) private var sizeClass

    /// Regular width stands the tools down the trailing edge, in the
    /// page's margin; compact width (the phone, Split View) lays them
    /// along the bottom edge, where there is no margin to spare.
    private var compact: Bool { sizeClass == .compact }

    /// The pen's colours fold into one dot of the current colour beside
    /// the pen; a tap on the dot opens the row, and a pick closes it.
    @State private var choosingColour = false

    /// Ink needs a Pencil, so a phone's palette has no pen and no eraser.
    static var inkable: Bool { UIDevice.current.userInterfaceIdiom == .pad }

    static func tools(primer: Bool, inkable: Bool) -> [(BookSession.Tool, String, String)] {
        var list: [(BookSession.Tool, String, String)] = []
        if inkable { list.append((.pen, "pencil.tip", "Pen")) }
        list.append((.highlighter, "highlighter", "Highlighter"))
        list.append((.ask, "questionmark.bubble", "Ask about a passage"))
        if primer {
            list.append((.note, "note.text.badge.plus", "Margin note: extend the primer"))
        }
        list.append((.capture, "square.and.arrow.up", "Send to Chiron: a passage into a capture"))
        if inkable { list.append((.eraser, "eraser", "Eraser")) }
        return list
    }

    private var tools: [(BookSession.Tool, String, String)] {
        Self.tools(primer: session.isPrimer, inkable: Self.inkable)
    }

    var body: some View {
        // One glass control per tool, sharing a container so the colours
        // grow out of the pen rather than appearing beside it.
        let stack = compact
            ? AnyLayout(HStackLayout(spacing: 8))
            : AnyLayout(VStackLayout(alignment: .trailing, spacing: 8))
        let beside = compact
            ? AnyLayout(VStackLayout(spacing: 8))
            : AnyLayout(HStackLayout(spacing: 8))
        GlassEffectContainer(spacing: 8) {
            stack {
                ForEach(tools, id: \.0) { tool, symbol, label in
                    beside {
                        // The pen's colour beside the pen while it is up,
                        // so the tools below keep their places.
                        if tool == .pen && session.tool == .pen {
                            if choosingColour {
                                ForEach(BookSession.PenColor.allCases, id: \.self) { c in
                                    Button {
                                        session.penColor = c
                                        choosingColour = false
                                    } label: {
                                        dot(c, ringed: session.penColor == c)
                                    }
                                    .buttonStyle(.glass)
                                    .glassEffectID("colour-\(c.rawValue)", in: palette)
                                    .accessibilityLabel("\(c.rawValue) pen")
                                    .accessibilityAddTraits(session.penColor == c ? .isSelected : [])
                                }
                            } else {
                                Button {
                                    choosingColour = true
                                } label: {
                                    dot(session.penColor, ringed: false)
                                }
                                .buttonStyle(.glass)
                                .glassEffectID("colour-\(session.penColor.rawValue)", in: palette)
                                .accessibilityLabel("Pen colour")
                                .accessibilityValue(session.penColor.rawValue)
                            }
                        }
                        toolButton(tool, symbol, label)
                    }
                }
            }
        }
        .onChange(of: session.tool) { choosingColour = false }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Tools")
    }

    private func dot(_ c: BookSession.PenColor, ringed: Bool) -> some View {
        Circle()
            .fill(Color(c.uiColor))
            .frame(width: 16, height: 16)
            .overlay(Circle().stroke(Color.primary.opacity(ringed ? 0.9 : 0), lineWidth: 2))
            .frame(width: 22, height: 22)
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
