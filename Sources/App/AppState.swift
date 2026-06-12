import AppKit
import Observation
import ServiceManagement
import SwiftUI
import UserNotifications

/// Root observable model. Bridges the core actors to the UI.
@Observable
@MainActor
final class AppState {
    var servers: [Server] = []
    var killTarget: Server?
    var logsServer: Server?

    var healthResults: [HealthResult] = []
    var healthLastChecked: Date?
    var activityEvents: [ActivityEvent] = []
    var logLines: [LogLine] = []

    // Settings — mirrored from SettingsStore so the UI can observe them.
    var configPollingInterval: TimeInterval = 5.0
    var configHealthInterval: TimeInterval = 30.0
    var configIgnoredPorts: [Int] = []
    var configCrashAlerts: Bool = true
    var configShowBadge: Bool = true
    var configLaunchAtLogin: Bool = false

    @ObservationIgnored let monitor: ProcessMonitor
    @ObservationIgnored let health: HealthChecker
    @ObservationIgnored let logs: LogCapture
    @ObservationIgnored let activity: ActivityStore?
    @ObservationIgnored let settings: SettingsStore
    @ObservationIgnored let logger: JSONLogger
    @ObservationIgnored let notifier: Notifier

    init(
        monitor: ProcessMonitor = ProcessMonitor(),
        health: HealthChecker = HealthChecker(),
        logs: LogCapture = LogCapture(),
        activity: ActivityStore? = nil,
        settings: SettingsStore? = nil,
        logger: JSONLogger? = nil,
        notifier: Notifier = Notifier()
    ) {
        self.monitor = monitor
        self.health = health
        self.logs = logs
        self.activity = activity ?? Self.defaultActivityStore()
        self.settings = settings ?? Self.defaultSettingsStore()
        self.logger = logger ?? Self.defaultLogger()
        self.notifier = notifier
        Task {
            await self.logger.info("BigServerMonitor started", context: [:])
            await loadSettings()
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

    private static func defaultSettingsStore() -> SettingsStore {
        (try? SettingsStore(path: supportDirectory() + "/config.json")) ?? SettingsStore.fallback()
    }

    private static func defaultLogger() -> JSONLogger {
        (try? JSONLogger(path: supportDirectory() + "/bigservermonitor.log")) ?? JSONLogger.fallback()
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
        logLines = []
        Task {
            if let pid = server.pid {
                await logs.captureSystemLogs(pid: Int(pid), port: server.port)
            }
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

    // MARK: - Settings

    func loadSettings() async {
        try? await settings.load()
        let config = await settings.current
        configPollingInterval = config.pollingInterval
        configHealthInterval = config.healthInterval
        configIgnoredPorts = config.ignoredPorts
        configCrashAlerts = config.notifications.crashAlerts
        configShowBadge = config.notifications.showBadge
        configLaunchAtLogin = config.launchAtLogin
    }

    func saveSettings() async {
        let config = AppConfig(
            scanDirs: [],
            pollingInterval: configPollingInterval,
            healthInterval: configHealthInterval,
            ignoredPorts: configIgnoredPorts,
            notifications: NotificationConfig(crashAlerts: configCrashAlerts, showBadge: configShowBadge),
            launchAtLogin: configLaunchAtLogin
        )
        try? await settings.save(config)
    }

    func setLaunchAtLogin(_ enabled: Bool) async {
        do {
            if enabled {
                try SMAppService.mainApp.register()
            } else {
                try await SMAppService.mainApp.unregister()
            }
        } catch {
            // SMAppService may throw if not permitted; log and continue
        }
    }

    // MARK: - Internal

    private func consumeEvents() async {
        let stream = await monitor.events()
        await syncServers()
        await requestNotificationPermission()
        for await event in stream {
            await logger.info("Server event", context: eventContext(event))
            await recordActivity(event)
            await handleCrashNotification(event)
            await syncServers()
        }
    }

    private func eventContext(_ event: ServerEvent) -> [String: String] {
        switch event {
        case .started(let server):
            return ["type": "started", "port": String(server.port), "process": server.processName]
        case .stopped(let server, let duration):
            return ["type": "stopped", "port": String(server.port), "process": server.processName, "duration": String(Int(duration))]
        }
    }

    private func handleCrashNotification(_ event: ServerEvent) async {
        guard case .stopped(let server, let duration) = event else { return }
        guard configCrashAlerts else { return }
        guard await notifier.shouldNotify(port: server.port) else { return }

        let message = formatCrashMessage(for: server, duration: duration)
        let content = UNMutableNotificationContent()
        content.title = "Server stopped"
        content.body = message
        content.sound = .default

        let request = UNNotificationRequest(
            identifier: "crash-\(server.port)-\(Date().timeIntervalSince1970)",
            content: content,
            trigger: nil
        )
        do {
            try await UNUserNotificationCenter.current().add(request)
            await notifier.recordNotification(port: server.port)
        } catch {
            await logger.error("Failed to deliver notification", context: ["port": String(server.port)])
        }
    }

    private func requestNotificationPermission() async {
        do {
            let granted = try await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound])
            if !granted {
                await logger.warn("Notification permission denied", context: [:])
            }
        } catch {
            await logger.error("Notification permission request failed", context: [:])
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
