import SwiftUI
import UniformTypeIdentifiers

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
                    #if targetEnvironment(macCatalyst)
                    MacServices.register()
                    #endif
                    #if DEBUG
                    KeyboardProbe.shared.start()
                    if Harness.requested {
                        Harness.shared.start(library)
                    }
                    #endif
                    if SelfTest.teachRequested {
                        library.teaching = true
                    } else if SelfTest.requested {
                        await SelfTest.run(library)
                    } else {
                        await library.launch()
                    }
                }
                .onChange(of: scenePhase) { phase in
                    if phase == .inactive || phase == .background {
                        library.session?.persist()
                    }
                }
                // The share extension and the capture intent hand over
                // through a URL naming a capture in the shared inbox; the
                // Camera app hands over another device's server setup.
                .onOpenURL { url in
                    #if DEBUG
                    library.lastOpenedURL = url.absoluteString
                    #endif
                    if let id = CaptureInbox.captureID(in: url) {
                        library.receiveCapture(id: id)
                    } else if let link = ServerLink(url) {
                        Task { await library.adopt(link) }
                    }
                }
                .onReceive(CaptureRouter.arrivals) { id in library.receiveCapture(id: id) }
        }
    }
}

struct ContentView: View {
    @EnvironmentObject var library: Library

    var body: some View {
        Group {
            if library.teaching {
                NavigationStack {
                    TeachView(sync: library.sync, demo: SelfTest.teachDemo)
                }
            } else if let session = library.session {
                BookView()
                    .environmentObject(session)
                    .id(session.subjectID)
            } else if let doc = library.document {
                DocumentReaderView(doc: doc)
                    .id(doc.id)
            } else {
                BookshelfView()
            }
        }
        // Fill the screen so the top-right chrome pins to the display corner.
        // Without this the stack shrinks to its content and the badge drifts
        // into the middle of the page on short screens.
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .sheet(isPresented: $library.requestsShown) {
            RequestsCard()
                .environmentObject(library)
        }
        .sheet(isPresented: $library.shellShown) {
            ShellView(shell: library.shell)
                .environmentObject(library)
                .presentationSizing(.page)
                .interactiveDismissDisabled()
        }
        .sheet(item: $library.pendingCapture) { capture in
            CaptureCard(capture: capture)
                .environmentObject(library)
        }
        .sheet(item: $library.planning) { _ in
            PlanCard()
                .environmentObject(library)
        }
        .sheet(isPresented: $library.deviceSetupShown) {
            DeviceSetupView()
                .environmentObject(library)
        }
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
                .environmentObject(session)
        }
        .sheet(item: $session.conflict) { c in
            ConflictCard(conflict: c)
                .environmentObject(session)
                .presentationDetents([.medium, .large])
        }
    }

    /// The page's chrome is the navigation bar and, while reading a book,
    /// a bottom bar: system bars, so they take the system's glass and the
    /// page scrolls beneath them. A tap on the page hides both.
    private var page: some View {
        NavigationStack {
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
            .navigationTitle(session.title)
            .toolbarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button { library.closeBook() } label: {
                        Label("Bookshelf", systemImage: "books.vertical")
                    }
                    .accessibilityLabel("Bookshelf")
                }
                ToolbarItem(placement: .topBarTrailing) {
                    ConnectionBadge(compact: true)
                }
                ToolbarSpacer(.fixed, placement: .topBarTrailing)
                ToolbarItemGroup(placement: .topBarTrailing) {
                    Button { session.contentsShown.toggle() } label: {
                        Label("Contents", systemImage: "list.bullet.rectangle")
                    }
                    .keyboardShortcut("c", modifiers: [.command, .shift])
                    .accessibilityLabel("Contents")
                    RequestButton()
                    ShellButton()
                }
                // A book read as it is turns pages: it has no check at the
                // end of a chapter to carry the reader on.
                if case .reading = session.screen, session.readsAsIs, session.chapter != nil {
                    ToolbarItemGroup(placement: .bottomBar) {
                        Button {
                            Task { await session.turnTo(session.previousChapter) }
                        } label: {
                            Label(session.previousChapter?.title ?? "Back", systemImage: "chevron.left")
                                .lineLimit(1)
                        }
                        .disabled(session.previousChapter == nil || session.busy)
                        .accessibilityLabel("Previous chapter")
                        Spacer()
                        Button {
                            Task { await session.turnTo(session.nextChapter) }
                        } label: {
                            Label(session.nextChapter?.title ?? "On", systemImage: "chevron.right")
                                .labelStyle(.titleAndIcon)
                                .lineLimit(1)
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(session.nextChapter == nil || session.busy)
                        .keyboardShortcut(.rightArrow, modifiers: .command)
                        .accessibilityLabel("Next chapter")
                    }
                }
                if case .reading = session.screen, !session.readsAsIs, let chapter = session.chapter {
                    ToolbarItemGroup(placement: .bottomBar) {
                        if let state = session.bookState, !state.debt.isEmpty {
                            Button {
                                Task { await session.catchMeUp() }
                            } label: {
                                Label("Catch me up", systemImage: "arrow.uturn.backward.circle")
                            }
                            .disabled(session.busy)
                        }
                        Spacer()
                        Button {
                            session.beginCheck()
                        } label: {
                            Text("Take the check")
                        }
                        .buttonStyle(.borderedProminent)
                        .keyboardShortcut(.return, modifiers: .command)
                        .disabled(session.busy)
                        .accessibilityHint("\(chapter.check.count) questions, closed book")
                        Button("Skip") {
                            Task { await session.skipCheck() }
                        }
                        .tint(.orange)
                        .disabled(session.busy)
                    }
                }
            }
            .toolbar(chromeHidden ? .hidden : .visible, for: .navigationBar, .bottomBar)
        }
        .animation(.easeInOut(duration: 0.2), value: chromeHidden)
    }
}

