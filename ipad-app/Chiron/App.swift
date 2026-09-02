import SwiftUI

@main
struct ChironApp: App {
    @StateObject private var library = Library()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(library)
                .task {
                    if let server = SelfTest.serverOverride {
                        library.sync.baseURL = server
                        await library.sync.probe()
                    }
                    #if DEBUG
                    if Harness.requested {
                        Harness.shared.start(library)
                    }
                    #endif
                    if SelfTest.teachRequested {
                        library.teaching = true
                    } else if SelfTest.requested {
                        await SelfTest.run(library)
                    } else {
                        await library.openActiveAtLaunch()
                    }
                }
                .onChange(of: scenePhase) { phase in
                    if phase == .inactive || phase == .background {
                        library.session?.persist()
                    }
                }
        }
    }
}

struct ContentView: View {
    @EnvironmentObject var library: Library

    var body: some View {
        Group {
            if library.teaching {
                TeachView(sync: library.sync, demo: SelfTest.teachDemo)
            } else if let session = library.session {
                BookView()
                    .environmentObject(session)
                    .id(session.subjectID)
            } else {
                BookshelfView()
            }
        }
        // Fill the screen so the top-right chrome pins to the display corner.
        // Without this the stack shrinks to its content and the badge drifts
        // into the middle of the page on short screens.
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

/// One open book: whichever screen its session is on, with the chrome that
/// leads back to the contents and the shelf.
struct BookView: View {
    @EnvironmentObject var library: Library
    @EnvironmentObject var session: BookSession
    @Environment(\.horizontalSizeClass) private var sizeClass

    /// Regular width shows the contents as a sidebar beside the page;
    /// compact width (Split View, Slide Over) presents it over the page.
    private var sidebar: Bool { sizeClass == .regular }
    private var chromeHidden: Bool {
        if case .reading = session.screen { return session.chromeHidden }
        return false
    }

    var body: some View {
        HStack(spacing: 0) {
            if sidebar && session.contentsShown {
                ContentsList()
                    .frame(width: 320)
                    .transition(.move(edge: .leading))
                Divider()
            }
            page
        }
        .animation(.easeInOut(duration: 0.2), value: session.contentsShown)
        .sheet(isPresented: Binding(
            get: { !sidebar && session.contentsShown },
            set: { session.contentsShown = $0 })) {
            ContentsView()
        }
    }

    private var page: some View {
        ZStack {
            switch session.screen {
            case .empty:
                Color.clear
            case .placement:
                if let screener = session.chapter?.screener {
                    PlacementView(screener: screener)
                }
            case .series:
                if let ch = session.chapter {
                    ItemFlowView(
                        title: ch.title,
                        subtitle: "A measurement, not a test. Answer what you can; \"I don't know\" is an answer.",
                        items: ch.check,
                        reveal: false,
                        submitLabel: "Finish"
                    ) { responses in
                        await session.submitCheck(responses)
                    }
                }
            case .reading:
                if let chapter = session.chapter {
                    ReaderContainer(chapter: chapter)
                }
            case .pretest:
                if let ch = session.chapter {
                    ItemFlowView(
                        title: "Before you read",
                        subtitle: "You are not supposed to know these yet - answering wrong here is part of how the chapter calibrates.",
                        items: ch.pretest,
                        reveal: true,
                        submitLabel: "Start the chapter"
                    ) { responses in
                        await session.submitPretest(responses)
                    }
                }
            case .check:
                if let ch = session.chapter {
                    ItemFlowView(
                        title: "Comprehension check - \(ch.title)",
                        subtitle: "Closed book. Rate your confidence before each reveal.",
                        items: ch.check,
                        reveal: true,
                        submitLabel: "Submit check",
                        onExit: { session.leaveCheck() }
                    ) { responses in
                        await session.submitCheck(responses)
                    }
                }
            case .results(let doc, let gate):
                ResultsView(doc: doc, gate: gate)
            case .authoring:
                AuthoringView()
            case .takingBreak(let suggestion):
                BreakView(suggestion: suggestion)
            case .error(let message):
                ErrorView(message: message)
            }

            if let wait = session.wait {
                WaitOverlay(text: wait.rawValue)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        // The chrome takes its own strip at the top rather than floating
        // over the page: at large text sizes a floating strip sat on the
        // headline. A tap on the page hides it while reading. Labelled, not
        // bare glyphs: on the mini these were two small icons in the corner
        // with nothing to say what they did, and the way back out of a
        // chapter should not be a guess.
        .safeAreaInset(edge: .top, spacing: 0) {
            if !chromeHidden {
                HStack(spacing: 10) {
                    Spacer()
                    ConnectionBadge()
                    Button { session.contentsShown.toggle() } label: {
                        Label("Contents", systemImage: "list.bullet.rectangle")
                            .font(.footnote)
                            .padding(.horizontal, 10).padding(.vertical, 6)
                    }
                    .background(.thinMaterial, in: Capsule())
                    .hoverEffect()
                    .keyboardShortcut("c", modifiers: [.command, .shift])
                    .accessibilityLabel("Contents")
                    Button { library.closeBook() } label: {
                        Label("Bookshelf", systemImage: "books.vertical")
                            .font(.footnote)
                            .padding(.horizontal, 10).padding(.vertical, 6)
                    }
                    .background(.thinMaterial, in: Capsule())
                    .hoverEffect()
                    .accessibilityLabel("Bookshelf")
                }
                .buttonStyle(.plain)
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
            }
        }
        .animation(.easeInOut(duration: 0.2), value: chromeHidden)
    }
}

/// The shelf: one card per book the server offers, the open one marked.
struct BookshelfView: View {
    @EnvironmentObject var library: Library
    @State private var showSettings = false

    var body: some View {
        VStack(spacing: 28) {
            VStack(spacing: 10) {
                Image("Logo")
                    .resizable()
                    .scaledToFit()
                    .frame(height: 160)
                    .accessibilityHidden(true)
                Text("Chiron").font(Typography.display(52))
                Text("The bookshelf")
                    .font(Typography.serifItalic(20))
                    .foregroundStyle(.secondary)
            }
            VStack(spacing: 12) {
                ForEach(library.subjects) { s in
                    Button {
                        Task { await library.open(s.id) }
                    } label: {
                        HStack {
                            VStack(alignment: .leading, spacing: 3) {
                                Text(s.title).font(Typography.serif(22, weight: .semibold, relativeTo: .title3))
                                Text(s.id == library.activeSubjectID ? "Open now" : s.progressLine)
                                    .font(Typography.sans(14, relativeTo: .caption)).foregroundStyle(.secondary)
                            }
                            Spacer()
                            Image(systemName: "chevron.right").foregroundStyle(.secondary)
                        }
                        .padding(16)
                        .frame(maxWidth: 480)
                        .background(Color.gray.opacity(0.14), in: RoundedRectangle(cornerRadius: 14))
                    }
                    .buttonStyle(.plain)
                    .hoverEffect()
                    .accessibilityLabel(s.title)
                    .accessibilityHint(s.id == library.activeSubjectID ? "Open now" : s.progressLine)
                }
                if library.subjects.isEmpty && !library.loadingShelf {
                    Text(library.shelfError ?? "No books on the shelf.")
                        .foregroundStyle(.secondary)
                }
            }
            Button {
                library.teaching = true
            } label: {
                Label("Teach me something else", systemImage: "sparkles")
                    .font(.callout)
            }
            .buttonStyle(.bordered)
            .disabled(!library.sync.connected)
            HStack(spacing: 14) {
                ConnectionBadge()
                Button {
                    Task { await library.refresh() }
                } label: { Image(systemName: "arrow.clockwise") }
                    .accessibilityLabel("Refresh the shelf")
                Button {
                    showSettings = true
                } label: { Label("Server", systemImage: "gearshape") }
                    .font(.callout)
            }
            if let err = library.shelfError, !library.subjects.isEmpty {
                Text(err).foregroundStyle(.red).font(.callout)
            }
        }
        .task { await library.refresh() }
        .sheet(isPresented: $showSettings) { ConnectionSettings(store: library.sync.servers) }
    }
}

/// Saved servers: pick one, add, edit, delete.
struct ConnectionSettings: View {
    @EnvironmentObject var library: Library
    @Environment(\.presentationMode) private var presentation
    @ObservedObject private var store: ServerStore
    @State private var editing: SavedServer?
    @State private var addingNew = false

    init(store: ServerStore) { self.store = store }

    var body: some View {
        NavigationView {
            List {
                Section {
                    if store.servers.isEmpty {
                        Text("No servers saved yet.").foregroundStyle(.secondary)
                    }
                    ForEach(store.servers) { server in
                        Button {
                            store.select(server)
                            library.sync.baseURL = server.url
                            Task { await refresh() }
                        } label: {
                            HStack {
                                Image(systemName: server.id == store.selectedID
                                      ? "largecircle.fill.circle" : "circle")
                                    .foregroundStyle(server.id == store.selectedID ? Color.accentColor : .secondary)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(server.name)
                                    Text(server.url).font(.caption).foregroundStyle(.secondary)
                                }
                                Spacer()
                                Button {
                                    editing = server
                                } label: {
                                    Image(systemName: "pencil")
                                }
                                .buttonStyle(.borderless)
                                .accessibilityLabel("Edit \(server.name)")
                            }
                        }
                        .buttonStyle(.plain)
                    }
                    .onDelete { offsets in
                        offsets.map { store.servers[$0] }.forEach(store.delete)
                        if let s = store.selected { library.sync.baseURL = s.url }
                        Task { await library.sync.probe() }
                    }
                } header: {
                    Text("Servers")
                } footer: {
                    Text("Swipe a server to delete it. The shared key is only needed for a server on the open internet.")
                }

                Section {
                    Button {
                        addingNew = true
                    } label: {
                        Label("Add a server", systemImage: "plus")
                    }
                }

                Section { ConnectionBadge() }
            }
            .navigationTitle("Connection")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { presentation.wrappedValue.dismiss() }
                }
            }
        }
        .navigationViewStyle(.stack)
        .sheet(isPresented: $addingNew) {
            ServerEditor(store: store, server: nil) { Task { await refresh() } }
        }
        .sheet(item: $editing) { server in
            ServerEditor(store: store, server: server) { Task { await refresh() } }
        }
    }

    private func refresh() async {
        if let s = store.selected { library.sync.baseURL = s.url }
        await library.sync.probe()
        await library.refresh()
    }
}

/// Add or edit one server.
struct ServerEditor: View {
    @Environment(\.presentationMode) private var presentation
    @ObservedObject var store: ServerStore
    let server: SavedServer?
    let onSave: () -> Void

