import Foundation
import Testing
@testable import BigServerMonitor

@Suite @MainActor struct AppStateTests {
    private func makeState() -> (AppState, MockDiscovery) {
        let mock = MockDiscovery(ports: [3000, 5173], infos: [
            3000: ProcessInfo_(pid: 1024, processName: "node", binaryPath: "/usr/local/bin/node",
                               workingDir: NSHomeDirectory() + "/projects/bigbase",
                               memoryMB: 124, startedAt: Date(timeIntervalSinceNow: -7380)),
            5173: ProcessInfo_(pid: 2048, processName: "vite", binaryPath: "/usr/local/bin/vite",
                               workingDir: NSHomeDirectory() + "/projects/bigbase",
                               memoryMB: 89, startedAt: Date(timeIntervalSinceNow: -2700)),
        ])
        return (AppState(monitor: ProcessMonitor(discovery: mock, ignoredPorts: [])), mock)
    }

    @Test func serversSortedByPortAfterPoll() async throws {
        let (state, _) = makeState()
        await state.monitor.pollOnce()
        state.refresh()
        try await Task.sleep(for: .milliseconds(200))

        #expect(state.servers.map(\.port) == [3000, 5173])
        #expect(state.activeCount == 2)
    }

    @Test func projectGroupsAbbreviateHomeAndCountActive() async throws {
        let (state, _) = makeState()
        state.refresh()
        try await Task.sleep(for: .milliseconds(200))

        #expect(state.projectGroups.count == 1)
        #expect(state.projectGroups.first?.path.hasPrefix("~/") == true)
        #expect(state.projectGroups.first?.active == 2)
    }
}