/// The library: the shelves the reader has made, then every book and
/// primer on no shelf, the open one marked. A shelf opens on its own
/// screen; a card dragged onto a shelf is filed there.
/// A newer build of the app, made on the MacBook, ready to install over
/// this one. Install hands the manifest to iOS, which asks and installs;
/// the Mac gets the zip.
struct BuildBanner: View {
    let build: AppBuild
    let url: URL

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "arrow.down.app")
                .font(.title2)
                .foregroundStyle(.tint)
            VStack(alignment: .leading, spacing: 2) {
                Text("\(build.label) is ready").font(Typography.sans(16, weight: .semibold))
                Text("commit \(build.commit)").font(.footnote.monospaced()).foregroundStyle(.secondary)
            }
            Spacer()
            Button("Install") { UIApplication.shared.open(url) }
                .buttonStyle(.borderedProminent)
        }
        .padding(14)
        .background(.quaternary.opacity(0.5), in: .rect(cornerRadius: 16))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(build.label) is ready to install")
    }
}

struct BookshelfView: View {
    @EnvironmentObject var library: Library

    @State private var namingShelf = false
    @State private var importingPDF = false
    @State private var newShelfName = ""
    @State private var readingLink = false
    @State private var link = ""

    var body: some View {
        NavigationStack(path: $library.shelfPath) {
            shelf
                .toolbarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        ConnectionBadge(compact: true)
                    }
                    ToolbarItemGroup(placement: .topBarTrailing) {
                        Button {
                            library.pendingCapture = Capture()
                        } label: {
                            Label("Capture", systemImage: "text.badge.plus")
                        }
                        .disabled(!library.sync.connected)
                        .accessibilityHint("Paste something and ask about it; a primer appears in the library")
                        Button {
                            library.teaching = true
                        } label: {
                            Label("Teach me something else", systemImage: "sparkles")
                        }
                        .disabled(!library.sync.connected)
                        Button {
                            newShelfName = ""
                            namingShelf = true
                        } label: {
                            Label("New shelf", systemImage: "folder.badge.plus")
                        }
                        .disabled(!library.sync.connected)
                        Button {
                            importingPDF = true
                        } label: {
                            Label("Import a PDF or EPUB", systemImage: "doc.badge.plus")
                        }
                        .disabled(!library.sync.connected)
                        .accessibilityHint("A PDF from Files goes on the shelf")
                        Button {
                            link = UIPasteboard.general.url?.absoluteString ?? ""
                            readingLink = true
                        } label: {
                            Label("Read a link or follow a blog", systemImage: "link.badge.plus")
                        }
                        .disabled(!library.sync.connected)
                        .accessibilityHint("A page on the web is read here; a feed is followed as a card")
                    }
                    ToolbarSpacer(.fixed, placement: .topBarTrailing)
                    ToolbarItemGroup(placement: .topBarTrailing) {
                        Button {
                            Task { await library.refresh() }
                        } label: { Label("Refresh the library", systemImage: "arrow.clockwise") }
                        Button {
                            library.settingsShown = true
                        } label: { Label("Server", systemImage: "gearshape") }
                        RequestButton()
                        ShellButton()
                    }
                }
                .navigationDestination(for: String.self) { id in
                    ShelfContentsView(shelfID: id)
                }
        }
        .task { await library.refresh() }
        // Every sheet carries the objects its view asks the environment
        // for. A sheet is its own presentation, and on the Mac its own
        // window: what the presenting view holds does not reach it, and a
        // view that asks for what is not there stops the app.
        .sheet(isPresented: $library.settingsShown) {
            ConnectionSettings(store: library.sync.servers)
                .environmentObject(library)
        }
        .fileImporter(isPresented: $importingPDF, allowedContentTypes: [.pdf, .epub]) { result in
            if case .success(let url) = result {
                Task {
                    if url.pathExtension.lowercased() == "epub" {
                        await library.importBook(at: url)
                    } else {
                        await library.importPDF(at: url)
                    }
                }
            }
        }
        .alert("Read a link", isPresented: $readingLink) {
            TextField("https://", text: $link)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button("Read it") { Task { await library.readPage(link) } }
            Button("Follow the blog") { Task { await library.followFeed(link) } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("A page is kept here to read and ask about. A feed is followed: its posts become chapters, newest first.")
        }
        .alert("New shelf", isPresented: $namingShelf) {
            TextField("Name", text: $newShelfName)
            Button("Make it") { Task { await library.createShelf(named: newShelfName) } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("A shelf holds the books and primers you keep together.")
        }
    }

    private var shelf: some View {
        ScrollView {
            VStack(spacing: 28) {
                VStack(spacing: 10) {
                    Image("Logo")
                        .resizable()
                        .scaledToFit()
                        .frame(height: 160)
                        .accessibilityHidden(true)
                    Text("Chiron").font(Typography.display(52))
                    Text("Library")
                        .font(Typography.serifItalic(20))
                        .foregroundStyle(.secondary)
                }
                if let build = library.availableBuild, let url = library.installURL {
                    BuildBanner(build: build, url: url)
                }
                VStack(spacing: 12) {
                    ForEach(library.shelves) { shelf in
                        ShelfFolderCard(shelf: shelf)
                    }
                    ForEach(library.unfiled) { s in
                        ShelfCard(subject: s)
                    }
                    if library.subjects.isEmpty && !library.loadingShelf {
                        Text(library.shelfError ?? "Nothing in the library yet.")
                            .foregroundStyle(.secondary)
                    }
                }
                if let err = library.shelfError, !library.subjects.isEmpty {
                    Text(err).foregroundStyle(.red).font(.callout)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 24)
            .frame(maxWidth: .infinity)
        }
    }
}

/// Saved servers: pick one, add, edit, delete.
struct ConnectionSettings: View {
    @EnvironmentObject var library: Library
    @Environment(\.dismiss) private var dismiss
    @ObservedObject private var store: ServerStore
    @State private var editing: SavedServer?
    @State private var addingNew = false
    @State private var settingUp = false

    init(store: ServerStore) { self.store = store }

    var body: some View {
        NavigationStack {
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
                    Button {
                        settingUp = true
                    } label: {
                        Label("Set up another device", systemImage: "qrcode")
                    }
                    .disabled(store.selected == nil)
                }

                DeviceKeySection(sync: library.sync)

                AgentSection(agent: library.agent)

                Section { ConnectionBadge() }
            }
            .navigationTitle("Connection")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
        .sheet(isPresented: $addingNew) {
            ServerEditor(store: store, server: nil) { Task { await refresh() } }
        }
        .sheet(item: $editing) { server in
            ServerEditor(store: store, server: server) { Task { await refresh() } }
        }
        .sheet(isPresented: $settingUp) {
            DeviceSetupView()
                .environmentObject(library)
        }
    }

    private func refresh() async {
        if let s = store.selected { library.sync.baseURL = s.url }
        await library.sync.probe()
        await library.refresh()
        // Saving a server with a shared key is the moment to enrol this
        // device's ssh key: the key proves the right to.
        if let id = library.sync.serverID, Credentials.token(for: id) != nil {
            _ = try? await library.sync.enrolDeviceKey(name: DeviceKeySection.deviceName)
        }
    }
}

