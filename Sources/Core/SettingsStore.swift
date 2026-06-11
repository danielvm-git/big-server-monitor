import Foundation

struct NotificationConfig: Codable, Sendable, Equatable {
    var crashAlerts: Bool = true
    var showBadge: Bool = true
}

/// Application settings persisted as JSON in Application Support.
struct AppConfig: Codable, Sendable, Equatable {
    var scanDirs: [String] = [NSHomeDirectory() + "/Developer"]
    var pollingInterval: TimeInterval = 5.0
    var healthInterval: TimeInterval = 30.0
    var ignoredPorts: [Int] = []
    var notifications: NotificationConfig = NotificationConfig()
    var launchAtLogin: Bool = false
}

/// Loads, saves, and resets application settings from a JSON file.
actor SettingsStore {
    private let configURL: URL
    private(set) var current: AppConfig

    init(path: String) throws {
        configURL = URL(fileURLWithPath: path)
        current = AppConfig()
    }

    /// Creates an in-memory fallback store when the disk path is unavailable.
    static func fallback() -> SettingsStore {
        try! SettingsStore(path: NSTemporaryDirectory() + "bigservermonitor-fallback-config.json")
    }

    /// Loads config from disk. If the file is missing or corrupt, keeps defaults.
    func load() throws {
        guard FileManager.default.fileExists(atPath: configURL.path) else { return }
        let data = try Data(contentsOf: configURL)
        let decoder = JSONDecoder()
        current = try decoder.decode(AppConfig.self, from: data)
    }

    /// Writes current config to disk. Creates parent directories if needed.
    func save(_ config: AppConfig? = nil) throws {
        if let config { current = config }
        let parent = configURL.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: parent, withIntermediateDirectories: true)
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        let data = try encoder.encode(current)
        try data.write(to: configURL, options: .atomic)
    }

    /// Restores all settings to their default values.
    func reset() {
        current = AppConfig()
    }
}
