import PDFKit
import SwiftUI

/// A PDF open in the reader: which page the reader is on, kept on the
/// server so the other device opens there too.
@MainActor
final class DocumentSession: ObservableObject, Identifiable {
    let id: String
    let title: String
    let pages: Int
    let fileURL: URL
    @Published private(set) var page: Int
    private(set) var position: Double
    /// A page the app asked the view to show (the harness, a link).
    @Published var requestedPage: Int?
    var onTurn: ((Int, Double) -> Void)?

    init(id: String, title: String, pages: Int, page: Int, position: Double, fileURL: URL) {
        self.id = id; self.title = title; self.pages = pages; self.fileURL = fileURL
        self.page = page; self.position = position
    }

    /// The view reports where the reader is.
    func turned(to page: Int, position: Double) {
        guard page != self.page || position != self.position else { return }
        self.page = page
        self.position = position
        onTurn?(page, position)
    }

    /// Show a page, and record it as read.
    func go(to page: Int) {
        requestedPage = page
        turned(to: page, position: 0)
    }
}

struct DocumentReaderView: View {
    @EnvironmentObject var library: Library
    @ObservedObject var doc: DocumentSession

    var body: some View {
        NavigationStack {
            PDFKitView(doc: doc)
                .ignoresSafeArea(edges: .bottom)
                .navigationTitle(doc.title)
                .toolbarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Button { library.closeBook() } label: {
                            Label("Bookshelf", systemImage: "books.vertical")
                        }
                        .accessibilityLabel("Bookshelf")
                    }
                    ToolbarItem(placement: .topBarTrailing) {
                        Text(doc.pages > 0 ? "\(doc.page + 1) of \(doc.pages)" : "")
                            .font(Typography.sans(14, relativeTo: .caption))
                            .foregroundStyle(.secondary)
                            .accessibilityLabel("Page \(doc.page + 1) of \(doc.pages)")
                    }
                }
        }
    }
}

/// PDFKit's view, scrolling page after page, reporting the page in view.
struct PDFKitView: UIViewRepresentable {
    @ObservedObject var doc: DocumentSession

    func makeUIView(context: Context) -> PDFView {
        let view = PDFView()
        view.autoScales = true
        view.displayMode = .singlePageContinuous
        view.displayDirection = .vertical
        view.document = PDFDocument(url: doc.fileURL)
        view.backgroundColor = .systemBackground
        context.coordinator.view = view
        if let page = view.document?.page(at: doc.page) {
            // The document lays out after it is in a window; going to the
            // page before that lands on the first page.
            DispatchQueue.main.async { view.go(to: page) }
        }
        NotificationCenter.default.addObserver(context.coordinator, selector: #selector(Coordinator.pageChanged),
                                               name: .PDFViewPageChanged, object: view)
        return view
    }

    func updateUIView(_ view: PDFView, context: Context) {
        if let wanted = doc.requestedPage {
            if let page = view.document?.page(at: wanted), view.currentPage != page {
                view.go(to: page)
            }
            DispatchQueue.main.async { doc.requestedPage = nil }
        }
    }

    func makeCoordinator() -> Coordinator { Coordinator(doc: doc) }

    final class Coordinator: NSObject {
        let doc: DocumentSession
        weak var view: PDFView?
        init(doc: DocumentSession) { self.doc = doc }

        @objc func pageChanged() {
            guard let view, let page = view.currentPage, let document = view.document else { return }
            let index = document.index(for: page)
            Task { @MainActor in self.doc.turned(to: index, position: 0) }
        }
    }
}
