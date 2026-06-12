import SwiftUI
import Testing
@testable import BigServerMonitor

@Suite("SidebarColorTests")
struct SidebarColorTests {
    @Test func onlineIsGreen() {
        #expect(ServerStatus.online.color == Color.green)
    }

    @Test func offlineIsRed() {
        #expect(ServerStatus.offline.color == Color.red)
    }

    @Test func unknownIsOrange() {
        #expect(ServerStatus.unknown.color == Color.orange)
    }

    @Test func onlineLabelIsRunning() {
        #expect(ServerStatus.online.statusLabel == "running")
    }

    @Test func offlineLabelIsCrashed() {
        #expect(ServerStatus.offline.statusLabel == "crashed")
    }

    @Test func unknownLabelIsUnresponsive() {
        #expect(ServerStatus.unknown.statusLabel == "unresponsive")
    }
}
