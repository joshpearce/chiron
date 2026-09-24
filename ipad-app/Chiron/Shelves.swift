import SwiftUI

/// A shelf in the library: a folder card that opens the shelf, takes a
/// dragged book, and carries the shelf's menu.
struct ShelfFolderCard: View {
    @EnvironmentObject var library: Library
    let shelf: ShelfInfo
    @State private var over = false
    @State private var renaming = false
    @State private var deleting = false

    var body: some View {
        NavigationLink(value: shelf.id) {
            HStack(alignment: .center, spacing: 14) {
                Image(systemName: over ? "folder.fill" : "folder")
                    .font(.title2)
                    .foregroundStyle(over ? Color.accentColor : .secondary)
                VStack(alignment: .leading, spacing: 3) {
                    Text(shelf.name).font(Typography.serif(22, weight: .semibold, relativeTo: .title3))
                    Text(ShelfContentsView.countLine(shelf.subjects.count))
                        .font(Typography.sans(14, relativeTo: .caption)).foregroundStyle(.secondary)
                }
                Spacer()
                Image(systemName: "chevron.right").foregroundStyle(.secondary)
            }
            .padding(18)
            .frame(maxWidth: 480)
            .background(over ? AnyShapeStyle(Color.accentColor.opacity(0.15)) : AnyShapeStyle(.fill.tertiary),
                        in: .rect(cornerRadius: 22))
            .overlay(RoundedRectangle(cornerRadius: 22).stroke(Color.accentColor, lineWidth: over ? 2 : 0))
        }
        .buttonStyle(.plain)
        .hoverEffect()
        .accessibilityLabel("Shelf: \(shelf.name)")
        .accessibilityHint(ShelfContentsView.countLine(shelf.subjects.count))
        .dropDestination(for: String.self) { ids, _ in
            guard let id = ids.first else { return false }
            Task { await library.move(id, to: shelf.id) }
            return true
        } isTargeted: { over = $0 }
        .contextMenu { ShelfMenu(renaming: $renaming, deleting: $deleting) }
        .modifier(ShelfPrompts(shelf: shelf, renaming: $renaming, deleting: $deleting))
    }
}

/// The order of the cards, chosen once for the library and every shelf:
/// a picker for a toolbar menu.
struct ShelfOrderPicker: View {
    @EnvironmentObject var library: Library

    var body: some View {
        Picker("Sort by", selection: $library.order) {
            ForEach(ShelfOrder.allCases) { order in
                Text(order.label).tag(order)
            }
        }
        .pickerStyle(.inline)
    }
}

/// Rename or delete a shelf: the same two items on the card's menu and
/// on the shelf's screen. The prompts they raise live on the view that
/// stays on screen (`ShelfPrompts`); a dialog attached inside a menu is
/// gone with the menu before it can show.
struct ShelfMenu: View {
    @Binding var renaming: Bool
    @Binding var deleting: Bool

    var body: some View {
        Button {
            renaming = true
        } label: {
            Label("Rename", systemImage: "pencil")
        }
        Button(role: .destructive) {
            deleting = true
        } label: {
            Label("Delete shelf", systemImage: "trash")
        }
    }
}

/// The rename alert and the delete confirmation for one shelf. Deleting
/// returns what it held to the library.
struct ShelfPrompts: ViewModifier {
    @EnvironmentObject var library: Library
    let shelf: ShelfInfo
    @Binding var renaming: Bool
    @Binding var deleting: Bool
    @State private var name = ""

    func body(content: Content) -> some View {
        content
            .alert("Rename shelf", isPresented: $renaming) {
                TextField("Name", text: $name)
                Button("Save") { Task { await library.renameShelf(shelf.id, to: name) } }
                Button("Cancel", role: .cancel) {}
            }
            .onChange(of: renaming) { _, on in
                if on { name = shelf.name }
            }
            .confirmationDialog("Delete \"\(shelf.name)\"?", isPresented: $deleting, titleVisibility: .visible) {
                Button("Delete the shelf", role: .destructive) {
                    Task { await library.deleteShelf(shelf.id) }
                }
                Button("Keep it", role: .cancel) {}
            } message: {
                Text(shelf.subjects.isEmpty
                     ? "The shelf is empty."
                     : "What is on it goes back to the library; nothing is lost.")
            }
    }
}

/// One shelf's screen: its cards, a place to drag one back to the
/// library, and the shelf's menu in the bar.
struct ShelfContentsView: View {
    @EnvironmentObject var library: Library
    let shelfID: String
    @State private var over = false
    @State private var renaming = false
    @State private var deleting = false

    private var shelf: ShelfInfo? { library.shelf(shelfID) }
    private var contents: [SubjectInfo] { library.subjects(on: shelfID) }

    static func countLine(_ n: Int) -> String {
        switch n {
        case 0: return "Empty"
        case 1: return "1 item"
        default: return "\(n) items"
        }
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 12) {
                // The way out: a drop target that reads as the library.
                HStack(spacing: 12) {
                    Image(systemName: "books.vertical")
                        .foregroundStyle(over ? Color.accentColor : .secondary)
                    Text(over ? "Drop to move it back to the library" : "Drag a card here to move it back to the library")
                        .font(Typography.sans(14, relativeTo: .caption))
                        .foregroundStyle(.secondary)
                    Spacer()
                }
                .padding(16)
                .frame(maxWidth: 480)
                .background(over ? AnyShapeStyle(Color.accentColor.opacity(0.15)) : AnyShapeStyle(.fill.quaternary),
                            in: .rect(cornerRadius: 22))
                .overlay(RoundedRectangle(cornerRadius: 22)
                    .strokeBorder(style: StrokeStyle(lineWidth: over ? 2 : 1, dash: over ? [] : [6, 4]))
                    .foregroundStyle(over ? Color.accentColor : .secondary.opacity(0.4)))
                .dropDestination(for: String.self) { ids, _ in
                    guard let id = ids.first else { return false }
                    Task { await library.move(id, to: nil) }
                    return true
                } isTargeted: { over = $0 }
                .accessibilityElement(children: .ignore)
                .accessibilityLabel("Back to the library")
                .accessibilityHint("Drop a card here to take it off the shelf")

                ForEach(contents) { s in
                    ShelfCard(subject: s)
                }
                if contents.isEmpty {
                    Text("Nothing on this shelf yet. Drag a card onto it from the library, or use Move to on a card.")
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.top, 24)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 24)
            .frame(maxWidth: .infinity)
        }
        .navigationTitle(shelf?.name ?? "Shelf")
        .toolbarTitleDisplayMode(.inline)
        .toolbar {
            if shelf != nil {
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        ShelfOrderPicker()
                        Divider()
                        ShelfMenu(renaming: $renaming, deleting: $deleting)
                    } label: {
                        Label("Shelf", systemImage: "ellipsis.circle")
                    }
                    .accessibilityLabel("Shelf menu")
                }
            }
        }
        .modifier(ShelfPrompts(shelf: shelf ?? ShelfInfo(id: shelfID, name: ""), renaming: $renaming, deleting: $deleting))
    }
}
