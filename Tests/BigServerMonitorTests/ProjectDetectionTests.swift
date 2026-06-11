import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct ProjectDetectionTests {
    private func makeTempDir() throws -> URL {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("bsm-tests-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        return url
    }

    @Test func detectsNameFromPackageJSON() throws {
        let dir = try makeTempDir()
        defer { try? FileManager.default.removeItem(at: dir) }
        try #"{"name": "bigbase-api", "version": "1.0.0"}"#
            .write(to: dir.appendingPathComponent("package.json"), atomically: true, encoding: .utf8)

        #expect(detectProjectName(workDir: dir.path) == "bigbase-api")
    }

    @Test func detectsNameFromGoMod() throws {
        let dir = try makeTempDir()
        defer { try? FileManager.default.removeItem(at: dir) }
        try "module portkeeper\n\ngo 1.22\n"
            .write(to: dir.appendingPathComponent("go.mod"), atomically: true, encoding: .utf8)

        #expect(detectProjectName(workDir: dir.path) == "portkeeper")
    }

    @Test func detectsNameFromCargoToml() throws {
        let dir = try makeTempDir()
        defer { try? FileManager.default.removeItem(at: dir) }
        try "[package]\nname = \"my-rust-app\"\nversion = \"0.1.0\"\n"
            .write(to: dir.appendingPathComponent("Cargo.toml"), atomically: true, encoding: .utf8)

        #expect(detectProjectName(workDir: dir.path) == "my-rust-app")
    }

    @Test func walksUpToParentMarker() throws {
        let dir = try makeTempDir()
        defer { try? FileManager.default.removeItem(at: dir) }
        let sub = dir.appendingPathComponent("src/server")
        try FileManager.default.createDirectory(at: sub, withIntermediateDirectories: true)
        try #"{"name": "monorepo-root"}"#
            .write(to: dir.appendingPathComponent("package.json"), atomically: true, encoding: .utf8)

        #expect(detectProjectName(workDir: sub.path) == "monorepo-root")
    }

    @Test func gitDirectoryIsFallbackMarker() throws {
        let dir = try makeTempDir()
        defer { try? FileManager.default.removeItem(at: dir) }
        try FileManager.default.createDirectory(
            at: dir.appendingPathComponent(".git"), withIntermediateDirectories: true)

        #expect(detectProjectName(workDir: dir.path) == dir.lastPathComponent)
    }

    @Test func emptyWorkDirGivesEmptyName() {
        #expect(detectProjectName(workDir: "") == "")
    }

    @Test func redactsSensitiveEnvVars() {
        let cases: [(String, String, Bool, String)] = [
            ("NODE_ENV", "production", true, "production"),
            ("PORT", "3000", true, "3000"),
            ("AWS_SECRET_ACCESS_KEY", "abc123", false, "***"),
            ("GITHUB_TOKEN", "ghp_xyz", false, "***"),
            ("MY_PASSWORD", "hunter2", false, "***"),
            ("RANDOM_VAR", "value", false, "***"),
        ]
        for (key, value, wantVisible, wantValue) in cases {
            let result = redactEnvVar(key: key, value: value)
            #expect(result.visible == wantVisible, "key \(key)")
            #expect(result.value == wantValue, "key \(key)")
        }
    }
}
