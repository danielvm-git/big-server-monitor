import Foundation

enum LogLevel: String, Codable, Sendable {
    case info
    case warn
    case error
}

struct LogLine: Sendable, Equatable, Identifiable {
    let id: Int // sequence number
    let timestamp: Date
    let level: LogLevel
    let text: String
    let stream: String // stdout, stderr, system
}

private let errorPatterns: [NSRegularExpression] = [
    // \w* prefix catches camelCase exception names like OperationalError.
    try! NSRegularExpression(pattern: #"\b(\w*error|err|\w*exception|traceback|panic|fatal|failed)\b"#, options: [.caseInsensitive]),
    try! NSRegularExpression(pattern: #"^\s*at \w+\.\w+"#),
    try! NSRegularExpression(pattern: #"exit code [^0]"#, options: [.caseInsensitive]),
]

private let warnPatterns: [NSRegularExpression] = [
    try! NSRegularExpression(pattern: #"\b(warn|warning|deprecated|caution)\b"#, options: [.caseInsensitive]),
]

/// Classifies a log line's severity by keyword patterns (Go parity).
func classifyLogLine(_ text: String) -> LogLevel {
    let range = NSRange(text.startIndex..., in: text)
    for pattern in errorPatterns where pattern.firstMatch(in: text, range: range) != nil {
        return .error
    }
    for pattern in warnPatterns where pattern.firstMatch(in: text, range: range) != nil {
        return .warn
    }
    return .info
}

/// In-memory per-port ring buffers of captured log lines (500/port max),
/// with the "Copy for AI agent" export.
actor LogCapture {
    static let maxLines = 500

    private var buffers: [Int: [LogLine]] = [:]
    private var sequence = 0

    func ingest(port: Int, text: String, stream: String = "stdout", timestamp: Date = Date()) {
        sequence += 1
        let line = LogLine(
            id: sequence, timestamp: timestamp,
            level: classifyLogLine(text), text: text, stream: stream
        )
        var buffer = buffers[port] ?? []
        buffer.append(line)
        if buffer.count > Self.maxLines {
            buffer.removeFirst(buffer.count - Self.maxLines)
        }
        buffers[port] = buffer
    }

    func clear(port: Int) {
        buffers[port] = nil
    }

    func lines(port: Int, level: LogLevel? = nil) -> [LogLine] {
        let all = buffers[port] ?? []
        guard let level else { return all }
        return all.filter { $0.level == level }
    }

    func counts(port: Int) -> [LogLevel: Int] {
        var counts: [LogLevel: Int] = [:]
        for line in buffers[port] ?? [] {
            counts[line.level, default: 0] += 1
        }
        return counts
    }

    /// Formatted context block for pasting into an AI agent, mirroring the Go
    /// GetLogsForAI output (rebranded header).
    func aiExport(for server: Server) -> String {
        let lines = buffers[server.port] ?? []
        guard !lines.isEmpty else { return "" }

        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss"

        let normal = lines.filter { $0.level == .info }
        let errWarn = lines.filter { $0.level != .info }

        var out = "=== BigServerMonitor Log Export ===\n"
        out += "Server:  \(server.projectName ?? "—")\n"
        out += "Process: \(server.processName)  (PID \(server.pid.map(String.init) ?? "n/a"))\n"
        out += "Port:    :\(server.port)\n"
        out += "Memory:  \(Int(server.memoryMB ?? 0)) MB   Uptime: \(server.uptime)\n"
        out += "Binary:  \(server.binaryPath ?? "—")\n"
        out += "\n"

        out += "--- stdout / stderr (\(normal.count) lines) ---\n"
        for line in normal.suffix(30) {
            out += "[\(formatter.string(from: line.timestamp))] [\(server.processName)] \(line.text)\n"
        }
        out += "\n"

        out += "--- Errors & warnings (\(errWarn.count) lines) ---\n"
        for line in errWarn {
            out += "[\(formatter.string(from: line.timestamp))] \(line.text)\n"
        }
        return out
    }
}
