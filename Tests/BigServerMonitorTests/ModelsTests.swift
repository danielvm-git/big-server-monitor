import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct ModelsTests {
    @Test func uptimeFormatsHoursAndMinutes() {
        var server = Server(
            port: 3000, processName: "node", pid: 1024, status: .online,
            projectName: "bigbase-api", projectPath: nil, binaryPath: nil,
            memoryMB: 124, startedAt: Date(timeIntervalSinceNow: -(2 * 3600 + 3 * 60))
        )
        #expect(server.uptime == "2h 03m")

        server.startedAt = Date(timeIntervalSinceNow: -45 * 60)
        #expect(server.uptime == "45m")
    }

    @Test func uptimeIsDashWhenOffline() {
        let server = Server(
            port: 8080, processName: "python3", pid: nil, status: .offline,
            projectName: nil, projectPath: nil, binaryPath: nil,
            memoryMB: nil, startedAt: nil
        )
        #expect(server.uptime == "—")
    }
}
