import SwiftUI

@main
struct DynamicBookApp: App {
    @StateObject private var model = AppModel()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(model)
                .persistentSystemOverlays(.hidden)
        }
    }
}

struct ContentView: View {
    @EnvironmentObject var model: AppModel
    @State private var showSpine = false

    var body: some View {
        ZStack {
            switch model.screen {
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
            HStack(spacing: 12) {
                ConnectionBadge()
                Button { showSpine = true } label: {
                    Image(systemName: "list.bullet.rectangle")
                }
            }
            .padding(10)
        }
        .statusBarHidden()
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
                while true {
                    await model.sync.probe()
                    try? await Task.sleep(for: .seconds(15))
                }
            }
    }
}

struct StartView: View {
    @EnvironmentObject var model: AppModel
    @State private var url: String = ""

    var body: some View {
        VStack(spacing: 24) {
            Text("The Book").font(.system(size: 44, weight: .semibold, design: .serif))
            Text("An adaptive course on how AI actually works,\ntuned to you as you read.")
                .multilineTextAlignment(.center)
                .foregroundStyle(.secondary)
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
