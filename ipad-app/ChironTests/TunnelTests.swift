import Network
import XCTest
@testable import Chiron

final class TunnelTests: XCTestCase {
    /// The loopback port opens and takes a connection from this process;
    /// nothing about the far end is needed for that.
    func testTunnelOpensALoopbackPortAndAccepts() async throws {
        let t = try await Tunnel.open(base: "http://localhost:1", key: "k")
        XCTAssertNotEqual(t.port, 0)
        let conn = NWConnection(host: "127.0.0.1", port: NWEndpoint.Port(rawValue: t.port)!, using: .tcp)
        let state: String = await withCheckedContinuation { cont in
            var done = false
            conn.stateUpdateHandler = { s in
                guard !done else { return }
                switch s {
                case .ready: done = true; cont.resume(returning: "ready")
                case .failed(let e): done = true; cont.resume(returning: "failed \(e)")
                case .cancelled: done = true; cont.resume(returning: "cancelled")
                default: break
                }
            }
            conn.start(queue: .global())
        }
        XCTAssertEqual(state, "ready")
        conn.cancel()
        t.close()
    }

    func testTunnelRejectsAnUnusableAddress() async {
        do {
            _ = try await Tunnel.open(base: "not a url", key: nil)
            XCTFail("opened a tunnel to nowhere")
        } catch {
            XCTAssertTrue("\(error)".contains("url"), "\(error)")
        }
    }
}
