import SwiftUI

/// One server row per the design: status dot, mono :port, process, project,
/// uptime; expands to PID/Memory/Binary; trailing logs + kill buttons.
struct ServerRowView: View {
    @Environment(AppState.self) private var appState
    let server: Server
    @Binding var expandedPort: Int?
    @State private var hovering = false

    private var isExpanded: Bool { expandedPort == server.port }

    var body: some View {
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

                if isExpanded {
                    expandedDetails
                }
            }
            Spacer(minLength: 0)

            HStack(spacing: 4) {
                RowIconButton(systemName: "terminal", tint: .secondary) {
                    appState.openLogs(for: server)
                }
                .help("View logs")
                if server.status == .online {
                    RowIconButton(systemName: "xmark", tint: .red) {
                        appState.killTarget = server
                    }
                    .help("Kill process")
                }
            }
            .padding(.top, 2)
        }
        .padding(.vertical, 9)
        .padding(.horizontal, 14)
        .contentShape(Rectangle())
        .background(hovering ? Color.primary.opacity(0.05) : .clear)
        .onHover { hovering = $0 }
        .onTapGesture {
            withAnimation(.easeOut(duration: 0.15)) {
                expandedPort = isExpanded ? nil : server.port
            }
        }
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

struct RowIconButton: View {
    let systemName: String
    var tint: Color = .secondary
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: systemName)
                .font(.system(size: 10, weight: .bold))
                .foregroundStyle(tint)
                .frame(width: 22, height: 22)
                .background(tint.opacity(0.1), in: RoundedRectangle(cornerRadius: 5))
        }
        .buttonStyle(.plain)
    }
}
