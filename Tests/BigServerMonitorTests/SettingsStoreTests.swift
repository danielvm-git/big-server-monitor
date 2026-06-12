import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct SettingsStoreTests {

    @Test func defaultConfigHasExpectedDefaults() {
        let config = AppConfig()
        #expect(config.scanDirs.first == NSHomeDirectory() + "/Developer")
        #expect(config.pollingInterval == 5.0)
        #expect(config.healthInterval == 30.0)
        #expect(config.ignoredPorts.isEmpty)
        #expect(config.notifications.crashAlerts == true)
        #expect(config.notifications.showBadge == true)
        #expect(config.launchAtLogin == false)
    }

    @Test func saveAndLoadRoundtrips() async throws {
        let dir = temporaryDirectory()
        let store = try SettingsStore(path: dir + "/config.json")
        var config = await store.current
        config.pollingInterval = 10.0
        config.ignoredPorts = [8080, 3000]
        config.notifications.crashAlerts = false
        try await store.save(config)

        let store2 = try SettingsStore(path: dir + "/config.json")
        try await store2.load()
        let loaded = await store2.current
        #expect(loaded.pollingInterval == 10.0)
        #expect(loaded.ignoredPorts == [8080, 3000])
        #expect(loaded.notifications.crashAlerts == false)
    }

    @Test func loadCreatesDefaultWhenFileMissing() async throws {
        let dir = temporaryDirectory()
        let store = try SettingsStore(path: dir + "/nonexistent.json")
        try await store.load()
        let config = await store.current
        #expect(config.scanDirs.first == NSHomeDirectory() + "/Developer")
    }

    @Test func resetRestoresDefaults() async throws {
        let dir = temporaryDirectory()
        let store = try SettingsStore(path: dir + "/config.json")
        var config = await store.current
        config.pollingInterval = 99.0
        try await store.save(config)
        await store.reset()
        let resetConfig = await store.current
        #expect(resetConfig.pollingInterval == 5.0)
        #expect(resetConfig.ignoredPorts.isEmpty)
    }

    @Test func saveCreatesParentDirectory() async throws {
        let dir = temporaryDirectory() + "/nested/sub"
        let store = try SettingsStore(path: dir + "/config.json")
        try await store.save(await store.current)

        var isDir: ObjCBool = false
        #expect(FileManager.default.fileExists(atPath: dir, isDirectory: &isDir))
        #expect(isDir.boolValue)
        #expect(FileManager.default.fileExists(atPath: dir + "/config.json"))
    }

    // MARK: - Helpers

    private func temporaryDirectory() -> String {
        let dir = NSTemporaryDirectory() + "SettingsStoreTests-\(UUID().uuidString)"
        try? FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        return dir
    }
}
