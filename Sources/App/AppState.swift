import Observation
import SwiftUI

/// Root observable model. Wires the core services to the UI.
/// (Populated with real services in epics e02+; e01 ships the skeleton.)
@Observable
@MainActor
final class AppState {
    var servers: [Server] = []

    var activeCount: Int {
        servers.filter { $0.status == .online }.count
    }
}
