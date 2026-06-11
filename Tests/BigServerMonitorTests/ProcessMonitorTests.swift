import Foundation
import Testing
@testable import BigServerMonitor

/// Thread-safe scriptable discovery double.
final class MockDiscovery: PortDiscovering, @unchecked Sendable {
    private let lock = NSLock()
    private var _ports: [Int]
    private var _infos: [Int: ProcessInfo_]

    init(ports: [Int] = [], infos: [Int: ProcessInfo_] = [:]) {
        _ports = ports
        _infos = infos
    }

    func set(ports: [Int], infos: [Int: ProcessInfo_]) {
        lock.lock(); defer { lock.unlock() }
        _ports = ports
        _infos = infos
    }

    func listeningPorts() throws -> [Int] {
        lock.lock(); defer { lock.unlock() }
        return _ports
    }

    func processInfo(port: Int) throws -> ProcessInfo_ {
        lock.lock(); defer { lock.unlock() }
        guard let info = _infos[port] else { throw DiscoveryError.noPID(port: port) }
        return info
    }
}

private func info(pid: Int32, name: String) -> ProcessInfo_ {
    ProcessInfo_(pid: pid, processName: name, binaryPath: "/usr/local/bin/\(name)",
                 workingDir: "", memoryMB: 42, startedAt: Date(timeIntervalSinceNow: -60))
}

@Suite struct ProcessMonitorTests {
    @Test func discoversServersOnPoll() async throws {
        let mock = MockDiscovery(ports: [3000, 5173], infos: [
            3000: info(pid: 1024, name: "node"),
            5173: info(pid: 2048, name: "vite"),
        ])
        let monitor = ProcessMonitor(discovery: mock, ignoredPorts: [])

        await monitor.pollOnce()
        let servers = await monitor.servers

        #expect(servers.count == 2)
        #expect(servers[3000]?.processName == "node")
        #expect(servers[3000]?.status == .online)
    }

    @Test func ignoredPortsAreSkipped() async throws {
        let mock = MockDiscovery(ports: [443, 3000], infos: [
            443: info(pid: 1, name: "https"),
            3000: info(pid: 1024, name: "node"),
        ])
        let monitor = ProcessMonitor(discovery: mock, ignoredPorts: [443])

        await monitor.pollOnce()
        let servers = await monitor.servers

        #expect(servers.keys.sorted() == [3000])
    }

    @Test func emitsStartedAndStoppedEvents() async throws {
        let mock = MockDiscovery(ports: [3000], infos: [3000: info(pid: 1024, name: "node")])
        let monitor = ProcessMonitor(discovery: mock, ignoredPorts: [])
        let stream = await monitor.events()
        var iterator = stream.makeAsyncIterator()

        await monitor.pollOnce()
        let first = await iterator.next()
        guard case .started(let started) = first else {
            Issue.record("expected .started, got \(String(describing: first))")
            return
        }
        #expect(started.port == 3000)

        mock.set(ports: [], infos: [:])
        await monitor.pollOnce()
        let second = await iterator.next()
        guard case .stopped(let stopped, let duration) = second else {
            Issue.record("expected .stopped, got \(String(describing: second))")
            return
        }
        #expect(stopped.port == 3000)
        #expect(stopped.status == .offline)
        #expect(duration > 0)
    }

    @Test func transientEmptyResultKeepsKnownServers() async throws {
        // Note: parity with Go — an all-empty poll after discovery keeps state...
        let mock = MockDiscovery(ports: [3000], infos: [3000: info(pid: 1024, name: "node")])
        let monitor = ProcessMonitor(discovery: mock, ignoredPorts: [])
        await monitor.pollOnce()

        mock.set(ports: [3000], infos: [:]) // lsof listed it but enrich failed
        await monitor.pollOnce()
        let servers = await monitor.servers
        #expect(servers[3000] != nil)
    }

    @Test func killProcessKillsSpawnedProcess() async throws {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/sleep")
        task.arguments = ["30"]
        try task.run()
        let pid = task.processIdentifier

        let monitor = ProcessMonitor(discovery: MockDiscovery(), ignoredPorts: [])
        try await monitor.killProcess(pid: pid)
        task.waitUntilExit()

        #expect(task.terminationReason == .uncaughtSignal)
    }

    @Test func killProcessThrowsForBogusPID() async throws {
        let monitor = ProcessMonitor(discovery: MockDiscovery(), ignoredPorts: [])
        await #expect(throws: KillError.self) {
            try await monitor.killProcess(pid: 99_999_999)
        }
    }
}
