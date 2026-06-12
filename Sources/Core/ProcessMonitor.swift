import Foundation

enum ServerEvent: Sendable, Equatable {
    case started(Server)
    case stopped(Server, duration: TimeInterval)
}

enum KillError: Error {
    case failed(pid: Int32, errno: Int32)
}

/// Polls listening TCP ports, diffs against the previous snapshot, and yields
/// started/stopped events. Swift port of the Go ProcessMonitor component.
actor ProcessMonitor {
    private let discovery: any PortDiscovering
    private var pollInterval: TimeInterval
    private var ignoredPorts: Set<Int>

    private(set) var servers: [Int: Server] = [:]
    private var pollTask: Task<Void, Never>?

    private var continuations: [UUID: AsyncStream<ServerEvent>.Continuation] = [:]

    init(
        discovery: any PortDiscovering = LsofPortDiscovery(),
        pollInterval: TimeInterval = 5,
        ignoredPorts: Set<Int> = [80, 443]
    ) {
        self.discovery = discovery
        self.pollInterval = pollInterval
        self.ignoredPorts = ignoredPorts
    }

    /// Subscribe to server lifecycle events.
    func events() -> AsyncStream<ServerEvent> {
        AsyncStream { continuation in
            let id = UUID()
            continuations[id] = continuation
            continuation.onTermination = { [weak self] _ in
                Task { await self?.removeContinuation(id) }
            }
        }
    }

    private func removeContinuation(_ id: UUID) {
        continuations[id] = nil
    }

    func updateConfig(pollInterval: TimeInterval, ignoredPorts: Set<Int>) {
        self.pollInterval = pollInterval
        self.ignoredPorts = ignoredPorts
    }

    func start() {
        guard pollTask == nil else { return }
        pollTask = Task {
            while !Task.isCancelled {
                pollOnce()
                try? await Task.sleep(for: .seconds(pollInterval))
            }
        }
    }

    func stop() {
        pollTask?.cancel()
        pollTask = nil
    }

    /// One discovery pass: list ports, enrich, diff, emit. Exposed for tests
    /// and the manual refresh button.
    func pollOnce() {
        guard let ports = try? discovery.listeningPorts() else { return }

        var current: [Int: Server] = [:]
        for port in ports where !ignoredPorts.contains(port) {
            guard let info = try? discovery.processInfo(port: port) else { continue }
            current[port] = Server(
                port: port,
                processName: info.processName,
                pid: info.pid,
                status: .online,
                projectName: detectProjectName(workDir: info.workingDir),
                projectPath: info.workingDir,
                binaryPath: info.binaryPath,
                memoryMB: info.memoryMB,
                startedAt: info.startedAt
            )
        }

        diffAndEmit(current)

        // A transient empty result must not wipe known servers (Go parity).
        if !current.isEmpty || servers.isEmpty {
            servers = current
        }
    }

    private func diffAndEmit(_ current: [Int: Server]) {
        for (port, server) in current where servers[port] == nil {
            emit(.started(server))
        }
        for (port, previous) in servers where current[port] == nil {
            let duration = previous.startedAt.map { Date().timeIntervalSince($0) } ?? 0
            var stopped = previous
            stopped.status = .offline
            stopped.pid = nil
            emit(.stopped(stopped, duration: duration))
        }
    }

    private func emit(_ event: ServerEvent) {
        for continuation in continuations.values {
            continuation.yield(event)
        }
    }

    /// SIGKILL the given PID.
    func killProcess(pid: Int32) throws {
        guard kill(pid, SIGKILL) == 0 else {
            throw KillError.failed(pid: pid, errno: errno)
        }
    }
}