/// This device's ssh key on the selected server: enrol it, see what the
/// server holds, revoke what should not be there.
struct DeviceKeySection: View {
    @ObservedObject var sync: Sync
    @State private var keys: [EnrolledKey] = []
    @State private var note: String?
    @State private var fingerprint: String?

    static var deviceName: String {
        let name = UIDevice.current.name
        let allowed = name.filter { $0.isLetter || $0.isNumber || $0 == " " || $0 == "." || $0 == "_" || $0 == "-" || $0 == "'" }
        return allowed.isEmpty ? "iPad" : String(allowed.prefix(64))
    }

    var body: some View {
        Section {
            Button {
                Task { await enrol() }
            } label: {
                Label("Enroll this iPad's key", systemImage: "key")
            }
            .disabled(sync.serverID == nil || Credentials.token(for: sync.serverID!) == nil)
            if let note {
                Text(note).font(.footnote).foregroundStyle(.secondary)
            }
            ForEach(keys) { key in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(key.name.isEmpty ? key.type : key.name)
                        Text(key.fingerprint).font(.caption.monospaced()).foregroundStyle(.secondary)
                    }
                    Spacer()
                    if key.fingerprint == fingerprint {
                        Text("this iPad").font(.caption).foregroundStyle(.secondary)
                    }
                }
                .swipeActions {
                    Button(role: .destructive) {
                        Task { await revoke(key) }
                    } label: { Label("Revoke", systemImage: "trash") }
                }
            }
        } header: {
            Text("Shell access")
        } footer: {
            Text("The shell signs in with a key made on this iPad. Enrolling it needs the server's shared key; swipe a key to revoke it.")
        }
        .task(id: sync.serverID) { await load() }
    }

    private func load() async {
        do {
            keys = try await sync.serverKeys()
            note = nil
        } catch ServiceError.status(404) {
            keys = []
            note = "This server does not enroll keys (a Mac dev server, or the gate is not installed yet)."
        } catch {
            keys = []
            note = nil
        }
    }

    private func enrol() async {
        do {
            let r = try await sync.enrolDeviceKey(name: Self.deviceName)
            fingerprint = r.fingerprint
            note = r.installed ? "Enrolled as \(r.name)." : "Already enrolled as \(r.name)."
            await load()
        } catch ServiceError.status(let code) {
            note = code == 404 ? "This server does not enroll keys." : "The server refused the key (\(code))."
        } catch {
            note = error.localizedDescription
        }
    }

    private func revoke(_ key: EnrolledKey) async {
        try? await sync.revokeKey(fingerprint: key.fingerprint)
        await load()
    }
}

