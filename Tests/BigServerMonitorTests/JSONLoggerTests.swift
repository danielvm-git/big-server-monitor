import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct JSONLoggerTests {

    @Test func writesValidJSONLines() async throws {
        let dir = temporaryDirectory()
        let logPath = dir + "/test.log"
        let logger = try JSONLogger(path: logPath)

        await logger.info("Server started", context: ["port": "3000", "pid": "1234"])
        await logger.error("Connection failed", context: ["port": "8080"])

        // Force flush by reading immediately — logger writes each line atomically
        let content = try String(contentsOfFile: logPath, encoding: .utf8)
        let lines = content.split(separator: "\n", omittingEmptySubsequences: true)

        #expect(lines.count == 2)

        let first = try JSONSerialization.jsonObject(with: Data(lines[0].utf8)) as? [String: Any]
        #expect(first?["level"] as? String == "INFO")
        #expect(first?["msg"] as? String == "Server started")
        #expect((first?["port"] as? String) == "3000")

        let second = try JSONSerialization.jsonObject(with: Data(lines[1].utf8)) as? [String: Any]
        #expect(second?["level"] as? String == "ERROR")
        #expect(second?["msg"] as? String == "Connection failed")
    }

    @Test func includesTimeAndLevelInEveryEntry() async throws {
        let dir = temporaryDirectory()
        let logPath = dir + "/test2.log"
        let logger = try JSONLogger(path: logPath)

        await logger.debug("debug test")
        await logger.warn("warn test")
        await logger.info("info test")
        await logger.error("error test")

        let content = try String(contentsOfFile: logPath, encoding: .utf8)
        let lines = content.split(separator: "\n", omittingEmptySubsequences: true)

        #expect(lines.count == 4)

        let expectedLevels = ["DEBUG", "WARN", "INFO", "ERROR"]
        for (i, line) in lines.enumerated() {
            let obj = try JSONSerialization.jsonObject(with: Data(line.utf8)) as? [String: Any]
            #expect(obj?["level"] as? String == expectedLevels[i])
            #expect(obj?["time"] as? String != nil)
            #expect(obj?["msg"] as? String != nil)
        }
    }

    @Test func createsParentDirectoryIfNeeded() async throws {
        let dir = temporaryDirectory() + "/nested/logs"
        let logPath = dir + "/app.log"
        let logger = try JSONLogger(path: logPath)

        await logger.info("test")

        var isDir: ObjCBool = false
        #expect(FileManager.default.fileExists(atPath: logPath))
        #expect(FileManager.default.fileExists(atPath: dir, isDirectory: &isDir))
        #expect(isDir.boolValue)
    }

    @Test func emptyContextIsOmitted() async throws {
        let dir = temporaryDirectory()
        let logPath = dir + "/test3.log"
        let logger = try JSONLogger(path: logPath)

        await logger.info("no context")

        let content = try String(contentsOfFile: logPath, encoding: .utf8)
        let lines = content.split(separator: "\n", omittingEmptySubsequences: true)
        let obj = try JSONSerialization.jsonObject(with: Data(lines[0].utf8)) as? [String: Any]

        // Should only have time, level, msg — no extra context keys
        #expect(obj?.count == 3)
        #expect(obj?["time"] != nil)
        #expect(obj?["level"] as? String == "INFO")
        #expect(obj?["msg"] as? String == "no context")
    }

    // MARK: - Helpers

    private func temporaryDirectory() -> String {
        let dir = NSTemporaryDirectory() + "JSONLoggerTests-\(UUID().uuidString)"
        try? FileManager.default.createDirectory(atPath: dir, withIntermediateDirectories: true)
        return dir
    }
}
