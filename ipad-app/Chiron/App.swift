import SwiftUI

@main
struct ChironApp: App {
    @StateObject private var model = AppModel()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(model)
                .task {
                    if let server = SelfTest.serverOverride {
                        model.sync.baseURL = server
                        await model.sync.probe()
                    }
                    if SelfTest.teachRequested {
                        model.screen = .teach
                    } else if let screen = SelfTest.inspectScreen {
                        SelfTest.inspect(model, screen: screen)
                    } else if SelfTest.requested {
                        await SelfTest.run(model)
                    }
                }
        }
    }
}

struct ContentView: View {
    @EnvironmentObject var model: AppModel
    @State private var showSpine = false

    var body: some View {
        ZStack {
            switch model.screen {
            case .menu:
                LibraryView()
            case .start:
                StartView()
            case .reading:
                if let chapter = model.chapter {
                    ReaderContainer(chapter: chapter)
                } else {
                    StartView()
                }
            case .pretest:
                if let ch = model.chapter {
                    ItemFlowView(
                        title: "Before you read",
                        subtitle: "You are not supposed to know these yet - answering wrong here is part of how the chapter calibrates.",
                        items: ch.pretest,
                        submitLabel: "Start the chapter"
                    ) { responses in
                        await model.submitPretest(responses)
                    }
                }
            case .check:
                if let ch = model.chapter {
                    ItemFlowView(
                        title: "Comprehension check - \(ch.title)",
                        subtitle: "Closed book. Rate your confidence before each reveal.",
                        items: ch.check,
                        submitLabel: "Submit check"
                    ) { responses in
                        await model.submitCheck(responses)
                    }
                }
            case .gate(let gate, let results):
                GateView(gate: gate, results: results)
            case .takingBreak(let suggestion):
                BreakView(suggestion: suggestion)
            case .teach:
                TeachView(sync: model.sync, demo: SelfTest.teachDemo)
            }

            if model.sync.busy {
                GeneratingOverlay()
            }
        }
        // Fill the screen so the top-right chrome pins to the display corner.
        // Without this the ZStack shrinks to its content and the badge drifts
        // into the middle of the page on short screens like the gate.
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .sheet(isPresented: $showSpine) { SpineView() }
        .overlay(alignment: .topTrailing) {
            if !model.isMenu && !model.isTeaching {
                // Labelled, not bare glyphs: on the mini these were two small
                // icons in the corner with nothing to say what they did, and
                // the way back out of a chapter should not be a guess.
                HStack(spacing: 10) {
                    ConnectionBadge()
                    Button { showSpine = true } label: {
                        Label("Spine", systemImage: "list.bullet.rectangle")
                            .font(.footnote)
                            .padding(.horizontal, 10).padding(.vertical, 6)
                    }
                    .background(.thinMaterial, in: Capsule())
                    Button { model.backToLibrary() } label: {
                        Label("Library", systemImage: "books.vertical")
                            .font(.footnote)
                            .padding(.horizontal, 10).padding(.vertical, 6)
                    }
                    .background(.thinMaterial, in: Capsule())
                }
                .buttonStyle(.plain)
                .padding(.horizontal, 12)
                .padding(.top, 6)
            }
        }
    }
}

struct LibraryView: View {
    @EnvironmentObject var model: AppModel
    @State private var showSettings = false

