import SwiftUI

/// The library's sidebar: what the reader is in the middle of, then
/// everything by kind, then the shelves. A row dragged onto a shelf is
/// filed there; dragged onto All, it comes off its shelf.
struct LibrarySidebar: View {
    @EnvironmentObject var library: Library
    let newShelf: () -> Void

    var body: some View {
        List(selection: $library.scope) {
            if !library.continuing.isEmpty {
                Section("Continue") {
                    ForEach(library.continuing) { s in
                        ContinueRow(subject: s)
                    }
                }
            }
            Section("Library") {
                ScopeRow(scope: .all, title: "All", symbol: "books.vertical", tint: .secondary,
                         count: library.subjects.count, dropsTo: .some(nil))
                ForEach(LibraryKind.allCases, id: \.self) { kind in
                    let n = library.listed(in: .kind(kind)).count
                    if n > 0 {
                        ScopeRow(scope: .kind(kind), title: kind.plural, symbol: kind.symbol, tint: kind.tint,
                                 count: kind == .feed ? library.unreadTotal : n)
                    }
                }
            }
            Section("Shelves") {
                ForEach(library.shelves) { shelf in
                    ShelfRow(shelf: shelf)
                }
                Button(action: newShelf) {
                    Label("New shelf", systemImage: "plus")
                }
                .disabled(!library.sync.connected)
            }
        }
        .navigationTitle("Library")
        .overlay {
            if library.subjects.isEmpty && !library.loadingShelf {
                Text(library.shelfError ?? "Nothing in the library yet.")
                    .foregroundStyle(.secondary)
                    .padding()
            }
        }
    }
}

/// One of the sidebar's picks. A row that takes drops files what lands
/// on it: on a shelf, or (with nil) off whichever shelf it was on.
private struct ScopeRow: View {
    @EnvironmentObject var library: Library
    let scope: LibraryScope
    let title: String
    let symbol: String
    let tint: Color
    let count: Int
    var dropsTo: String?? = .none
    @State private var over = false

    var body: some View {
        NavigationLink(value: scope) {
            Label {
                Text(title)
            } icon: {
                Image(systemName: symbol).foregroundStyle(tint)
            }
        }
        .badge(count)
        .listRowBackground(over ? Color.accentColor.opacity(0.18) : nil)
        .modifier(DropFiling(to: dropsTo, over: $over))
        .accessibilityIdentifier(scope == .all ? "library-all" : "library-\(scope.harness)")
        .accessibilityHint(dropsTo != nil ? "Drop a row here to take it off its shelf" : "")
    }
}

/// A shelf in the sidebar: picks it, takes a dragged row, and carries the
/// shelf's menu.
private struct ShelfRow: View {
    @EnvironmentObject var library: Library
    let shelf: ShelfInfo
    @State private var over = false
    @State private var renaming = false
    @State private var deleting = false

    var body: some View {
        NavigationLink(value: LibraryScope.shelf(shelf.id)) {
            Label(shelf.name, systemImage: over ? "folder.fill" : "folder")
        }
        .badge(shelf.subjects.count)
        .listRowBackground(over ? Color.accentColor.opacity(0.18) : nil)
        .accessibilityLabel("Shelf: \(shelf.name)")
        .modifier(DropFiling(to: .some(shelf.id), over: $over))
        .contextMenu { ShelfMenu(renaming: $renaming, deleting: $deleting) }
        .modifier(ShelfPrompts(shelf: shelf, renaming: $renaming, deleting: $deleting))
    }
}

/// Where a dragged row lands. `to` is the shelf it goes on; `.some(nil)`
/// takes it off its shelf; `.none` takes no drops.
private struct DropFiling: ViewModifier {
    @EnvironmentObject var library: Library
    let to: String??
    @Binding var over: Bool

    func body(content: Content) -> some View {
        if let shelf = to {
            content.dropDestination(for: String.self) { ids, _ in
                guard let id = ids.first else { return false }
                Task { await library.move(id, to: shelf) }
                return true
            } isTargeted: { over = $0 }
        } else {
            content
        }
    }
}

/// Something the reader is in the middle of, opened from the sidebar.
private struct ContinueRow: View {
    @EnvironmentObject var library: Library
    let subject: SubjectInfo

