import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct LogCaptureTests {
    @Test func classifiesLevels() {
        #expect(classifyLogLine("GET /api/health → 200 OK") == .info)
        #expect(classifyLogLine("WARN Memory usage above threshold") == .warn)
        #expect(classifyLogLine("[vite] hmr update — deprecated API") == .warn)
        #expect(classifyLogLine("sqlalchemy.exc.OperationalError: no such table") == .error)
        #expect(classifyLogLine("Traceback (most recent call last):") == .error)
        #expect(classifyLogLine("  at Object.fn (/app/index.js:42)") == .error)
        #expect(classifyLogLine("process exited with exit code 1") == .error)
        #expect(classifyLogLine("clean exit code 0") == .info)
    }

    @Test func ringBufferCapsAt500Lines() async {
        let capture = LogCapture()
        for i in 1...520 {
            await capture.ingest(port: 3000, text: "line \(i)")
        }
        let lines = await capture.lines(port: 3000)

        #expect(lines.count == 500)
        #expect(lines.first?.text == "line 21")
        #expect(lines.last?.text == "line 520")
    }

    @Test func filtersByLevelAndCounts() async {
        let capture = LogCapture()
        await capture.ingest(port: 3000, text: "ok line")
        await capture.ingest(port: 3000, text: "WARN slow query")
        await capture.ingest(port: 3000, text: "ERROR boom")

        let errors = await capture.lines(port: 3000, level: .error)
        #expect(errors.count == 1)

        let counts = await capture.counts(port: 3000)
        #expect(counts[.info] == 1)
        #expect(counts[.warn] == 1)
        #expect(counts[.error] == 1)
    }

    @Test func clearRemovesBuffer() async {
        let capture = LogCapture()
        await capture.ingest(port: 3000, text: "hello")
        await capture.clear(port: 3000)
        let lines = await capture.lines(port: 3000)

        #expect(lines.isEmpty)
    }

    @Test func aiExportHasContextAndSections() async {
        let capture = LogCapture()
        let server = Server(
            port: 8080, processName: "python3", pid: 555, status: .online,
            projectName: "api-server", projectPath: nil,
            binaryPath: "/usr/bin/python3", memoryMB: 64,
            startedAt: Date(timeIntervalSinceNow: -3600)
        )
        await capture.ingest(port: 8080, text: "Starting api-server on :8080")
        await capture.ingest(port: 8080, text: "Traceback (most recent call last):")

        let export = await capture.aiExport(for: server)

        #expect(export.hasPrefix("=== BigServerMonitor Log Export ==="))
        #expect(export.contains("Server:  api-server"))
        #expect(export.contains("Process: python3  (PID 555)"))
        #expect(export.contains("Port:    :8080"))
        #expect(export.contains("--- stdout / stderr (1 lines) ---"))
        #expect(export.contains("--- Errors & warnings (1 lines) ---"))
        #expect(export.contains("Traceback"))
    }

    @Test func aiExportEmptyWhenNoLines() async {
        let capture = LogCapture()
        let server = Server(
            port: 9999, processName: "x", pid: nil, status: .offline,
            projectName: nil, projectPath: nil, binaryPath: nil,
            memoryMB: nil, startedAt: nil
        )
        #expect(await capture.aiExport(for: server) == "")
    }
}
