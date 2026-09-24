import XCTest
@testable import Chiron

/// The transport's second tries: a request the sprite was not up for
/// goes again; one the server answered, or that timed out, does not.
@MainActor
final class SyncTests: XCTestCase {
    private final class Tally { var tries = 0 }

    func testARequestTheSpriteWasNotUpForGoesAgain() async throws {
        let t = Tally()
        let got: Int = try await Sync.retrying(pauses: [0, 0, 0]) {
            t.tries += 1
            if t.tries < 3 { throw URLError(.cannotConnectToHost) }
            return 7
        }
        XCTAssertEqual(got, 7)
        XCTAssertEqual(t.tries, 3, "refused twice, then through")
    }

    func testAGateStillWaitingForItsServerGoesAgain() async throws {
        let t = Tally()
        let got: String = try await Sync.retrying(pauses: [0, 0, 0]) {
            t.tries += 1
            if t.tries == 1 { throw ServiceError.status(502) }
            if t.tries == 2 { throw ServiceError.status(503) }
            return "up"
        }
        XCTAssertEqual(got, "up")
        XCTAssertEqual(t.tries, 3)
    }

    func testARequestTheServerAnsweredDoesNotGoAgain() async {
        for failure in [ServiceError.status(400), ServiceError.status(409), ServiceError.status(500)] {
            let t = Tally()
            do {
                let _: Int = try await Sync.retrying(pauses: [0, 0, 0]) { t.tries += 1; throw failure }
                XCTFail("the failure should have come through")
            } catch {}
            XCTAssertEqual(t.tries, 1, "the server said no; asking again would not change its mind")
        }
    }

    /// A timeout may mean the server took the request and is still on
    /// it; sending it again could do the work twice.
    func testATimeoutDoesNotGoAgain() async {
        let t = Tally()
        do {
            let _: Int = try await Sync.retrying(pauses: [0, 0, 0]) { t.tries += 1; throw URLError(.timedOut) }
            XCTFail("the timeout should have come through")
        } catch {}
        XCTAssertEqual(t.tries, 1)
    }

    func testTheTriesRunOut() async {
        let t = Tally()
        do {
            let _: Int = try await Sync.retrying(pauses: [0, 0]) { t.tries += 1; throw URLError(.cannotConnectToHost) }
            XCTFail("the last refusal should have come through")
        } catch let e as URLError {
            XCTAssertEqual(e.code, .cannotConnectToHost)
        } catch {
            XCTFail("the last refusal comes through as it was: \(error)")
        }
        XCTAssertEqual(t.tries, 3, "one try per pause, and one more")
    }
}
