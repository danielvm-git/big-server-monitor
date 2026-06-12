import SwiftUI

/// Detail panel shown when a specific server is selected in the sidebar.
struct ServerDetailView: View {
    @Environment(AppState.self) private var appState
    let server: Server

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack(spacing: 8) {
                Circle()
                    .fill(server.status.color)
                    .frame(width: 10, height: 10)
                Text(server.displayName)
                    .font(.title2.bold())
                Spacer()
                StatusBadge(status: server.status)
            }
            .padding(.horizontal)
            .padding(.top)

            Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 10) {
                GridRow {
                    Text("Port").foregroundStyle(.secondary)
                    Text(":\(server.port)").font(.system(.body, design: .monospaced))
                }
                GridRow {
                    Text("Process").foregroundStyle(.secondary)
                    Text(server.processName)
                }
                if let pid = server.pid {
                    GridRow {
                        Text("PID").foregroundStyle(.secondary)
                        Text("\(pid)").font(.system(.body, design: .monospaced))
                    }
                }
                if let mb = server.memoryMB {
                    GridRow {
                        Text("Memory").foregroundStyle(.secondary)
                        Text("\(Int(mb)) MB").font(.system(.body, design: .monospaced))
                    }
                }
                GridRow {
                    Text("Uptime").foregroundStyle(.secondary)
                    Text(server.uptime)
                }
                if let path = server.projectPath {
                    GridRow {
                        Text("Path").foregroundStyle(.secondary)
                        Text(path).font(.system(.caption, design: .monospaced)).lineLimit(1)
                    }
                }
            }
            .padding(.horizontal)

            Spacer()
        }
    }
}
