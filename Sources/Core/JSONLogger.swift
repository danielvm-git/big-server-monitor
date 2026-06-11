import Foundation

/// Writes structured JSON log lines to a file. Each line is a complete JSON object.
/// Follows the observability contract from CLAUDE.md: time, level, msg, context.
actor JSONLogger {
    private let logURL: URL
    private let formatter: ISO8601DateFormatter
    private let encoder: JSONEncoder

    init(path: String) throws {
        logURL = URL(fileURLWithPath: path)
        formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        encoder = JSONEncoder()
        encoder.outputFormatting = .sortedKeys

        // Ensure parent directory exists
        let parent = logURL.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: parent, withIntermediateDirectories: true)
    }

    func debug(_ msg: String, context: [String: String] = [:]) async {
        await log(level: .debug, msg: msg, context: context)
    }

    func info(_ msg: String, context: [String: String] = [:]) async {
        await log(level: .info, msg: msg, context: context)
    }

    func warn(_ msg: String, context: [String: String] = [:]) async {
        await log(level: .warn, msg: msg, context: context)
    }

    func error(_ msg: String, context: [String: String] = [:]) async {
        await log(level: .error, msg: msg, context: context)
    }

    private func log(level: LogLevel, msg: String, context: [String: String]) async {
        var entry: [String: Any] = [
            "time": formatter.string(from: Date()),
            "level": level.rawValue.uppercased(),
            "msg": msg
        ]
        for (key, value) in context {
            entry[key] = value
        }

        guard let data = try? JSONSerialization.data(withJSONObject: entry, options: .sortedKeys) else { return }
        var line = String(data: data, encoding: .utf8) ?? ""
        line += "\n"
        guard let lineData = line.data(using: .utf8) else { return }

        if let handle = try? FileHandle(forWritingTo: logURL) {
            _ = try? handle.seekToEndCompat()
            try? handle.write(contentsOf: lineData)
            try? handle.close()
        } else {
            // File doesn't exist yet — create it
            try? lineData.write(to: logURL, options: .atomic)
        }
    }
}

// Polyfill: seekToEnd() is not available on all platforms in the same way
private extension FileHandle {
    func seekToEndCompat() throws {
        if #available(macOS 10.15.4, *) {
            try seekToEnd()
        } else {
            seekToEndOfFile()
        }
    }
}
