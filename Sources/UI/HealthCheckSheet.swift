import SwiftUI

/// HTTP reachability sheet per the design: dot · :port · name · pill · code · latency.
struct HealthCheckSheet: View {
    @Environment(AppState.self) private var appState
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            SheetTitleBar(title: "Health Check")

            HStack {
                Text("HTTP REACHABILITY")
                    .font(.system(size: 11, weight: .semibold))
                    .kerning(0.6)
                    .foregroundStyle(.tertiary)
                Spacer()
                if let checked = appState.healthLastChecked {
                    Text("Last checked \(checked.formatted(date: .omitted, time: .shortened))")
                        .font(.system(size: 11))
                        .foregroundStyle(.tertiary)
                }
            }
            .padding(.horizontal, 18)
            .padding(.vertical, 10)

            if appState.healthResults.isEmpty {
                Text("No results yet — run a check.")
                    .font(.system(size: 13))
                    .foregroundStyle(.tertiary)
                    .padding(.vertical, 32)
            } else {
                ScrollView {
                    VStack(spacing: 0) {
                        ForEach(appState.healthResults) { result in
                            row(result)
                            Divider()
                        }
                    }
                }
                .frame(maxHeight: 280)
            }

            Divider()
            HStack {
                Spacer()
                Button("Close") { dismiss() }
                Button("Test All Now") { appState.runHealthCheck() }
                    .buttonStyle(.borderedProminent)
            }
            .padding(.init(top: 12, leading: 18, bottom: 12, trailing: 18))
        }
        .frame(width: 540)
        .onAppear { appState.runHealthCheck() }
    }

    private func row(_ result: HealthResult) -> some View {
        HStack(spacing: 12) {
            Circle()
                .fill(color(result.status))
                .frame(width: 10, height: 10)
            Text(":\(String(result.port))")
                .font(.system(size: 13, weight: .semibold, design: .monospaced))
                .frame(width: 56, alignment: .leading)
            Text(serverName(result.port))
                .font(.system(size: 12))
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
            Text(tagLabel(result))
                .font(.system(size: 10, weight: .semibold))
                .kerning(0.4)
                .padding(.horizontal, 7)
                .padding(.vertical, 2)
                .background(color(result.status).opacity(0.15), in: Capsule())
                .foregroundStyle(color(result.status))
            Text(result.statusCode > 0 ? "\(result.statusCode)" : "—")
                .font(.system(size: 12, design: .monospaced))
                .frame(width: 36, alignment: .trailing)
            Text(result.latencyMS > 0 ? "\(result.latencyMS)ms" : "—")
                .font(.system(size: 12, design: .monospaced))
                .foregroundStyle(.tertiary)
                .frame(width: 52, alignment: .trailing)
        }
        .padding(.horizontal, 18)
        .padding(.vertical, 10)
    }

    private func serverName(_ port: Int) -> String {
        appState.servers.first { $0.port == port }?.projectName ?? ""
    }

    private func tagLabel(_ result: HealthResult) -> String {
        switch result.status {
        case .ok: result.latencyMS > 300 ? "SLOW" : "OK"
        case .warn: "WARN"
        case .error: "ERROR"
        case .timeout: "TIMEOUT"
        }
    }

    private func color(_ status: HealthStatus) -> Color {
        switch status {
        case .ok: .green
        case .warn: .orange
        case .error, .timeout: .red
        }
    }
}

/// Shared compact title bar for popover-presented sheets.
struct SheetTitleBar: View {
    let title: String
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        ZStack {
            Text(title)
                .font(.system(size: 13, weight: .semibold))
            HStack {
                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                Spacer()
            }
            .padding(.horizontal, 12)
        }
        .frame(height: 38)
        .background(.bar)
        .overlay(alignment: .bottom) { Divider() }
    }
}
