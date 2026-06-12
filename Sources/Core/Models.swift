import Foundation

enum ServerStatus: String, Codable, Sendable {
    case online
    case offline
    case unknown
}

/// A development server listening on a local TCP port.
struct Server: Identifiable, Codable, Sendable, Equatable {
    var id: Int { port }

    let port: Int
    var processName: String
    var pid: Int32?
    var status: ServerStatus
    var projectName: String?
    var projectPath: String?
    var binaryPath: String?
    var memoryMB: Double?
    var startedAt: Date?

    var uptime: String {
        guard let startedAt, status == .online else { return "—" }
        let seconds = Int(Date().timeIntervalSince(startedAt))
        let h = seconds / 3600, m = (seconds % 3600) / 60
        return h > 0 ? "\(h)h \(String(format: "%02d", m))m" : "\(m)m"
    }
}