    var body: some View {
        VStack(spacing: 28) {
            VStack(spacing: 10) {
                Image("Logo")
                    .resizable()
                    .scaledToFit()
                    .frame(height: 160)
                Text("Chiron").font(.system(size: 52, weight: .semibold, design: .serif))
                Text("Choose a subject. The book adapts as you read.")
                    .foregroundStyle(.secondary)
            }
            VStack(spacing: 12) {
                ForEach(model.subjects) { s in
                    Button {
                        Task { await model.openSubject(s.id) }
                    } label: {
                        HStack {
                            VStack(alignment: .leading, spacing: 3) {
                                Text(s.title).font(.title3.weight(.semibold))
                                Text(s.progressLine)
                                    .font(.caption).foregroundStyle(.secondary)
                            }
                            Spacer()
                            Image(systemName: "chevron.right").foregroundStyle(.secondary)
                        }
                        .padding(16)
                        .frame(maxWidth: 480)
                        .background(Color.gray.opacity(0.14), in: RoundedRectangle(cornerRadius: 14))
                    }
                    .buttonStyle(.plain)
                }
            }
            Button {
                model.screen = .teach
            } label: {
                Label("Teach me something else", systemImage: "sparkles")
                    .font(.callout)
            }
            .buttonStyle(.bordered)
            .disabled(!model.sync.connected)
            HStack(spacing: 14) {
                ConnectionBadge()
                Button {
                    Task { await model.refreshSubjects() }
                } label: { Image(systemName: "arrow.clockwise") }
                Button {
                    showSettings = true
                } label: { Label("Server", systemImage: "gearshape") }
                    .font(.callout)
            }
            if let err = model.errorMessage {
                Text(err).foregroundStyle(.red).font(.callout)
            }
        }
        .task { await model.refreshSubjects() }
        .sheet(isPresented: $showSettings) { ConnectionSettings(store: model.sync.servers) }
    }
}

/// Saved servers: pick one, add, edit, delete.
///
/// This lives on the library screen as well as the start screen: once a chapter
/// has been restored the app opens straight into the reader, and the start
/// screen - the only place these fields used to exist - is unreachable without
/// finishing or abandoning the chapter.
struct ConnectionSettings: View {
    @EnvironmentObject var model: AppModel
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
                            model.sync.baseURL = server.url
                            Task {
                                await model.sync.probe()
                                await model.refreshSubjects()
                            }
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
                            }
                        }
                        .buttonStyle(.plain)
                    }
                    .onDelete { offsets in
                        offsets.map { store.servers[$0] }.forEach(store.delete)
                        if let s = store.selected { model.sync.baseURL = s.url }
                        Task { await model.sync.probe() }
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
        if let s = store.selected { model.sync.baseURL = s.url }
        await model.sync.probe()
        await model.refreshSubjects()
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
    @EnvironmentObject var model: AppModel

    var body: some View {
        let t = model.sync.transport
        Label(t, systemImage: t == "wifi" ? "wifi" : t == "usb" ? "cable.connector" : "airplane")
            .font(.caption)
            .padding(.horizontal, 8).padding(.vertical, 4)
            .background(.thinMaterial, in: Capsule())
            .foregroundStyle(t == "offline" ? .orange : .green)
            .task {
                while !Task.isCancelled {
                    await model.sync.probe()
                    try? await Task.sleep(nanoseconds: 15_000_000_000)
                }
            }
    }
}

struct StartView: View {
    @EnvironmentObject var model: AppModel
    @State private var showSettings = false

    var body: some View {
        VStack(spacing: 24) {
            Text(model.currentSubjectTitle)
                .font(.system(size: 40, weight: .semibold, design: .serif))
            Button {
                Task { await model.start() }
            } label: {
                Text(model.bookState == nil ? "Begin" : "Continue")
                    .font(.title2).padding(.horizontal, 40).padding(.vertical, 10)
            }
            .buttonStyle(.borderedProminent)
            Button("No server? Read the built-in book") {
                model.startStatic()
            }
            .buttonStyle(.bordered)
            HStack(spacing: 14) {
                ConnectionBadge()
                Button {
                    showSettings = true
                } label: {
                    Label(model.sync.servers.selected?.name ?? "Server", systemImage: "gearshape")
                }
                .font(.callout)
            }
            Button("Back to library") { model.backToLibrary() }
                .buttonStyle(.plain).foregroundStyle(.secondary)
            if let err = model.errorMessage {
                Text(err).foregroundStyle(.red).font(.callout)
            }
        }
        .sheet(isPresented: $showSettings) {
            ConnectionSettings(store: model.sync.servers)
        }
    }
}

struct GeneratingOverlay: View {
    var body: some View {
        VStack(spacing: 14) {
            ProgressView().controlSize(.large)
            Text("Thinking about what you need next…")
                .font(.callout).foregroundStyle(.secondary)
        }
        .padding(30)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 16))
    }
}
