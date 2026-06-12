import SwiftUI

struct OverviewPanel: View {
    @Environment(AppState.self) private var appState
    var searchText: String

    private var filtered: [Server] {
        guard !searchText.isEmpty else { return appState.servers }
        return appState.servers.filter {
            $0.displayName.localizedCaseInsensitiveContains(searchText) ||
            String($0.port).contains(searchText)
        }
    }

    private var activeCount: Int   { appState.servers.filter { $0.status == .online }.count }
    private var crashedCount: Int  { appState.servers.filter { $0.status == .offline }.count }
    private var unknownCount: Int  { appState.servers.filter { $0.status == .unknown }.count }
    private var totalMemoryMB: Double { appState.servers.compactMap(\.memoryMB).reduce(0, +) }

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            VStack(alignment: .leading, spacing: 2) {
                Text("Overview")
                    .font(.title2.bold())
                Text("Monitoring \(appState.servers.count) development server\(appState.servers.count == 1 ? "" : "s")")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal)
            .padding(.top)

            HStack(spacing: 12) {
                StatCard(value: "\(activeCount)",
                         label: "ACTIVE",
                         subtitle: "of \(appState.servers.count) servers",
                         valueColor: .primary)
                StatCard(value: "\(crashedCount)",
                         label: "CRASHED",
                         subtitle: crashedCount > 0 ? "needs attention" : "all clear",
                         valueColor: crashedCount > 0 ? .red : .secondary)
                StatCard(value: "\(unknownCount)",
                         label: "UNRESPONSIVE",
                         subtitle: "check ports",
                         valueColor: unknownCount > 0 ? .orange : .secondary)
                StatCard(value: String(format: "%.0f", totalMemoryMB),
                         label: "MEMORY",
                         subtitle: "MB across all procs",
                         valueColor: .primary)
            }
            .padding(.horizontal)

            VStack(alignment: .leading, spacing: 8) {
                Text("SERVERS")
                    .font(.caption.bold())
                    .foregroundStyle(.secondary)
                    .padding(.horizontal)

                if filtered.isEmpty {
                    ContentUnavailableView(
                        searchText.isEmpty ? "No Servers" : "No Results",
                        systemImage: "server.rack",
                        description: Text(searchText.isEmpty ? "Start a local server to see it here." : "No servers match \"\(searchText)\".")
                    )
                } else {
                    ServerTable(servers: filtered)
                }
            }

            Spacer(minLength: 0)
        }
    }
}

// MARK: - Stat Card

struct StatCard: View {
    let value: String
    let label: String
    let subtitle: String
    let valueColor: Color

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption2.bold())
                .foregroundStyle(.secondary)
            Text(value)
                .font(.system(size: 32, weight: .bold, design: .rounded))
                .foregroundStyle(valueColor)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
    }
}

// MARK: - Server Table

struct ServerTable: View {
    let servers: [Server]

    var body: some View {
        Table(servers) {
            TableColumn("Name") { server in
                HStack(spacing: 6) {
                    Circle()
                        .fill(server.status.color)
                        .frame(width: 8, height: 8)
                    Text(server.displayName)
                        .lineLimit(1)
                }
            }
            TableColumn("Port") { server in
                Text(":\(server.port)")
                    .font(.system(.body, design: .monospaced))
                    .foregroundStyle(.secondary)
            }
            .width(70)
            TableColumn("Status") { server in
                StatusBadge(status: server.status)
            }
            .width(100)
            TableColumn("Memory") { server in
                if let mb = server.memoryMB {
                    Text("\(Int(mb)) MB")
                        .font(.system(.body, design: .monospaced))
                } else {
                    Text("—").foregroundStyle(.tertiary)
                }
            }
            .width(80)
            TableColumn("Uptime") { server in
                Text(server.uptime)
                    .font(.system(.body, design: .monospaced))
                    .foregroundStyle(.secondary)
            }
            .width(80)
        }
    }
}

// MARK: - Status Badge

struct StatusBadge: View {
    let status: ServerStatus

    var body: some View {
        Text(status.label)
            .font(.caption.bold())
            .foregroundStyle(status.color)
            .padding(.horizontal, 7)
            .padding(.vertical, 3)
            .background(status.color.opacity(0.12), in: Capsule())
    }
}

extension ServerStatus {
    var label: String {
        switch self {
        case .online:  "running"
        case .offline: "crashed"
        case .unknown: "unresponsive"
        }
    }
}
