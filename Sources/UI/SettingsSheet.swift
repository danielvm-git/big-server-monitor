import SwiftUI

struct SettingsSheet: View {
    @Environment(AppState.self) private var appState
    @Environment(\.dismiss) private var dismiss

    @State private var ignoredPortsText: String = ""

    var body: some View {
        VStack(spacing: 0) {
            SheetTitleBar(title: "Settings")

            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    monitoringSection
                    notificationsSection
                    startupSection
                }
                .padding(18)
            }
            .frame(maxHeight: 380)

            Divider()
            HStack {
                Spacer()
                Button("Close") { dismiss() }
            }
            .padding(.init(top: 12, leading: 18, bottom: 12, trailing: 18))
        }
        .frame(width: 500)
        .onAppear { loadConfig() }
        .onDisappear { saveConfig() }
    }

    // MARK: - Sections

    private var monitoringSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            sectionHeader("MONITORING")

            intervalRow(label: "Polling interval", seconds: pollingBinding)
            intervalRow(label: "Health check interval", seconds: healthBinding)

            VStack(alignment: .leading, spacing: 6) {
                Text("Ignored ports (comma-separated)")
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
                TextField("e.g. 5000, 7000", text: $ignoredPortsText)
                    .textFieldStyle(.roundedBorder)
                    .font(.system(size: 12, design: .monospaced))
            }
        }
    }

    private var notificationsSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            sectionHeader("NOTIFICATIONS")

            Toggle("Crash alerts", isOn: crashAlertsBinding)
                .font(.system(size: 13))
            Toggle("Menu bar badge", isOn: showBadgeBinding)
                .font(.system(size: 13))
        }
    }

    private var startupSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            sectionHeader("STARTUP")

            Toggle("Launch at login", isOn: launchAtLoginBinding)
                .font(.system(size: 13))
                .onChange(of: appState.configLaunchAtLogin) { _, newValue in
                    Task { await appState.setLaunchAtLogin(newValue) }
                }
        }
    }

    // MARK: - Helpers

    private func sectionHeader(_ title: String) -> some View {
        Text(title)
            .font(.system(size: 11, weight: .semibold))
            .kerning(0.6)
            .foregroundStyle(.tertiary)
    }

    private func intervalRow(label: String, seconds: Binding<Double>) -> some View {
        HStack(spacing: 12) {
            Text(label)
                .font(.system(size: 13))
                .foregroundStyle(.secondary)
                .frame(width: 140, alignment: .leading)
            Text("\(Int(seconds.wrappedValue))s")
                .font(.system(size: 12, design: .monospaced))
                .foregroundStyle(.tertiary)
                .frame(width: 40, alignment: .trailing)
            Slider(value: seconds, in: 1...300, step: 1)
        }
    }

    private func loadConfig() {
        ignoredPortsText = appState.configIgnoredPorts.map(String.init).joined(separator: ", ")
    }

    private func saveConfig() {
        let ports = ignoredPortsText
            .split(separator: ",")
            .compactMap { Int($0.trimmingCharacters(in: .whitespaces)) }
        appState.configIgnoredPorts = ports
        Task { await appState.saveSettings() }
    }

    // MARK: - Bindings

    private var pollingBinding: Binding<Double> {
        Binding(get: { appState.configPollingInterval }, set: { appState.configPollingInterval = $0 })
    }

    private var healthBinding: Binding<Double> {
        Binding(get: { appState.configHealthInterval }, set: { appState.configHealthInterval = $0 })
    }

    private var crashAlertsBinding: Binding<Bool> {
        Binding(get: { appState.configCrashAlerts }, set: { appState.configCrashAlerts = $0 })
    }

    private var showBadgeBinding: Binding<Bool> {
        Binding(get: { appState.configShowBadge }, set: { appState.configShowBadge = $0 })
    }

    private var launchAtLoginBinding: Binding<Bool> {
        Binding(get: { appState.configLaunchAtLogin }, set: { appState.configLaunchAtLogin = $0 })
    }
}
