import SwiftUI

/// 360pt popover per specs/design-handoff/PortKeeper.html.
struct PopoverView: View {
    @Environment(AppState.self) private var appState
    @Environment(\.openWindow) private var openWindow
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
                // VStack, not LazyVStack: MenuBarExtra's window sizing pass
                // proposes zero height to lazy content, collapsing the list.
                // ScrollView won't grow on its own here either, so size it
                // explicitly from the row count (~56pt/row), capped at 280.
                ScrollView {
                    VStack(spacing: 0) {
                        ForEach(appState.servers) { server in
                            ServerRowView(server: server, expandedPort: $expandedPort)
                        }
                    }
                    .fixedSize(horizontal: false, vertical: true)
                }
                .frame(height: min(280, CGFloat(appState.servers.count) * 56))
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
                openWindow(id: "health")
            }
            FooterButton(icon: "clock.arrow.circlepath", title: "Activity Log") {
                openWindow(id: "activity")
            }
            FooterButton(icon: "gearshape", title: "Settings") {
                openWindow(id: "settings")
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
