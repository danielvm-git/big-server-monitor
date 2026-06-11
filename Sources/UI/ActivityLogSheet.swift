import SwiftUI

/// Timeline of process events per the design: colored dot, title, body, mono time.
struct ActivityLogSheet: View {
    @Environment(AppState.self) private var appState
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            SheetTitleBar(title: "Activity Log")

            if appState.activityEvents.isEmpty {
                Text("No activity recorded yet.")
                    .font(.system(size: 13))
                    .foregroundStyle(.tertiary)
                    .padding(.vertical, 40)
            } else {
                ScrollView {
                    VStack(spacing: 0) {
                        ForEach(appState.activityEvents) { event in
                            row(event)
                            Divider()
                        }
                    }
                }
                .frame(maxHeight: 320)
            }

            Divider()
            HStack {
                Button("Clear History") { appState.clearHistory() }
                Spacer()
                Button("Done") { dismiss() }
            }
            .padding(.init(top: 12, leading: 18, bottom: 12, trailing: 18))
        }
        .frame(width: 540)
        .onAppear { appState.loadActivity() }
    }

    private func row(_ event: ActivityEvent) -> some View {
        HStack(alignment: .top, spacing: 12) {
            Circle()
                .fill(color(event.type))
                .frame(width: 8, height: 8)
                .padding(.top, 4)
            VStack(alignment: .leading, spacing: 2) {
                Text("\(event.processName) :\(String(event.port)) \(event.type.rawValue)")
                    .font(.system(size: 13, weight: .medium))
                if let message = event.message {
                    Text(message)
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                }
            }
            Spacer()
            Text(event.timestamp.formatted(date: .omitted, time: .shortened))
                .font(.system(size: 11, design: .monospaced))
                .foregroundStyle(.tertiary)
        }
        .padding(.horizontal, 18)
        .padding(.vertical, 11)
    }

    private func color(_ type: ActivityEventType) -> Color {
        switch type {
        case .started: .green
        case .stopped: .gray
        case .crashed: .red
        case .unresponsive: .orange
        }
    }
}
