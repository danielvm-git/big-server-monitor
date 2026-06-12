import SwiftUI

extension ServerStatus {
    var color: Color {
        switch self {
        case .online:  .green
        case .offline: .red
        case .unknown: .orange
        }
    }
}
