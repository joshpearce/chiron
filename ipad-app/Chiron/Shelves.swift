import SwiftUI

/// What a thing in the library is, in the order the sidebar lists them.
enum LibraryKind: String, CaseIterable, Hashable {
    case book, primer, feed, reading, pdf

    var plural: String {
        switch self {
        case .book: return "Books"
        case .primer: return "Primers"
        case .feed: return "Feeds"
        case .reading: return "Readings"
        case .pdf: return "PDFs"
        }
    }

    var symbol: String {
        switch self {
        case .book: return "brain"
        case .primer: return "doc.text"
        case .feed: return "dot.radiowaves.up.forward"
        case .reading: return "book"
        case .pdf: return "doc.richtext"
        }
    }

    var tint: Color {
        switch self {
        case .book: return Color(red: 0.21, green: 0.34, blue: 0.56)
        case .primer: return Color(red: 0.25, green: 0.49, blue: 0.36)
        case .feed: return Color(red: 0.77, green: 0.42, blue: 0.14)
        case .reading: return Color(red: 0.48, green: 0.29, blue: 0.53)
        case .pdf: return Color(red: 0.55, green: 0.29, blue: 0.27)
        }
    }
}

/// What the library's list shows, picked in the sidebar.
enum LibraryScope: Hashable {
    case all
    case kind(LibraryKind)
    case shelf(String)

    /// The harness's spelling: "all", a kind's name, or "shelf:<id>".
    init?(harness raw: String) {
        if raw == "all" { self = .all; return }
        if raw.hasPrefix("shelf:") { self = .shelf(String(raw.dropFirst(6))); return }
        guard let k = LibraryKind(rawValue: raw) else { return nil }
        self = .kind(k)
    }

    var harness: String {
        switch self {
        case .all: return "all"
        case .kind(let k): return k.rawValue
        case .shelf(let id): return "shelf:\(id)"
        }
    }
}

/// The list's order, in every scope: the reader's choice, kept on the
/// device.
enum LibrarySort: String, CaseIterable {
    case recent, title, kind, unread

    var label: String {
        switch self {
        case .recent: return "Most recent first"
        case .title: return "Title"
        case .kind: return "Kind"
        case .unread: return "Unread"
        }
    }

    /// Recent puts what last changed or was opened first, and what the
    /// server sent no stamp for after it, by title. Kind and unread keep
    /// the server's order among rows that tie.
    func apply(_ rows: [SubjectInfo]) -> [SubjectInfo] {
        func byTitle(_ a: SubjectInfo, _ b: SubjectInfo) -> Bool {
            a.title.localizedCaseInsensitiveCompare(b.title) == .orderedAscending
        }
        switch self {
        case .title:
            return rows.sorted(by: byTitle)
        case .recent:
            return rows.sorted { a, b in
                switch (a.updated, b.updated) {
                case let (x?, y?) where x != y: return x > y
                case (.some, .none): return true
                case (.none, .some): return false
                default: return byTitle(a, b)
                }
            }
        case .kind, .unread:
            func key(_ s: SubjectInfo) -> Int {
                self == .kind ? LibraryKind.allCases.firstIndex(of: s.libraryKind) ?? 0 : -(s.isFeed ? s.unread ?? 0 : 0)
            }
            return rows.enumerated().sorted { a, b in
                key(a.element) != key(b.element) ? key(a.element) < key(b.element) : a.offset < b.offset
            }.map(\.element)
        }
    }
}

extension SubjectInfo {
    var libraryKind: LibraryKind {
        if isPDF { return .pdf }
        if isFeed { return .feed }
        if isReading { return .reading }
        if isPrimer && scale != "book" { return .primer }
        return .book
    }

    /// Started and not finished, or still being written.
    var underway: Bool {
        if authoring || building { return true }
        if isPDF { return (page ?? 0) > 0 }
        if isFeed || isPrimer { return false }
        guard let c = unitsCleared, let t = unitsTotal else { return false }
        return c > 0 && c < t
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
