import SwiftUI

/// 360pt popover per specs/design-handoff/PortKeeper.html.
struct PopoverView: View {
    @Environment(AppState.self) private var appState
    @State private var expandedPort: Int?

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            Divider()

            if appState.servers.isEmpty {
                Text("No servers detected")
                    .font(.system(size: 12))
                    .foregroundStyle(.tertiary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 28)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(appState.servers) { server in
                            ServerRowView(server: server, expandedPort: $expandedPort)
                        }
                    }
                }
                .frame(maxHeight: 280)
            }

            if !appState.projectGroups.isEmpty {
                Divider()
                projectsSection
            }

            Divider()
            footer
        }
        .frame(width: 360)
        .alert(
            "Kill \(appState.killTarget?.processName ?? "") on :\(String(appState.killTarget?.port ?? 0))?",
            isPresented: killAlertBinding
        ) {
            Button("Cancel", role: .cancel) {}
            Button("Kill", role: .destructive) { appState.confirmKill() }
        } message: {
            Text("\(appState.killTarget?.projectName ?? "Process") (PID \(appState.killTarget?.pid.map(String.init) ?? "—")) will be killed and removed from the list.")
        }
        .sheet(item: sheetBinding) { sheet in
            switch sheet {
            case .health: HealthCheckSheet()
            case .activity: ActivityLogSheet()
            case .settings: Text("Settings — coming in e05").padding(40)
            }
        }
        .sheet(item: logsServerBinding) { server in
            LogsSheet(server: server)
        }
    }

    private var sheetBinding: Binding<ActiveSheet?> {
        Binding(
            get: { appState.activeSheet },
            set: { appState.activeSheet = $0 }
        )
    }

    private var logsServerBinding: Binding<Server?> {
        Binding(
            get: { appState.logsServer },
            set: { appState.logsServer = $0 }
        )
    }

    private var killAlertBinding: Binding<Bool> {
        Binding(
            get: { appState.killTarget != nil },
            set: { if !$0 { appState.killTarget = nil } }
        )
    }

    private var header: some View {
        HStack(spacing: 8) {
            LogoMark(size: 24)
            VStack(alignment: .leading, spacing: 1) {
                Wordmark(fontSize: 14)
                Text("\(appState.activeCount) server\(appState.activeCount == 1 ? "" : "s") active")
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Button {
                appState.refresh()
            } label: {
                Image(systemName: "arrow.clockwise")
                    .font(.system(size: 11))
            }
            .buttonStyle(.bordered)
            .controlSize(.small)
            .help("Refresh")
        }
        .padding(.init(top: 12, leading: 14, bottom: 10, trailing: 14))
    }

    private var projectsSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("PROJECTS")
                .font(.system(size: 10, weight: .semibold))
                .kerning(0.6)
                .foregroundStyle(.tertiary)
            ForEach(appState.projectGroups, id: \.path) { group in
                HStack(spacing: 6) {
                    Image(systemName: "folder")
                        .font(.system(size: 11))
                        .foregroundStyle(.secondary)
                    Text(group.path)
                        .font(.system(size: 12, design: .monospaced))
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Text("· \(group.active) active")
                        .font(.system(size: 11))
                        .foregroundStyle(.tertiary)
                }
            }
        }
        .padding(.vertical, 8)
        .padding(.horizontal, 14)
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var footer: some View {
        VStack(spacing: 2) {
            FooterButton(icon: "waveform.path.ecg", title: "Health Check") {
                appState.activeSheet = .health
            }
            FooterButton(icon: "clock.arrow.circlepath", title: "Activity Log") {
                appState.activeSheet = .activity
            }
            FooterButton(icon: "gearshape", title: "Settings") {
                appState.activeSheet = .settings
            }
        }
        .padding(8)
    }
}

struct FooterButton: View {
    let icon: String
    let title: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 8) {
                Image(systemName: icon)
                    .font(.system(size: 13))
                    .foregroundStyle(.secondary)
                    .frame(width: 16)
                Text(title)
                    .font(.system(size: 13))
                Spacer()
            }
            .padding(.vertical, 7)
            .padding(.horizontal, 10)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
