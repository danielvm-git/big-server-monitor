import Foundation

/// Metadata for a process listening on a port.
struct ProcessInfo_: Sendable, Equatable {
    var pid: Int32
    var processName: String
    var binaryPath: String
    var workingDir: String
    var memoryMB: Double
    var startedAt: Date
}

/// Abstracts OS-level port and process discovery so the monitor is testable.
protocol PortDiscovering: Sendable {
    func listeningPorts() throws -> [Int]
    func processInfo(port: Int) throws -> ProcessInfo_
}

enum DiscoveryError: Error {
    case commandFailed(String)
    case noPID(port: Int)
}

/// lsof-based discovery with netstat fallback, mirroring the Go
/// LsofPortDiscovery behavior.
struct LsofPortDiscovery: PortDiscovering {
    func listeningPorts() throws -> [Int] {
        do {
            let out = try run("/usr/sbin/lsof", ["-iTCP", "-sTCP:LISTEN", "-n", "-P"])
            return parseLsofOutput(out)
        } catch {
            let out = try run("/usr/sbin/netstat", ["-an", "-p", "tcp"])
            return parseNetstatOutput(out)
        }
    }

    func processInfo(port: Int) throws -> ProcessInfo_ {
        let out = try run("/usr/sbin/lsof", ["-iTCP:\(port)", "-sTCP:LISTEN", "-n", "-P"])
        guard let pid = firstPID(inLsofOutput: out) else {
            throw DiscoveryError.noPID(port: port)
        }

        let comm = (try? run("/bin/ps", ["-p", "\(pid)", "-o", "comm="])) ?? ""
        let processName = (comm.split(separator: "/").last.map(String.init) ?? comm)
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let binaryPath = comm.trimmingCharacters(in: .whitespacesAndNewlines)

        let cwdOut = (try? run("/usr/sbin/lsof", ["-p", "\(pid)", "-a", "-d", "cwd", "-Fn"])) ?? ""
        let workingDir = cwdOut.split(separator: "\n")
            .first { $0.hasPrefix("n") }
            .map { String($0.dropFirst()) } ?? ""

        let rss = (try? run("/bin/ps", ["-p", "\(pid)", "-o", "rss="])) ?? "0"
        let memoryMB = (Double(rss.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0) / 1024

        let etime = (try? run("/bin/ps", ["-p", "\(pid)", "-o", "etime="])) ?? ""
        let elapsed = parseEtime(etime.trimmingCharacters(in: .whitespacesAndNewlines))
        let startedAt = Date(timeIntervalSinceNow: -elapsed)

        return ProcessInfo_(
            pid: pid, processName: processName, binaryPath: binaryPath,
            workingDir: workingDir, memoryMB: memoryMB, startedAt: startedAt
        )
    }

    private func run(_ launchPath: String, _ args: [String]) throws -> String {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: launchPath)
        task.arguments = args
        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = Pipe()
        try task.run()
        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        task.waitUntilExit()
        guard task.terminationStatus == 0 else {
            throw DiscoveryError.commandFailed("\(launchPath) exited \(task.terminationStatus)")
        }
        return String(decoding: data, as: UTF8.self)
    }
}

// MARK: - Pure parsing functions (ported from portdiscovery.go)

/// Extracts the port from an address like `*:8080` or `127.0.0.1:3000`.
func extractPort(_ addr: String) -> Int {
    guard let last = addr.split(separator: ":").last, addr.contains(":") else { return 0 }
    return Int(last) ?? 0
}

/// Extracts unique listening port numbers from `lsof -iTCP -sTCP:LISTEN` output.
func parseLsofOutput(_ output: String) -> [Int] {
    var ports = Set<Int>()
    for line in output.split(separator: "\n") {
        let fields = line.split(separator: " ", omittingEmptySubsequences: true)
        guard fields.count >= 9 else { continue }
        var idx = fields.count - 1
        if fields[idx] == "(LISTEN)", idx > 0 { idx -= 1 }
        let port = extractPort(String(fields[idx]))
        if port > 0 { ports.insert(port) }
    }
    return ports.sorted()
}

/// Extracts unique listening ports from `netstat -an -p tcp` output.
/// macOS netstat uses `.` before the port (e.g. `127.0.0.1.3000`).
func parseNetstatOutput(_ output: String) -> [Int] {
    var ports = Set<Int>()
    for line in output.split(separator: "\n") where line.contains("LISTEN") {
        let fields = line.split(separator: " ", omittingEmptySubsequences: true)
        guard fields.count >= 4 else { continue }
        let addr = String(fields[3])
        let normalized = addr.contains(":") ? addr : normalizeDotAddress(addr)
        let port = extractPort(normalized)
        if port > 0 { ports.insert(port) }
    }
    return ports.sorted()
}

private func normalizeDotAddress(_ addr: String) -> String {
    guard let lastDot = addr.lastIndex(of: ".") else { return addr }
    return addr[..<lastDot] + ":" + addr[addr.index(after: lastDot)...]
}

/// First PID column value in lsof output (skips the header row).
func firstPID(inLsofOutput output: String) -> Int32? {
    for line in output.split(separator: "\n") {
        let fields = line.split(separator: " ", omittingEmptySubsequences: true)
        guard fields.count >= 2, let pid = Int32(fields[1]), pid > 0 else { continue }
        return pid
    }
    return nil
}

/// Parses ps etime format ([[dd-]hh:]mm:ss) into seconds.
func parseEtime(_ etime: String) -> TimeInterval {
    var days = 0.0
    var rest = etime
    if let dash = etime.firstIndex(of: "-") {
        days = Double(etime[..<dash]) ?? 0
        rest = String(etime[etime.index(after: dash)...])
    }
    let parts = rest.split(separator: ":").map { Double($0) ?? 0 }
    let hms: Double
    switch parts.count {
    case 3: hms = parts[0] * 3600 + parts[1] * 60 + parts[2]
    case 2: hms = parts[0] * 60 + parts[1]
    case 1: hms = parts[0]
    default: hms = 0
    }
    return days * 86400 + hms
}
