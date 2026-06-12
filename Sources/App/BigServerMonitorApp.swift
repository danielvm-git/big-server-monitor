import SwiftUI

@main
struct BigServerMonitorApp: App {
    @State private var appState = AppState()
    @Environment(\.openWindow) private var openWindow

    var body: some Scene {
        // Primary: full dock window app
        Window("BigServerMonitor", id: "main") {
            MainAppView()
                .environment(appState)
        }
        .windowResizability(.contentMinSize)
        .defaultSize(width: 900, height: 600)

        // Secondary: menubar status icon — clicking opens/focuses the main window
        MenuBarExtra {
            MenuBarQuickMenu()
                .environment(appState)
        } label: {
            MenuBarLabel(activeCount: appState.activeCount)
        }
        .menuBarExtraStyle(.menu)

        Window("Health Check", id: "health") {
            HealthCheckSheet().environment(appState).activateOnAppear()
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)

        Window("Activity Log", id: "activity") {
            ActivityLogSheet().environment(appState).activateOnAppear()
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)

        Window("Settings", id: "settings") {
            SettingsSheet().environment(appState).activateOnAppear()
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)

        Window("Logs", id: "logs") {
            if let server = appState.logsServer {
                LogsSheet(server: server).environment(appState).activateOnAppear()
            }
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)
    }
}

extension View {
    func activateOnAppear() -> some View {
        onAppear { NSApp.activate(ignoringOtherApps: true) }
    }
}

// MARK: - Menubar status icon label

struct MenuBarLabel: View {
    let activeCount: Int

    var body: some View {
        HStack(spacing: 3) {
            Image(systemName: "waveform.path.ecg")
            if activeCount > 0 {
                Text("\(activeCount)")
                    .font(.system(size: 11, weight: .bold))
            }
        }
    }
}

// MARK: - Menubar quick menu (opens main window or tool sheets)

struct MenuBarQuickMenu: View {
    @Environment(AppState.self) private var appState
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        Button("Open BigServerMonitor") {
            openWindow(id: "main")
            NSApp.activate(ignoringOtherApps: true)
        }

        Divider()

        if appState.servers.isEmpty {
            Text("No servers running")
                .foregroundStyle(.secondary)
        } else {
            ForEach(appState.servers.prefix(8)) { server in
                Label {
                    Text(server.displayName)
                } icon: {
                    Circle()
                        .fill(server.status.color)
                        .frame(width: 8, height: 8)
                }
            }
        }

        Divider()

        Button("Health Check") { openWindow(id: "health"); NSApp.activate(ignoringOtherApps: true) }
        Button("Activity Log") { openWindow(id: "activity"); NSApp.activate(ignoringOtherApps: true) }
        Button("Settings")     { openWindow(id: "settings"); NSApp.activate(ignoringOtherApps: true) }

        Divider()

        Button("Quit BigServerMonitor") { NSApp.terminate(nil) }
    }
}
