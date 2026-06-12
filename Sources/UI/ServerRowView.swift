import SwiftUI

/// One server row per the design: status dot, mono :port, process, project,
/// uptime; expands to PID/Memory/Binary; trailing logs + kill buttons.
struct ServerRowView: View {
    @Environment(AppState.self) private var appState
    @Environment(\.openWindow) private var openWindow
    let server: Server
    @Binding var expandedPort: Int?

    private var isExpanded: Bool { expandedPort == server.port }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .top, spacing: 10) {
                statusDot
                    .padding(.top, 4)

                VStack(alignment: .leading, spacing: 1) {
                    HStack(alignment: .firstTextBaseline, spacing: 6) {
                        Text(":\(String(server.port))")
                            .font(.system(size: 13, weight: .semibold, design: .monospaced))
                        Text(server.processName)
                            .font(.system(size: 11))
                            .foregroundStyle(.secondary)
                    }
                    if let project = server.projectName, !project.isEmpty {
                        Text(project)
                            .font(.system(size: 12))
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                    Text(server.uptime)
                        .font(.system(size: 11))
                        .foregroundStyle(.tertiary)
                }

                Spacer(minLength: 4)

                // Expand/collapse chevron button
                Button {
                    withAnimation(.easeOut(duration: 0.15)) {
                        expandedPort = isExpanded ? nil : server.port
                    }
                } label: {
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(.secondary)
                        .frame(width: 20, height: 20)
                }
                .buttonStyle(.plain)
                .help(isExpanded ? "Collapse" : "Expand")
            }

            if isExpanded {
                expandedDetails
                    .padding(.leading, 18)
            }
        }
        .padding(.vertical, 9)
        .padding(.horizontal, 14)
        .overlay(alignment: .bottom) { Divider() }
    }

    private var statusDot: some View {
        Circle()
            .fill(dotColor)
            .frame(width: 8, height: 8)
            .shadow(color: server.status == .online ? dotColor.opacity(0.5) : .clear, radius: 2.5)
    }

    private var dotColor: Color {
        switch server.status {
        case .online: .green
        case .offline: .red
        case .unknown: .gray
        }
    }

    private var expandedDetails: some View {
        VStack(alignment: .leading, spacing: 8) {
            Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 6) {
                GridRow {
                    detail("PID", server.pid.map(String.init) ?? "—")
                    detail("MEMORY", server.memoryMB.map { String(format: "%.0f MB", $0) } ?? "—")
                }
                GridRow {
                    detail("BINARY", server.binaryPath ?? "—")
                        .gridCellColumns(2)
                }
            }

            HStack(spacing: 6) {
                Button {
                    appState.openLogs(for: server)
                    openWindow(id: "logs")
                } label: {
                    Label("Logs", systemImage: "terminal")
                        .font(.system(size: 11))
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                if server.status == .online {
                    Button(role: .destructive) {
                        appState.killTarget = server
                    } label: {
                        Label("Kill", systemImage: "xmark")
                            .font(.system(size: 11))
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }
        }
        .padding(8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 7))
        .padding(.top, 8)
    }

    private func detail(_ key: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(key)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(.tertiary)
            Text(value)
                .font(.system(size: 11, design: .monospaced))
                .lineLimit(1)
                .truncationMode(.middle)
        }
    }
}
