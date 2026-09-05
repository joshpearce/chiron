import SwiftUI
import VisionKit

/// The selected server as a QR code, for the Camera app or Chiron's own
/// scanner on another device. The code carries the shared key, so it is
/// for the reader's own devices and says so.
struct DeviceSetupView: View {
    @EnvironmentObject var library: Library
    @Environment(\.dismiss) private var dismiss

    private var link: ServerLink? {
        guard let s = library.sync.servers.selected else { return nil }
        return ServerLink(name: s.name, url: s.url, key: Credentials.token(for: s.id))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                if let link, let image = QRCode.image(of: link.asURL.absoluteString) {
                    Image(uiImage: image)
                        .resizable()
                        .interpolation(.none)
                        .scaledToFit()
                        .frame(maxWidth: 320)
                        .padding(16)
                        .background(.white, in: .rect(cornerRadius: 22))
                        .accessibilityLabel("Setup code for \(link.name)")
                    VStack(spacing: 6) {
                        Text(link.name).font(Typography.sans(20, weight: .semibold))
                        Text(link.url).font(.footnote.monospaced()).foregroundStyle(.secondary)
                    }
                    Text("On your other device, point the Camera at this code and open it in Chiron, or use \"Scan a code\" under Add a server.")
                        .font(.callout)
                        .multilineTextAlignment(.center)
                    Text(link.key == nil
                         ? "This server has no shared key."
                         : "The code carries the server's shared key. Show it to your own devices only.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                } else {
                    Text("Select a server first.").foregroundStyle(.secondary)
                }
            }
            .padding(24)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .navigationTitle("Set up another device")
            .toolbarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
    }
}

/// The camera, looking for one setup code. Calls back with the link the
/// first time a chiron://server code is in view.
struct CodeScanner: View {
    let found: (ServerLink) -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Group {
                if DataScannerViewController.isSupported && DataScannerViewController.isAvailable {
                    ScannerView(found: found)
                        .ignoresSafeArea()
                } else {
                    Text("Scanning needs a camera. Type the server in instead, or open the code with the Camera app.")
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                        .padding()
                }
            }
            .navigationTitle("Scan a code")
            .toolbarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
}

private struct ScannerView: UIViewControllerRepresentable {
    let found: (ServerLink) -> Void

    func makeUIViewController(context: Context) -> DataScannerViewController {
        let scanner = DataScannerViewController(
            recognizedDataTypes: [.barcode(symbologies: [.qr])],
            qualityLevel: .balanced,
            isHighlightingEnabled: true)
        scanner.delegate = context.coordinator
        return scanner
    }

    func updateUIViewController(_ scanner: DataScannerViewController, context: Context) {
        if !scanner.isScanning { try? scanner.startScanning() }
    }

    func makeCoordinator() -> Coordinator { Coordinator(found: found) }

    final class Coordinator: NSObject, DataScannerViewControllerDelegate {
        let found: (ServerLink) -> Void
        private var done = false
        init(found: @escaping (ServerLink) -> Void) { self.found = found }

        func dataScanner(_ scanner: DataScannerViewController, didAdd added: [RecognizedItem], allItems: [RecognizedItem]) {
            guard !done else { return }
            for item in added {
                if case .barcode(let code) = item, let text = code.payloadStringValue,
                   let url = URL(string: text), let link = ServerLink(url) {
                    done = true
                    scanner.stopScanning()
                    found(link)
                    return
                }
            }
        }
    }
}
