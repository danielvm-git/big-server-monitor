import SwiftUI

/// 360pt popover skeleton per specs/design-handoff/PortKeeper.html.
/// Server rows, projects, and sheets land in e03/e04.
struct PopoverView: View {
    @Environment(AppState.self) private var appState

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
            }

            Divider()
            footer
        }
        .frame(width: 360)
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
                // refresh wired in e03
            } label: {
                Image(systemName: "arrow.clockwise")
                    .font(.system(size: 11))
            }
            .buttonStyle(.bordered)
            .controlSize(.small)
        }
        .padding(.init(top: 12, leading: 14, bottom: 10, trailing: 14))
    }

    private var footer: some View {
        VStack(spacing: 2) {
            FooterButton(icon: "waveform.path.ecg", title: "Health Check") {}
            FooterButton(icon: "clock.arrow.circlepath", title: "Activity Log") {}
            FooterButton(icon: "gearshape", title: "Settings") {}
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
