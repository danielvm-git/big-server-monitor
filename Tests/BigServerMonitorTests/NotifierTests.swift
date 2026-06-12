import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct NotifierTests {

    @Test func shouldNotifyFirstTime() async {
        let notifier = Notifier(rateLimitSeconds: 60)
        let result = await notifier.shouldNotify(port: 3000)
        #expect(result == true)
    }

    @Test func rateLimitsWithinWindow() async {
        let notifier = Notifier(rateLimitSeconds: 60)
        _ = await notifier.shouldNotify(port: 3000)
        await notifier.recordNotification(port: 3000)

        // Same port immediately — should be rate limited
        let result = await notifier.shouldNotify(port: 3000)
        #expect(result == false)
    }

    @Test func differentPortsAreIndependent() async {
        let notifier = Notifier(rateLimitSeconds: 60)
        _ = await notifier.shouldNotify(port: 3000)
        await notifier.recordNotification(port: 3000)

        // Different port — should notify
        let result = await notifier.shouldNotify(port: 8080)
        #expect(result == true)
    }

    @Test func rateLimitResetsAfterWindow() async {
        let notifier = Notifier(rateLimitSeconds: 0) // 0-second window for testing
        _ = await notifier.shouldNotify(port: 3000)
        await notifier.recordNotification(port: 3000, timestamp: Date().addingTimeInterval(-10))

        // Window expired — should notify again
        let result = await notifier.shouldNotify(port: 3000)
        #expect(result == true)
    }

    @Test func formatsCrashMessage() {
        var server = Server(
            port: 3000,
            processName: "node",
            pid: 1234,
            status: .offline,
            projectName: "bigbase-api",
            startedAt: Date().addingTimeInterval(-7200) // 2 hours ago
        )

        let message = formatCrashMessage(for: server, duration: 7200)
        #expect(message.contains("bigbase-api"))
        #expect(message.contains("node"))
        #expect(message.contains("3000"))
        #expect(message.contains("2h"))
    }

    @Test func formatsCrashMessageWithoutProjectName() {
        var server = Server(
            port: 8080,
            processName: "python3",
            pid: 5678,
            status: .offline,
            startedAt: Date().addingTimeInterval(-60)
        )

        let message = formatCrashMessage(for: server, duration: 60)
        #expect(message.contains("python3"))
        #expect(message.contains("8080"))
        #expect(message.contains("1m"))
        #expect(!message.contains("(")) // no project name = no parenthetical
    }
}
