import AppKit
import Observation
import SwiftUI

enum ActiveSheet: String, Identifiable {
    case health
    case activity
    case settings

    var id: String { rawValue }
}

/// Root observable model. Bridges the core actors to the UI.
@Observable
@MainActor
final class AppState {
    var servers: [Server] = []
    var killTarget: Server?
    var activeSheet: ActiveSheet?
    var logsServer: Server?

    var healthResults: [HealthResult] = []
    var healthLastChecked: Date?
    var activityEvents: [ActivityEvent] = []
    var logLines: [LogLine] = []

    @ObservationIgnored let monitor: ProcessMonitor
    @ObservationIgnored let health: HealthChecker
    @ObservationIgnored let logs: LogCapture
    @ObservationIgnored let activity: ActivityStore?

    init(
        monitor: ProcessMonitor = ProcessMonitor(),
        health: HealthChecker = HealthChecker(),
        logs: LogCapture = LogCapture(),
        activity: ActivityStore? = nil
    ) {
        self.monitor = monitor
        self.health = health
        self.logs = logs
        self.activity = activity ?? Self.defaultActivityStore()
        Task {
            await monitor.start()
            await consumeEvents()
        }
    }

    static func supportDirectory() -> String {
        let base = NSSearchPathForDirectoriesInDomains(.applicationSupportDirectory, .userDomainMask, true).first
            ?? NSHomeDirectory() + "/Library/Application Support"
        return base + "/BigServerMonitor"
    }

    private static func defaultActivityStore() -> ActivityStore? {
        try? ActivityStore(path: supportDirectory() + "/activity.db")
    }

    var activeCount: Int {
        servers.filter { $0.status == .online }.count
    }

    /// Servers grouped by project path for the PROJECTS section.
    var projectGroups: [(path: String, active: Int)] {
        var groups: [String: Int] = [:]
        for server in servers {
            guard let path = server.projectPath, !path.isEmpty else { continue }
            groups[path, default: 0] += server.status == .online ? 1 : 0
        }
        return groups.sorted { $0.key < $1.key }.map { (abbreviateHome($0.key), $0.value) }
    }

    func refresh() {
        Task {
            await monitor.pollOnce()
            await syncServers()
        }
    }

    func confirmKill() {
        guard let target = killTarget, let pid = target.pid else { return }
        killTarget = nil
        Task {
            try? await monitor.killProcess(pid: pid)
            try? await Task.sleep(for: .milliseconds(300))
            await monitor.pollOnce()
            await syncServers()
        }
    }

    // MARK: - Health

    func runHealthCheck() {
        let ports = servers.filter { $0.status == .online }.map(\.port)
        Task {
            healthResults = await health.runAll(ports: ports)
            healthLastChecked = Date()
        }
    }

    // MARK: - Activity

    func loadActivity() {
        guard let activity else { return }
        Task {
            activityEvents = (try? await activity.events()) ?? []
        }
    }

    func clearHistory() {
        guard let activity else { return }
        Task {
            try? await activity.clearHistory()
            activityEvents = []
        }
    }

    // MARK: - Logs

    func openLogs(for server: Server) {
        logsServer = server
        Task {
            logLines = await logs.lines(port: server.port)
        }
    }

    func copyLogs(_ lines: [LogLine]) {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss"
        let text = lines
            .map { "[\(formatter.string(from: $0.timestamp))] \($0.text)" }
            .joined(separator: "\n")
        setPasteboard(text)
    }

    func copyLogsForAI(server: Server) {
        Task {
            let export = await logs.aiExport(for: server)
            setPasteboard(export)
        }
    }

    private func setPasteboard(_ text: String) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
    }

    // MARK: - Internal

    private func consumeEvents() async {
        let stream = await monitor.events()
        await syncServers()
        for await event in stream {
            await recordActivity(event)
            await syncServers()
        }
    }

    private func recordActivity(_ event: ServerEvent) async {
        guard let activity else { return }
        let record: ActivityEvent
        switch event {
        case .started(let server):
            record = ActivityEvent(
                id: nil, type: .started, port: server.port,
                processName: server.processName, projectName: server.projectName,
                timestamp: Date(), durationSeconds: nil, exitCode: nil,
                message: "\(server.projectName ?? server.processName) started"
            )
        case .stopped(let server, let duration):
            record = ActivityEvent(
                id: nil, type: .stopped, port: server.port,
                processName: server.processName, projectName: server.projectName,
                timestamp: Date(), durationSeconds: duration, exitCode: nil,
                message: "\(server.projectName ?? server.processName) stopped"
            )
        }
        try? await activity.record(record)
    }

    private func syncServers() async {
        let snapshot = await monitor.servers
        servers = snapshot.values.sorted { $0.port < $1.port }
    }

    private func abbreviateHome(_ path: String) -> String {
        let home = NSHomeDirectory()
        return path.hasPrefix(home) ? "~" + path.dropFirst(home.count) : path
    }
}
