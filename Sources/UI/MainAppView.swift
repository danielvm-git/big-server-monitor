import SwiftUI

struct MainAppView: View {
    @Environment(AppState.self) private var appState
    @State private var selection: SidebarItem? = .overview
    @State private var searchText = ""

    var body: some View {
        NavigationSplitView {
            SidebarView(selection: $selection, searchText: searchText)
                .navigationSplitViewColumnWidth(min: 180, ideal: 200)
        } detail: {
            switch selection {
            case .overview, .none:
                OverviewPanel(searchText: searchText)
            case .server(let id):
                if let server = appState.servers.first(where: { $0.id == id }) {
                    ServerDetailView(server: server)
                } else {
                    OverviewPanel(searchText: searchText)
                }
            }
        }
        .searchable(text: $searchText, placement: .toolbar, prompt: "Search servers…")
        .toolbar {
            ToolbarItemGroup(placement: .automatic) {
                AppearanceToggleButton()
                Button { appState.refresh() } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .help("Refresh server list")
            }
        }
        .navigationTitle("")
        .frame(minWidth: 700, minHeight: 450)
    }
}

// MARK: - Sidebar selection

enum SidebarItem: Hashable {
    case overview
    case server(Int)
}
