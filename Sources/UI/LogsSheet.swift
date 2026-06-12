import SwiftUI

private enum LogFilter: String, CaseIterable {
    case all = "All"
    case errors = "Errors"
    case warnings = "Warnings"
}

/// Per-server log viewer: filter bar, dark terminal pane, Copy / Copy for AI.
struct LogsSheet: View {
    @Environment(AppState.self) private var appState
    @Environment(\.dismiss) private var dismiss
    let server: Server

    @State private var filter: LogFilter = .all
    @State private var copied = false
    @State private var copiedAI = false

    private var allLines: [LogLine] { appState.logLines }

    private var filtered: [LogLine] {
        switch filter {
        case .all: allLines
        case .errors: allLines.filter { $0.level == .error }
        case .warnings: allLines.filter { $0.level == .error || $0.level == .warn }
        }
    }

    private var errCount: Int { allLines.filter { $0.level == .error }.count }
    private var warnCount: Int { allLines.filter { $0.level == .warn }.count }

    var body: some View {
        VStack(spacing: 0) {
            SheetTitleBar(title: "Logs — \(server.projectName ?? server.processName)  :\(String(server.port))")

            filterBar
            Divider()
            terminal
            Divider()
            aiHint
            Divider()
            footer
        }
        .frame(width: 620)
    }

    private var filterBar: some View {
        HStack(spacing: 4) {
            ForEach(LogFilter.allCases, id: \.self) { f in
                let count = f == .all ? allLines.count : (f == .errors ? errCount : warnCount)
                Button {
                    filter = f
                } label: {
                    Text("\(f.rawValue) (\(count))")
                        .font(.system(size: 12, weight: filter == f ? .semibold : .regular))
                        .padding(.horizontal, 10)
                        .padding(.vertical, 3)
                        .background(
                            filter == f ? Color.accentColor : Color.primary.opacity(0.06),
                            in: RoundedRectangle(cornerRadius: 5)
                        )
                        .foregroundStyle(filter == f ? .white : .secondary)
                }
                .buttonStyle(.plain)
                .disabled(f != .all && count == 0)
            }
            Spacer()
            Text("\(filtered.count) line\(filtered.count == 1 ? "" : "s")")
                .font(.system(size: 11))
                .foregroundStyle(.tertiary)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
    }

    private var terminal: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if filtered.isEmpty {
                    Text(filter == .all
                        ? "No system-log entries for this process in the last 5 minutes.\nProcesses writing only to their own stdout/stderr won't appear here."
                        : "No \(filter.rawValue.lowercased()) entries in log.")
                        .foregroundStyle(Color(red: 0.52, green: 0.52, blue: 0.52))
                } else {
                    ForEach(filtered) { line in
                        HStack(alignment: .firstTextBaseline, spacing: 8) {
                            Text("[\(line.timestamp.formatted(.dateTime.hour().minute().second()))]")
                                .foregroundStyle(Color(red: 0.52, green: 0.52, blue: 0.52))
                            Text(line.text)
                                .foregroundStyle(lineColor(line.level))
                        }
                        .lineSpacing(4)
                    }
                }
            }
            .font(.system(size: 12, design: .monospaced))
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.init(top: 14, leading: 16, bottom: 14, trailing: 16))
        }
        .frame(minHeight: 240, maxHeight: 280)
        .background(Color(red: 0.118, green: 0.118, blue: 0.118))
    }

    private func lineColor(_ level: LogLevel) -> Color {
        switch level {
        case .debug: Color(red: 0.55, green: 0.55, blue: 0.55)
        case .info: Color(red: 0.83, green: 0.83, blue: 0.83)
        case .warn: Color(red: 0.86, green: 0.86, blue: 0.67)
        case .error: Color(red: 0.96, green: 0.28, blue: 0.28)
        }
    }

    private var aiHint: some View {
        Text("💡 **Copy for AI agent** includes server context (process, PID, binary, memory) + full log — ready to paste into Claude or ChatGPT.")
            .font(.system(size: 11))
            .foregroundStyle(.tertiary)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
    }

    private var footer: some View {
        HStack(spacing: 8) {
            Spacer()
            Button {
                appState.copyLogs(filtered)
                flash($copied)
            } label: {
                Label(copied ? "✓ Copied!" : "Copy", systemImage: "doc.on.doc")
            }
            Button {
                appState.copyLogsForAI(server: server)
                flash($copiedAI)
            } label: {
                Label(copiedAI ? "✓ Copied for AI!" : "Copy for AI agent", systemImage: "sparkles")
            }
            .buttonStyle(.borderedProminent)
            Button("Done") { dismiss() }
        }
        .padding(.init(top: 12, leading: 18, bottom: 12, trailing: 18))
    }

    private func flash(_ flag: Binding<Bool>) {
        flag.wrappedValue = true
        Task {
            try? await Task.sleep(for: .seconds(2))
            flag.wrappedValue = false
        }
    }
}
