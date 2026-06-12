import Foundation

/// Rate-limited crash notification manager.
/// Tracks per-port notification timestamps to enforce a minimum interval between alerts.
actor Notifier {
    private let rateLimitSeconds: TimeInterval
    private var lastNotified: [Int: Date] = [:]

    init(rateLimitSeconds: TimeInterval = 60) {
        self.rateLimitSeconds = rateLimitSeconds
    }

    /// Returns true if a notification should be sent for this port (rate limit not exceeded).
    func shouldNotify(port: Int) -> Bool {
        guard let last = lastNotified[port] else { return true }
        return Date().timeIntervalSince(last) >= rateLimitSeconds
    }

    /// Records that a notification was sent for the given port.
    func recordNotification(port: Int, timestamp: Date = Date()) {
        lastNotified[port] = timestamp
    }
}

/// Formats a human-readable crash notification message.
func formatCrashMessage(for server: Server, duration: TimeInterval) -> String {
    let name = server.projectName ?? server.processName
    let uptimeStr = formatUptimeCompact(seconds: Int(duration))

    if let project = server.projectName {
        return "\(project) (\(server.processName)) on :\(server.port) stopped after \(uptimeStr)"
    } else {
        return "\(server.processName) on :\(server.port) stopped after \(uptimeStr)"
    }
}

/// Compact uptime format: "2h 03m" or "45s"
private func formatUptimeCompact(seconds: Int) -> String {
    let h = seconds / 3600
    let m = (seconds % 3600) / 60
    let s = seconds % 60
    if h > 0 {
        return "\(h)h \(String(format: "%02d", m))m"
    } else if m > 0 {
        return "\(m)m"
    } else {
        return "\(s)s"
    }
}
