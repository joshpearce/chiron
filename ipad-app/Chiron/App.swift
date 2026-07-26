import SwiftUI

@main
struct ChironApp: App {
    @StateObject private var model = AppModel()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(model)
                .task {
                    if SelfTest.requested {
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
                if model.chapter != nil {
                    ReaderContainer()
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
            }

            if model.sync.busy {
                GeneratingOverlay()
            }
        }
        .sheet(isPresented: $showSpine) { SpineView() }
        .overlay(alignment: .topTrailing) {
            if !model.isMenu {
                HStack(spacing: 12) {
                    ConnectionBadge()
                    Button { showSpine = true } label: {
                        Image(systemName: "list.bullet.rectangle")
                    }
                    Button { model.backToLibrary() } label: {
                        Image(systemName: "books.vertical")
                    }
                }
                .padding(10)
            }
        }
    }
}

struct LibraryView: View {
    @EnvironmentObject var model: AppModel

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
            HStack {
                ConnectionBadge()
                Button {
                    Task { await model.refreshSubjects() }
                } label: { Image(systemName: "arrow.clockwise") }
            }
            if let err = model.errorMessage {
                Text(err).foregroundStyle(.red).font(.callout)
            }
        }
        .task { await model.refreshSubjects() }
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
    @State private var url: String = ""

    var body: some View {
        VStack(spacing: 24) {
            Text(model.currentSubjectTitle)
                .font(.system(size: 40, weight: .semibold, design: .serif))
            HStack {
                TextField("server", text: $url)
                    .textFieldStyle(.roundedBorder)
                    .autocapitalization(.none)
                    .disableAutocorrection(true)
                    .frame(width: 280)
                Button("Save") { model.sync.baseURL = url; Task { await model.sync.probe() } }
            }
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
            Button("Back to library") { model.backToLibrary() }
                .buttonStyle(.plain).foregroundStyle(.secondary)
            if let err = model.errorMessage {
                Text(err).foregroundStyle(.red).font(.callout)
            }
        }
        .onAppear { url = model.sync.baseURL }
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
