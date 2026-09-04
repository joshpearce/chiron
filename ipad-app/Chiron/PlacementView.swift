import SwiftUI

/// The placement screener: one question, five levels, one tap. There is no
/// "I don't know" here by design - not knowing where you stand is the lowest
/// level, and that is a row.
struct PlacementView: View {
    @EnvironmentObject var session: BookSession
    let screener: CheckItem

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                Text("Before the book begins")
                    .font(Typography.display(30))
                MathText(text: screener.prompt, size: 20)
                VStack(spacing: 10) {
                    ForEach(Array((screener.options ?? []).enumerated()), id: \.offset) { i, option in
                        Button {
                            Task { await session.place(level: i + 1) }
                        } label: {
                            HStack(alignment: .firstTextBaseline, spacing: 14) {
                                Text("\(i + 1)")
                                    .font(Typography.serif(20, weight: .semibold, relativeTo: .title3))
                                    .foregroundStyle(.secondary)
                                    .frame(width: 28, alignment: .trailing)
                                Text(option.text)
                                    .font(Typography.serif(19))
                                    .multilineTextAlignment(.leading)
                                Spacer(minLength: 0)
                            }
                            .padding(.vertical, 14).padding(.horizontal, 16)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(.fill.tertiary, in: .rect(cornerRadius: 16))
                        }
                        .buttonStyle(.plain)
                        .hoverEffect()
                        .disabled(session.busy)
                        .accessibilityLabel("Level \(i + 1): \(option.text)")
                        .keyboardShortcut(KeyEquivalent(Character("\(i + 1)")), modifiers: [])
                    }
                }
                Text("Pick the row that fits. The questions that follow size you within it; nothing here is a grade.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }
            .padding(32)
            .frame(maxWidth: 680)
            .frame(maxWidth: .infinity)
        }
    }
}
