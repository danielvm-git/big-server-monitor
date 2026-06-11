import Observation
import SwiftUI

/// Root observable model. Bridges the core actors to the UI.
@Observable
@MainActor
final class AppState {
    var servers: [Server] = []
    var killTarget: Server?

    @ObservationIgnored let monitor: ProcessMonitor

    init(monitor: ProcessMonitor = ProcessMonitor()) {
        self.monitor = monitor
        Task {
            await monitor.start()
            await consumeEvents()
        }
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

    private func consumeEvents() async {
        let stream = await monitor.events()
        await syncServers()
        for await _ in stream {
            await syncServers()
        }
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
