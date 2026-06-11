import Foundation

// Project-name detection and env redaction, ported from processmonitor.go.

/// Walks up from `workDir` looking for project markers; falls back to .git
/// directory name, then the directory's own name.
func detectProjectName(workDir: String, fileManager: FileManager = .default) -> String {
    guard !workDir.isEmpty else { return "" }

    var dir = workDir
    if dir.hasPrefix("~") {
        dir = NSString(string: dir).expandingTildeInPath
    }

    let markers: [(String, (String) -> String?)] = [
        ("package.json", parsePackageJSONName),
        ("go.mod", parseGoModName),
        ("Cargo.toml", { parseTOMLName($0, section: "[package]") }),
        ("pyproject.toml", { parseTOMLName($0, section: "[project]") }),
        ("composer.json", parsePackageJSONName),
    ]

    var current = dir
    while true {
        for (marker, parser) in markers {
            let path = (current as NSString).appendingPathComponent(marker)
            if fileManager.fileExists(atPath: path), let name = parser(path), !name.isEmpty {
                return name
            }
        }
        if fileManager.fileExists(atPath: (current as NSString).appendingPathComponent(".git")) {
            return (current as NSString).lastPathComponent
        }
        let parent = (current as NSString).deletingLastPathComponent
        if parent == current || parent.isEmpty { break }
        current = parent
    }

    return (dir as NSString).lastPathComponent
}

func parsePackageJSONName(_ path: String) -> String? {
    guard let data = FileManager.default.contents(atPath: path),
          let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
    else { return nil }
    return obj["name"] as? String
}

func parseGoModName(_ path: String) -> String? {
    guard let content = try? String(contentsOfFile: path, encoding: .utf8) else { return nil }
    for line in content.split(separator: "\n") {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        if trimmed.hasPrefix("module ") {
            return trimmed.dropFirst("module".count).trimmingCharacters(in: .whitespaces)
        }
    }
    return nil
}

/// Reads `name = "..."` from the given TOML section ([package] or [project]).
func parseTOMLName(_ path: String, section: String) -> String? {
    guard let content = try? String(contentsOfFile: path, encoding: .utf8) else { return nil }
    var inSection = false
    for line in content.split(separator: "\n") {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        if trimmed == section { inSection = true; continue }
        if inSection, trimmed.hasPrefix("[") { break }
        if inSection, trimmed.hasPrefix("name") {
            let parts = trimmed.split(separator: "=", maxSplits: 1)
            if parts.count == 2 {
                return parts[1].trimmingCharacters(in: .whitespaces)
                    .trimmingCharacters(in: CharacterSet(charactersIn: "\""))
            }
        }
    }
    return nil
}

// MARK: - Env redaction

/// Env vars safe to display verbatim.
let safeEnvKeys: Set<String> = [
    "NODE_ENV", "APP_ENV", "GO_ENV", "PORT", "HOST",
    "DATABASE_URL", "REDIS_URL", "LOG_LEVEL", "DEBUG",
]

/// Returns whether the variable is visible and its (possibly redacted) value.
/// Allowlist-based: anything not explicitly safe is masked.
func redactEnvVar(key: String, value: String) -> (visible: Bool, value: String) {
    if safeEnvKeys.contains(key) { return (true, value) }
    return (false, "***")
}
