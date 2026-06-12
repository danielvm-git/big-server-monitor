import Foundation

enum HealthStatus: String, Codable, Sendable {
    case ok       // 2xx
    case warn     // 3xx / 4xx
    case error    // 5xx
    case timeout  // no response
}

struct HealthResult: Identifiable, Sendable, Equatable {
    var id: Int { port }
    let port: Int
    let status: HealthStatus
    let statusCode: Int
    let latencyMS: Int
    let checkedAt: Date
}

/// Classifies an HTTP response per the Go healthcheck component.
func classifyHealth(statusCode: Int) -> HealthStatus {
    switch statusCode {
    case 200..<300: .ok
    case 300..<500: .warn
    default: .error
    }
}

/// Probes local servers over HTTP. ≤10 concurrent probes, 3s timeout.
actor HealthChecker {
    private let timeout: TimeInterval
    private let maxConcurrent: Int
    private(set) var results: [Int: HealthResult] = [:]
    private let session: URLSession

    init(timeout: TimeInterval = 3, maxConcurrent: Int = 10) {
        self.timeout = timeout
        self.maxConcurrent = maxConcurrent
        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = timeout
        config.timeoutIntervalForResource = timeout
        self.session = URLSession(configuration: config)
    }

    /// Probe all given ports concurrently (bounded) and cache the results.
    func runAll(ports: [Int]) async -> [HealthResult] {
        var probed: [HealthResult] = []
        var index = 0
        while index < ports.count {
            let batch = Array(ports[index..<min(index + maxConcurrent, ports.count)])
            await withTaskGroup(of: HealthResult.self) { group in
                for port in batch {
                    group.addTask { await self.probe(port: port) }
                }
                for await result in group {
                    probed.append(result)
                }
            }
            index += batch.count
        }
        for result in probed {
            results[result.port] = result
        }
        return probed.sorted { $0.port < $1.port }
    }

    private func probe(port: Int) async -> HealthResult {
        let url = URL(string: "http://localhost:\(port)/")!
        var request = URLRequest(url: url)
        request.httpMethod = "HEAD"
        let start = Date()
        do {
            let (_, response) = try await session.data(for: request)
            let latency = Int(Date().timeIntervalSince(start) * 1000)
            let code = (response as? HTTPURLResponse)?.statusCode ?? 0
            return HealthResult(
                port: port, status: classifyHealth(statusCode: code),
                statusCode: code, latencyMS: latency, checkedAt: Date()
            )
        } catch {
            return HealthResult(
                port: port, status: .timeout, statusCode: 0,
                latencyMS: 0, checkedAt: Date()
            )
        }
    }
}
