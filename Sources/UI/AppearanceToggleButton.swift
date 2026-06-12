import AppKit
import SwiftUI

/// Toolbar button that cycles system → light → dark → system and applies the change immediately.
struct AppearanceToggleButton: View {
    @Environment(AppState.self) private var appState

    var body: some View {
        Button {
            let next = appState.configAppearanceMode.next
            appState.configAppearanceMode = next
            applyAppearance(next)
            Task { await appState.saveSettings() }
        } label: {
            Image(systemName: appState.configAppearanceMode.iconName)
        }
        .help("Toggle appearance (\(appState.configAppearanceMode.rawValue))")
    }

    private func applyAppearance(_ mode: AppearanceMode) {
        NSApp.appearance = switch mode {
        case .system: nil
        case .light:  NSAppearance(named: .aqua)
        case .dark:   NSAppearance(named: .darkAqua)
        }
    }
}

extension AppearanceMode {
    var next: AppearanceMode {
        switch self {
        case .system: .light
        case .light:  .dark
        case .dark:   .system
        }
    }

    var iconName: String {
        switch self {
        case .system: "circle.lefthalf.filled"
        case .light:  "sun.max"
        case .dark:   "moon"
        }
    }
}
