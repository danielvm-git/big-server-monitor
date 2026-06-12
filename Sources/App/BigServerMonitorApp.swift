import SwiftUI

@main
struct BigServerMonitorApp: App {
    @State private var appState = AppState()

    var body: some Scene {
        MenuBarExtra {
            PopoverView()
                .environment(appState)
        } label: {
            MenuBarLabel(activeCount: appState.activeCount)
        }
        .menuBarExtraStyle(.window)

        // Standalone windows, not .sheet: sheets attached to the MenuBarExtra
        // popover die when the popover auto-dismisses on first click.
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
    /// LSUIElement apps don't activate on openWindow; force focus.
    func activateOnAppear() -> some View {
        onAppear { NSApp.activate(ignoringOtherApps: true) }
    }
}

/// Template-style pulse icon plus active-server count, per the design's
/// menubar item (zap icon + badge).
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
