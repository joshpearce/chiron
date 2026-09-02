import SwiftUI
import PencilKit

/// A fixed box to write an answer in by hand, finger or Pencil. The strokes
/// leave as normalized polylines in the box's own coordinates, so the server
/// rasterizes them at the same aspect and the transcriber sees what was
/// written, undistorted. The interpretation lives server-side, next to the
/// grade, exactly as for the e-ink client.
struct InkBox: View {
    @Binding var drawing: PKDrawing
    /// The drawing as the ink contract wants it, kept current by the
    /// canvas, which knows its own bounds; nil while the box is empty.
    @Binding var answer: InkAnswer?
    /// Width over height of the box; the raster on the server matches it.
    static let aspect: Double = 3

    var body: some View {
        InkCanvas(drawing: $drawing, answer: $answer)
            .aspectRatio(Self.aspect, contentMode: .fit)
            .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 10))
            .overlay(RoundedRectangle(cornerRadius: 10).stroke(.quaternary))
            .accessibilityLabel("Handwriting box")
    }

    /// The drawing as the ink contract wants it: polylines normalized to the
    /// box, sampled from the stroke paths. Empty when nothing was written.
    static func answer(from drawing: PKDrawing, in size: CGSize) -> InkAnswer? {
        guard size.width > 0, size.height > 0 else { return nil }
        var strokes: [[InkAnswer.Point]] = []
        for stroke in drawing.strokes {
            let path = stroke.path
            var points: [InkAnswer.Point] = []
            let count = path.count
            guard count > 0 else { continue }
            // Every control point, plus interpolation between them for long
            // gaps, keeps curves smooth in the raster without shipping the
            // full resampled path.
            for i in 0..<count {
                let p = path[i].location
                points.append(InkAnswer.Point(x: clamp(p.x / size.width), y: clamp(p.y / size.height)))
            }
            if points.count == 1 { points.append(points[0]) }
            strokes.append(points)
        }
        return strokes.isEmpty ? nil : InkAnswer(strokes: strokes, aspect: Double(size.width / size.height))
    }

    private static func clamp(_ v: CGFloat) -> Double {
        Double(min(max(v, 0), 1))
    }
}

private struct InkCanvas: UIViewRepresentable {
    @Binding var drawing: PKDrawing
    @Binding var answer: InkAnswer?

    func makeCoordinator() -> Coordinator { Coordinator(drawing: $drawing, answer: $answer) }

    func makeUIView(context: Context) -> PKCanvasView {
        let canvas = PKCanvasView()
        // Finger or Pencil: the mini 4 has no Pencil at all.
        canvas.drawingPolicy = .anyInput
        canvas.tool = PKInkingTool(.pen, color: .label, width: 3)
        canvas.backgroundColor = .clear
        canvas.isOpaque = false
        canvas.isScrollEnabled = false
        canvas.delegate = context.coordinator
        canvas.drawing = drawing
        return canvas
    }

    func updateUIView(_ canvas: PKCanvasView, context: Context) {
        if canvas.drawing != drawing {
            canvas.drawing = drawing
            context.coordinator.publish(canvas)
        }
    }

    final class Coordinator: NSObject, PKCanvasViewDelegate {
        @Binding var drawing: PKDrawing
        @Binding var answer: InkAnswer?
        init(drawing: Binding<PKDrawing>, answer: Binding<InkAnswer?>) {
            _drawing = drawing
            _answer = answer
        }
        func canvasViewDrawingDidChange(_ canvasView: PKCanvasView) {
            drawing = canvasView.drawing
            publish(canvasView)
        }
        func publish(_ canvasView: PKCanvasView) {
            answer = InkBox.answer(from: canvasView.drawing, in: canvasView.bounds.size)
        }
    }
}