    var body: some View {
        Button {
            Task { await library.open(subject.id) }
        } label: {
            HStack(spacing: 8) {
                KindIcon(kind: subject.libraryKind, size: 20)
                Text(subject.title).lineLimit(1)
                Spacer(minLength: 4)
                if subject.authoring {
                    ProgressView().controlSize(.mini)
                } else if subject.id == library.activeSubjectID {
                    Circle().fill(Color.accentColor).frame(width: 7, height: 7)
                        .accessibilityLabel("Open now")
                }
            }
        }
        .disabled(subject.authoring)
        .accessibilityLabel("Continue: \(subject.title)")
    }
}

/// The list the sidebar picked, in the order and density chosen from its
/// view menu.
struct LibraryList: View {
    @EnvironmentObject var library: Library
    let scope: LibraryScope
    @AppStorage("library.dense") private var dense = false
    @State private var renaming = false
    @State private var deleting = false

    private var shelf: ShelfInfo? {
        if case .shelf(let id) = scope { return library.shelf(id) }
        return nil
    }

    private var title: String {
        switch scope {
        case .all: return "All"
        case .kind(let k): return k.plural
        case .shelf: return shelf?.name ?? "Shelf"
        }
    }

    var body: some View {
        List {
            if let build = library.availableBuild, let url = library.installURL {
                BuildBanner(build: build, url: url)
                    .listRowSeparator(.hidden)
            }
            ForEach(library.listed(in: scope)) { s in
                LibraryRow(subject: s, showShelf: shelf == nil, dense: dense)
                    .listRowInsets(EdgeInsets(top: dense ? 3 : 6, leading: 16, bottom: dense ? 3 : 6, trailing: 16))
            }
            if let err = library.shelfError, !library.subjects.isEmpty {
                Text(err).foregroundStyle(.red).font(.callout)
            }
        }
        .listStyle(.plain)
        .environment(\.defaultMinListRowHeight, 36)
        .overlay {
            if library.listed(in: scope).isEmpty && !library.subjects.isEmpty {
                Text(shelf != nil
                     ? "Nothing on this shelf yet. Drag a row onto it in the sidebar, or use Move to on a row."
                     : "Nothing here yet.")
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .padding(32)
            }
        }
        .refreshable { await library.refresh() }
        .navigationTitle(title)
        .toolbarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Menu {
                    Picker("Show as", selection: $dense) {
                        Text("Rows").tag(false)
                        Text("Dense rows").tag(true)
                    }
                    Picker("Sort by", selection: $library.order) {
                        ForEach(LibrarySort.allCases, id: \.self) { Text($0.label).tag($0) }
                    }
                } label: {
                    Label("View", systemImage: "line.3.horizontal.decrease.circle")
                }
            }
            if shelf != nil {
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        ShelfMenu(renaming: $renaming, deleting: $deleting)
                    } label: {
                        Label("Shelf", systemImage: "ellipsis.circle")
                    }
                    .accessibilityLabel("Shelf menu")
                }
            }
        }
        .modifier(ShelfPrompts(shelf: shelf ?? ShelfInfo(id: "", name: ""), renaming: $renaming, deleting: $deleting))
    }
}

/// A kind's mark: its symbol, white on its colour.
struct KindIcon: View {
    let kind: LibraryKind
    var size: CGFloat = 28

    var body: some View {
        Image(systemName: kind.symbol)
            .font(.system(size: size * 0.5, weight: .semibold))
            .foregroundStyle(.white)
            .frame(width: size, height: size)
            .background(kind.tint, in: .rect(cornerRadius: size * 0.26))
            .accessibilityHidden(true)
    }
}

/// One book, primer, blog, reading or PDF in the list. A book carries its
/// progress; a primer its source and date, with a spinner while the
/// server writes it and in red when that failed; a blog what is unread.
struct LibraryRow: View {
    @EnvironmentObject var library: Library
    let subject: SubjectInfo
    var showShelf = true
    var dense = false

    private var openable: Bool { subject.drafting || (!subject.authoring && !subject.failed) }
    private var isOpen: Bool { subject.id == library.activeSubjectID }

