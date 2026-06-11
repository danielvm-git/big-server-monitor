import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct ActivityStoreTests {
    private func event(
        type: ActivityEventType = .started, port: Int = 3000,
        timestamp: Date = Date()
    ) -> ActivityEvent {
        ActivityEvent(
            id: nil, type: type, port: port, processName: "node",
            projectName: "bigbase-api", timestamp: timestamp,
            durationSeconds: nil, exitCode: nil, message: nil
        )
    }

    @Test func recordsAndFetchesEvents() async throws {
        let store = try ActivityStore.inMemory()
        try await store.record(event(type: .started))
        try await store.record(event(type: .stopped, port: 5173))

        let all = try await store.events()
        #expect(all.count == 2)
        #expect(all.first?.processName == "node")
    }

    @Test func filtersByPortAndType() async throws {
        let store = try ActivityStore.inMemory()
        try await store.record(event(type: .started, port: 3000))
        try await store.record(event(type: .crashed, port: 8080))
        try await store.record(event(type: .stopped, port: 3000))

        let port3000 = try await store.events(filter: ActivityFilter(port: 3000))
        #expect(port3000.count == 2)

        let crashes = try await store.events(filter: ActivityFilter(types: [.crashed]))
        #expect(crashes.count == 1)
        #expect(crashes[0].port == 8080)
    }

    @Test func countsByType() async throws {
        let store = try ActivityStore.inMemory()
        try await store.record(event(type: .started))
        try await store.record(event(type: .started, port: 5173))
        try await store.record(event(type: .crashed, port: 8080))

        let counts = try await store.eventCounts()
        #expect(counts[.started] == 2)
        #expect(counts[.crashed] == 1)
        #expect(counts[.stopped] == nil)
    }

    @Test func clearHistoryDeletesAll() async throws {
        let store = try ActivityStore.inMemory()
        try await store.record(event())
        try await store.clearHistory()

        #expect(try await store.events().isEmpty)
    }

    @Test func purgeRemovesExpiredEvents() async throws {
        let store = try ActivityStore.inMemory()
        try await store.record(event(timestamp: Date(timeIntervalSinceNow: -40 * 86400)))
        try await store.record(event(timestamp: Date()))

        try await store.purgeExpired()
        let remaining = try await store.events()
        #expect(remaining.count == 1)
    }

    @Test func persistsToDiskAcrossInstances() async throws {
        let path = FileManager.default.temporaryDirectory
            .appendingPathComponent("bsm-activity-\(UUID().uuidString).db").path
        defer { try? FileManager.default.removeItem(atPath: path) }

        do {
            let store = try ActivityStore(path: path)
            try await store.record(event())
        }
        let reopened = try ActivityStore(path: path)
        #expect(try await reopened.events().count == 1)
    }
}
