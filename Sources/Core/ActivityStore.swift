import Foundation
import GRDB

enum ActivityEventType: String, Codable, Sendable, CaseIterable {
    case started
    case stopped
    case crashed
    case unresponsive
}

struct ActivityEvent: Codable, Sendable, Equatable, Identifiable, FetchableRecord, MutablePersistableRecord {
    static let databaseTableName = "events"

    var id: Int64?
    var type: ActivityEventType
    var port: Int
    var processName: String
    var projectName: String?
    var timestamp: Date
    var durationSeconds: Double?
    var exitCode: Int?
    var message: String?

    mutating func didInsert(_ inserted: InsertionSuccess) {
        id = inserted.rowID
    }
}

struct ActivityFilter: Sendable {
    var port: Int?
    var types: [ActivityEventType]?
    var since: Date?
    var limit: Int = 200
}

/// SQLite-backed activity history (GRDB), 30-day retention.
actor ActivityStore {
    private let dbQueue: DatabaseQueue
    private let retentionDays: Int

    init(path: String, retentionDays: Int = 30) throws {
        self.retentionDays = retentionDays
        try FileManager.default.createDirectory(
            atPath: (path as NSString).deletingLastPathComponent,
            withIntermediateDirectories: true
        )
        dbQueue = try DatabaseQueue(path: path)
        try dbQueue.write { db in
            try db.create(table: "events", ifNotExists: true) { t in
                t.autoIncrementedPrimaryKey("id")
                t.column("type", .text).notNull()
                t.column("port", .integer).notNull()
                t.column("processName", .text).notNull()
                t.column("projectName", .text)
                t.column("timestamp", .datetime).notNull()
                t.column("durationSeconds", .double)
                t.column("exitCode", .integer)
                t.column("message", .text)
            }
            try db.create(index: "idx_events_timestamp", on: "events", columns: ["timestamp"], ifNotExists: true)
        }
        let cutoff = Date().addingTimeInterval(-Double(retentionDays) * 86400)
        _ = try dbQueue.write { db in
            try ActivityEvent.filter(Column("timestamp") < cutoff).deleteAll(db)
        }
    }

    /// In-memory store for tests.
    static func inMemory() throws -> ActivityStore {
        try ActivityStore(inMemoryRetentionDays: 30)
    }

    private init(inMemoryRetentionDays: Int) throws {
        retentionDays = inMemoryRetentionDays
        dbQueue = try DatabaseQueue()
        try dbQueue.write { db in
            try db.create(table: "events", ifNotExists: true) { t in
                t.autoIncrementedPrimaryKey("id")
                t.column("type", .text).notNull()
                t.column("port", .integer).notNull()
                t.column("processName", .text).notNull()
                t.column("projectName", .text)
                t.column("timestamp", .datetime).notNull()
                t.column("durationSeconds", .double)
                t.column("exitCode", .integer)
                t.column("message", .text)
            }
        }
    }

    func record(_ event: ActivityEvent) throws {
        var event = event
        try dbQueue.write { db in
            try event.insert(db)
        }
    }

    func events(filter: ActivityFilter = ActivityFilter()) throws -> [ActivityEvent] {
        try dbQueue.read { db in
            var request = ActivityEvent.order(Column("timestamp").desc)
            if let port = filter.port {
                request = request.filter(Column("port") == port)
            }
            if let types = filter.types, !types.isEmpty {
                request = request.filter(types.map(\.rawValue).contains(Column("type")))
            }
            if let since = filter.since {
                request = request.filter(Column("timestamp") >= since)
            }
            return try request.limit(filter.limit).fetchAll(db)
        }
    }

    func eventCounts() throws -> [ActivityEventType: Int] {
        try dbQueue.read { db in
            var counts: [ActivityEventType: Int] = [:]
            let rows = try Row.fetchAll(db, sql: "SELECT type, COUNT(*) AS c FROM events GROUP BY type")
            for row in rows {
                if let type = ActivityEventType(rawValue: row["type"]) {
                    counts[type] = row["c"]
                }
            }
            return counts
        }
    }

    func clearHistory() throws {
        _ = try dbQueue.write { db in
            try ActivityEvent.deleteAll(db)
        }
    }

    /// Deletes events older than the retention window.
    func purgeExpired() throws {
        let cutoff = Date().addingTimeInterval(-Double(retentionDays) * 86400)
        _ = try dbQueue.write { db in
            try ActivityEvent.filter(Column("timestamp") < cutoff).deleteAll(db)
        }
    }
}