    private var statusLine: String {
        if subject.authoring {
            return subject.building ? subject.progressLine : "Writing the primer · \(subject.sourceLine)"
        }
        if subject.failed && !subject.drafting { return subject.error ?? "The primer could not be written." }
        return subject.progressLine
    }

    private var kindName: String {
        if subject.drafting || subject.building { return "Draft" }
        switch subject.libraryKind {
        case .pdf: return "PDF"
        case .feed: return "Followed blog"
        case .reading: return "Imported book"
        case .primer: return "Primer"
        case .book: return "Smart book"
        }
    }

    var body: some View {
        Button {
            Task {
                if subject.drafting { await library.openDraft(subject.id) } else { await library.open(subject.id) }
            }
        } label: {
            HStack(spacing: 12) {
                KindIcon(kind: subject.libraryKind, size: dense ? 22 : 30)
                VStack(alignment: .leading, spacing: 2) {
                    Text(subject.title)
                        .font(Typography.serif(dense ? 16 : 18, weight: .semibold, relativeTo: .body))
                        .lineLimit(1)
                    if !dense {
                        Text(statusLine)
                            .font(Typography.sans(13, relativeTo: .caption))
                            .foregroundStyle(subject.failed && !subject.drafting ? .red : .secondary)
                            .lineLimit(1)
                    }
                }
                Spacer(minLength: 8)
                if showShelf, let id = subject.shelf, let name = library.shelf(id)?.name {
                    Text(name)
                        .font(Typography.sans(12, relativeTo: .caption2))
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }
                trailing
            }
            .padding(.vertical, dense ? 0 : 2)
            .contentShape(Rectangle())
            .opacity(openable ? 1 : 0.55)
        }
        .buttonStyle(.plain)
        .hoverEffect()
        .disabled(!openable)
        .accessibilityLabel(subject.title)
        .accessibilityValue(kindName)
        .accessibilityHint(subject.authoring ? "Still being written" : (isOpen ? "Open now" : statusLine))
        // Filed by dragging onto a shelf in the sidebar, or by the menu for
        // a hand that would rather not drag.
        .draggable(subject.id)
        .contextMenu {
            Menu("Move to") {
                Button {
                    Task { await library.move(subject.id, to: nil) }
                } label: {
                    Label("Library", systemImage: "books.vertical")
                }
                .disabled(subject.shelf == nil)
                ForEach(library.shelves) { shelf in
                    Button {
                        Task { await library.move(subject.id, to: shelf.id) }
                    } label: {
                        Label(shelf.name, systemImage: "folder")
                    }
                    .disabled(subject.shelf == shelf.id)
                }
            }
            if subject.isFeed || subject.isReading || subject.isPrimer {
                Button(role: .destructive) {
                    Task { await library.forget(subject.id) }
                } label: {
                    Label(subject.isFeed ? "Unfollow" : "Take off the shelf", systemImage: "trash")
                }
                .disabled(subject.authoring || subject.building)
            }
        }
    }

    /// At the row's end, what most wants seeing: work in progress, what is
    /// unread, the one open, and in dense rows the progress the second
    /// line would have carried.
    @ViewBuilder private var trailing: some View {
        HStack(spacing: 6) {
            if subject.drafting || subject.building { Text("🔨").font(.footnote) }
            if subject.authoring {
                ProgressView().controlSize(.small)
            } else if subject.isFeed, let n = subject.unread, n > 0 {
                Text("\(n)")
                    .font(Typography.sans(12, weight: .semibold, relativeTo: .caption))
                    .monospacedDigit()
                    .foregroundStyle(.white)
                    .padding(.horizontal, 7)
                    .padding(.vertical, 2)
                    .background(LibraryKind.feed.tint, in: .capsule)
                    .accessibilityLabel("\(n) unread")
            } else if dense {
                Text(statusLine)
                    .font(Typography.sans(12, relativeTo: .caption))
                    .foregroundStyle(subject.failed && !subject.drafting ? .red : .secondary)
                    .lineLimit(1)
            }
            if isOpen {
                Circle().fill(Color.accentColor).frame(width: 8, height: 8)
                    .accessibilityLabel("Open now")
            }
        }
    }
}
