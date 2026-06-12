import SwiftUI

struct SidebarView: View {
    @Environment(AppState.self) private var appState
    @Environment(\.openWindow) private var openWindow
    @Binding var selection: SidebarItem?
    var searchText: String

    private var filteredServers: [Server] {
        appState.servers.filter { $0.matches(searchText: searchText) }
    }

    var body: some View {
        List(selection: $selection) {
            Section("Monitors") {
                Label {
                    HStack {
                        Text("Overview")
                        Spacer()
                        if appState.activeCount > 0 {
                            Text("\(appState.activeCount)")
                                .font(.caption2.bold())
                                .foregroundStyle(.white)
                                .padding(.horizontal, 5)
                                .padding(.vertical, 2)
                                .background(Color.accentColor, in: Capsule())
                        }
                    }
                } icon: {
                    Image(systemName: "square.grid.2x2")
                }
                .tag(SidebarItem.overview)
            }

            Section("Servers") {
                ForEach(filteredServers) { server in
                    Label {
                        Text(server.displayName)
                    } icon: {
                        Circle()
                            .fill(server.status.color)
                            .frame(width: 8, height: 8)
                    }
                    .tag(SidebarItem.server(server.id))
                }
            }

            Section("Tools") {
                Button {
                    openWindow(id: "health")
                    NSApp.activate(ignoringOtherApps: true)
                } label: {
                    Label("Health Check", systemImage: "stethoscope")
                }
                .buttonStyle(.plain)

                Button {
                    openWindow(id: "activity")
                    NSApp.activate(ignoringOtherApps: true)
                } label: {
                    Label("Activity Log", systemImage: "list.bullet.rectangle")
                }
                .buttonStyle(.plain)

                Button {
                    openWindow(id: "settings")
                    NSApp.activate(ignoringOtherApps: true)
                } label: {
                    Label("Settings", systemImage: "gearshape")
                }
                .buttonStyle(.plain)
            }
        }
        .listStyle(.sidebar)
    }
}