/// Whether the sprite's agent may drive this app.
struct AgentSection: View {
    @ObservedObject var agent: AgentLink

    var body: some View {
        Section {
            Toggle(isOn: Binding(
                get: { agent.enabled },
                set: { on in
                    agent.enabled = on
                    if on { agent.start() } else { agent.stop() }
                })) {
                Label("Let the sprite's agent drive this app", systemImage: "hand.point.up.left")
            }
            if agent.enabled {
                Text(agent.connected ? "The agent is connected." : "Waiting for the server.")
                    .font(.footnote).foregroundStyle(.secondary)
            }
        } header: {
            Text("Agent")
        } footer: {
            Text("With this on, Claude Code on the sprite can open books, answer checks and take screenshots here, and a badge shows while it is connected.")
        }
    }
}

/// The chrome's way into the shell.
struct ShellButton: View {
    @EnvironmentObject var library: Library

    var body: some View {
        Button { library.shellShown = true } label: {
            Label("Shell", systemImage: "terminal")
        }
        .keyboardShortcut("`", modifiers: [.command])
        .accessibilityLabel("Shell")
    }
}

/// Add or edit one server.
struct ServerEditor: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var store: ServerStore
    let server: SavedServer?
    let onSave: () -> Void

    @State private var name = ""
    @State private var url = ""
    @State private var key = ""
    /// Typing a shared key on a tablet keyboard without being able to see it is
    /// how you end up debugging a 401 that was a transposed character.
    @State private var revealKey = false
    @State private var scanning = false

    var body: some View {
        NavigationStack {
            Form {
                if server == nil {
                    Section {
                        if CodeScanner.available {
                            Button {
                                scanning = true
                            } label: {
                                Label("Scan a code", systemImage: "qrcode.viewfinder")
                            }
                        }
                        Button {
                            if let link = ServerLink(pasted: UIPasteboard.general.string ?? "") {
                                name = link.name
                                url = link.url
                                key = link.key ?? ""
                            }
                        } label: {
                            Label("Paste a setup link", systemImage: "doc.on.clipboard")
                        }
                    } footer: {
                        Text("The code, or the copied link, from \"Set up another device\" on a device that already has the server.")
                    }
                }
                Section("Name") {
                    TextField("Mac, sprite, ...", text: $name)
                        .autocorrectionDisabled()
                }
                Section("Address") {
                    TextField("http://192.168.2.1:8080", text: $url)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
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
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
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
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        if let existing = server {
                            store.update(existing, name: name, url: url, key: key)
                        } else {
                            store.add(name: name, url: url, key: key)
                        }
                        onSave()
                        dismiss()
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
            .sheet(isPresented: $scanning) {
                CodeScanner { link in
                    name = link.name
                    url = link.url
                    key = link.key ?? ""
                    scanning = false
                }
            }
        }
    }
}

/// Whether the server is reachable, and whether the sprite's agent has the
/// app. In a toolbar it is the symbols alone; on a page, the words too.
struct ConnectionBadge: View {
    @EnvironmentObject var library: Library
    @ObservedObject private var agentState = AgentBadgeState.shared
    var compact = false

    var body: some View {
        let connected = library.sync.connected
        Group {
            if compact {
                labels(connected).labelStyle(.iconOnly).font(.body)
            } else {
                labels(connected).labelStyle(.titleAndIcon).font(.caption)
            }
        }
        .task {
            while !Task.isCancelled {
                await library.sync.probe()
                try? await Task.sleep(nanoseconds: 15_000_000_000)
            }
        }
    }

    private func labels(_ connected: Bool) -> some View {
        HStack(spacing: 10) {
            if agentState.driven {
                Label("driven by the agent", systemImage: "hand.point.up.left.fill")
                    .foregroundStyle(.purple)
                    .accessibilityLabel("The sprite's agent is driving this app")
            }
            Label(connected ? "connected" : "offline", systemImage: connected ? "wifi" : "wifi.slash")
                .foregroundStyle(connected ? .green : .orange)
                .accessibilityLabel(connected ? "Server connected" : "Server offline")
        }
    }
}


/// The badge's view of the agent link, mirrored so the badge (which is
/// used in several places) needs no extra plumbing.
@MainActor
final class AgentBadgeState: ObservableObject {
    static let shared = AgentBadgeState()
    @Published var driven = false
}


/// One book or primer on the shelf. A book carries its progress; a primer
/// its source and capture date, greyed with a spinner while the server is
/// still writing it, red when that failed. The badge in the corner says
/// which is which without reading.
struct ShelfCard: View {
    @EnvironmentObject var library: Library
    let subject: SubjectInfo

    private var openable: Bool { subject.drafting || (!subject.authoring && !subject.failed) }

    /// The line under the title. The book last open says so; a blog says
    /// so and still says how much of it is waiting.
    private var openLine: String {
        guard subject.id == library.activeSubjectID else { return subject.progressLine }
        if subject.isFeed, let n = subject.unread, n > 0 { return "Open now · \(n) unread" }
        return "Open now"
    }

    var body: some View {
        Button {
            Task {
                if subject.drafting { await library.openDraft(subject.id) } else { await library.open(subject.id) }
            }
        } label: {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(subject.title).font(Typography.serif(22, weight: .semibold, relativeTo: .title3))
                    if subject.authoring {
                        HStack(spacing: 8) {
                            ProgressView().controlSize(.small)
                            Text(subject.building ? subject.progressLine : "Writing the primer · \(subject.sourceLine)")
                        }
                        .font(Typography.sans(14, relativeTo: .caption)).foregroundStyle(.secondary)
                    } else if subject.drafting && !subject.failed {
                        Text(subject.progressLine)
                            .font(Typography.sans(14, relativeTo: .caption)).foregroundStyle(.secondary)
                    } else if subject.failed {
                        Text(subject.error ?? "The primer could not be written.")
                            .font(Typography.sans(14, relativeTo: .caption)).foregroundStyle(.red)
                    } else {
                        Text(openLine)
                            .font(Typography.sans(14, relativeTo: .caption)).foregroundStyle(.secondary)
                    }
                }
                Spacer()
                Image(systemName: "chevron.right").foregroundStyle(.secondary)
            }
            .padding(18)
            .padding(.bottom, 10)
            .frame(maxWidth: 480)
            .background(.fill.tertiary, in: .rect(cornerRadius: 22))
            .overlay(alignment: .bottomTrailing) {
                // A draft wears a hammer beside what it will be: work in
                // progress the reader can pick up again.
                HStack(spacing: 4) {
                    if subject.drafting || subject.building { Text("🔨").font(.footnote) }
                    Image(systemName: subject.isPDF ? "doc.richtext"
                          : subject.isFeed ? "dot.radiowaves.up.forward"
                          : subject.isReading ? "book"
                          : (subject.scale == "book" || !subject.isPrimer ? "brain" : "doc.text"))
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                .padding(10)
                .accessibilityLabel(subject.drafting || subject.building ? "Draft"
                                    : subject.isPDF ? "PDF"
                                    : subject.isFeed ? "Followed blog"
                                    : subject.isReading ? "Imported book"
                                    : (subject.isPrimer ? "Primer" : "Smart book"))
            }
            .opacity(openable ? 1 : 0.55)
        }
        .buttonStyle(.plain)
        .hoverEffect()
        .disabled(!openable)
        .accessibilityLabel(subject.title)
        .accessibilityHint(subject.authoring ? "Still being written" : (subject.id == library.activeSubjectID ? "Open now" : subject.progressLine))
        // Filed by dragging onto a shelf, or by the menu for a hand that
        // would rather not drag.
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
            if subject.isFeed || subject.isReading {
                Button(role: .destructive) {
                    Task { await library.forget(subject.id) }
                } label: {
                    Label(subject.isFeed ? "Unfollow" : "Take off the shelf", systemImage: "trash")
                }
            }
        }
    }
}
