import XCTest
@testable import Chiron

/// Every file in fixtures/ is a real server response, captured against a
/// fresh two-subject server by scripts/capture-fixtures.py. The wire models
/// must decode all of them: a single missing key is a total decode failure
/// for the whole exchange, and that is how the app went blind to the
/// calibration series for two months.
final class FixtureDecodingTests: XCTestCase {
    private var fixtures: URL {
        get throws {
            try XCTUnwrap(Bundle(for: Self.self).url(forResource: "fixtures", withExtension: nil))
        }
    }

    private func decode<T: Decodable>(_ type: T.Type, _ name: String) throws -> T {
        let url = try fixtures.appendingPathComponent("\(name).json")
        return try JSONDecoder().decode(type, from: Data(contentsOf: url))
    }

    func testEveryExchangeFixtureDecodes() throws {
        let files = try FileManager.default.contentsOfDirectory(at: try fixtures, includingPropertiesForKeys: nil)
            .filter { $0.lastPathComponent.hasPrefix("exchange-") }
        XCTAssertGreaterThanOrEqual(files.count, 3, "expected start, screener and series exchanges")
        for f in files {
            XCTAssertNoThrow(try JSONDecoder().decode(ExchangeResponse.self, from: Data(contentsOf: f)), f.lastPathComponent)
        }
    }

    func testStartDeliversTheScreenerAlone() throws {
        let r = try decode(ExchangeResponse.self, "exchange-start")
        let ch = try XCTUnwrap(r.chapter)
        XCTAssertEqual(ch.calibration, true)
        XCTAssertEqual(ch.check.count, 1)
        let screener = ch.check[0]
        XCTAssertEqual(screener.check, "screener")
        XCTAssertEqual(screener.options?.count, 5, "five self-rating levels")
        XCTAssertNil(r.gate, "placement is never gated")
    }

    func testScreenerDeliversTheSeriesWithRevealsIntact() throws {
        let r = try decode(ExchangeResponse.self, "exchange-screener")
        let ch = try XCTUnwrap(r.chapter)
        XCTAssertEqual(ch.calibration, true)
        XCTAssertGreaterThan(ch.check.count, 1)
        XCTAssertNil(r.gate)
        for item in ch.check {
            XCTAssertNotEqual(item.check, "screener")
            if item.kind == "mcq" {
                let reveal = try XCTUnwrap(item.reveal?.options, item.id)
                XCTAssertEqual(reveal.filter { $0.correct }.count, 1, "\(item.id): exactly one correct option")
                XCTAssertEqual(reveal.count, item.options?.count, "\(item.id): a reveal per option")
            } else {
                XCTAssertNotNil(item.reveal?.answer, "\(item.id): constructed items carry a reference answer")
            }
        }
    }

    func testGradedSeriesCarriesACalibrationGate() throws {
        let r = try decode(ExchangeResponse.self, "exchange-series")
        let gate = try XCTUnwrap(r.gate)
        XCTAssertEqual(gate.calibration, true)
        XCTAssertTrue(gate.passed, "calibration is measurement, never a verdict")
        XCTAssertNotNil(gate.score)
        XCTAssertFalse(r.results.isEmpty)
        XCTAssertEqual(r.state.spine.first?.unit, "u0")
    }

    func testSubjectsCarryTheActiveBook() throws {
        let r = try decode(SubjectsResponse.self, "subjects")
        XCTAssertEqual(r.subjects.map(\.id), ["ai", "data"])
        XCTAssertEqual(r.active, "data", "the last book exchanged with is the open one")
    }

    func testStateDecodes() throws {
        let st = try decode(BookState.self, "state")
        XCTAssertFalse(st.spine.isEmpty)
        XCTAssertEqual(st.spine[0].unit, "u0")
    }
}
