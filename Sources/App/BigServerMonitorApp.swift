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
