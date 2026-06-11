import Foundation
import Network
import Testing
@testable import BigServerMonitor

@Suite struct HealthCheckerTests {
    @Test func classifiesStatusCodes() {
        #expect(classifyHealth(statusCode: 200) == .ok)
        #expect(classifyHealth(statusCode: 204) == .ok)
        #expect(classifyHealth(statusCode: 301) == .warn)
        #expect(classifyHealth(statusCode: 404) == .warn)
        #expect(classifyHealth(statusCode: 500) == .error)
        #expect(classifyHealth(statusCode: 503) == .error)
    }

    @Test func probeAgainstClosedPortIsTimeout() async {
        let checker = HealthChecker(timeout: 1)
        let results = await checker.runAll(ports: [59999])

        #expect(results.count == 1)
        #expect(results[0].status == .timeout)
        #expect(results[0].statusCode == 0)
    }

    @Test func resultsAreCachedPerPort() async {
        let checker = HealthChecker(timeout: 1)
        _ = await checker.runAll(ports: [59998])
        let cached = await checker.results

        #expect(cached[59998] != nil)
    }
}