    @State private var name = ""
    @State private var url = ""
    @State private var key = ""
    /// Typing a shared key on a tablet keyboard without being able to see it is
    /// how you end up debugging a 401 that was a transposed character.
    @State private var revealKey = false

    var body: some View {
        NavigationView {
            Form {
                Section("Name") {
                    TextField("Mac, sprite, ...", text: $name)
                        .disableAutocorrection(true)
                }
                Section("Address") {
                    TextField("http://192.168.2.1:8080", text: $url)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .keyboardType(.URL)
                }
                Section {
                    HStack(spacing: 8) {
                        Group {
                            if revealKey {
                                TextField("blank on the local network", text: $key)
                            } else {
                                SecureField("blank on the local network", text: $key)
                            }
                        }
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        Button {
                            revealKey.toggle()
                        } label: {
                            Image(systemName: revealKey ? "eye.slash" : "eye")
                        }
                        .buttonStyle(.borderless)
                        .accessibilityLabel(revealKey ? "Hide the key" : "Show the key")
                    }
                } header: {
                    Text("Shared key")
                } footer: {
                    Text("Only needed for a server reachable from the open internet.")
                }
            }
            .navigationTitle(server == nil ? "Add a server" : "Edit server")
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { presentation.wrappedValue.dismiss() }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Save") {
                        if let existing = server {
                            store.update(existing, name: name, url: url, key: key)
                        } else {
                            store.add(name: name, url: url, key: key)
                        }
                        onSave()
                        presentation.wrappedValue.dismiss()
                    }
                    .disabled(url.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .onAppear {
                if let s = server {
                    name = s.name
                    url = s.url
                    key = Credentials.token(for: s.id) ?? ""
                }
            }
        }
        .navigationViewStyle(.stack)
    }
}

struct ConnectionBadge: View {
    @EnvironmentObject var library: Library

    var body: some View {
        let connected = library.sync.connected
        Label(connected ? "connected" : "offline", systemImage: connected ? "wifi" : "wifi.slash")
            .font(.caption)
            .padding(.horizontal, 8).padding(.vertical, 4)
            .background(.thinMaterial, in: Capsule())
            .foregroundStyle(connected ? .green : .orange)
            .accessibilityLabel(connected ? "Server connected" : "Server offline")
            .task {
                while !Task.isCancelled {
                    await library.sync.probe()
                    try? await Task.sleep(nanoseconds: 15_000_000_000)
                }
            }
    }
}
