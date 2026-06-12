import Foundation
import Testing
@testable import BigServerMonitor

@Suite struct PortDiscoveryTests {
    let lsofFixture = """
    COMMAND   PID      USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
    node     1024 danielvm   23u  IPv4 0x1a2b3c4d5e6f      0t0  TCP *:3000 (LISTEN)
    node     1024 danielvm   24u  IPv6 0x1a2b3c4d5e70      0t0  TCP [::1]:3000 (LISTEN)
    vite     2048 danielvm   31u  IPv4 0x2b3c4d5e6f70      0t0  TCP 127.0.0.1:5173 (LISTEN)
    bun      3120 danielvm   12u  IPv4 0x3c4d5e6f7081      0t0  TCP *:4321 (LISTEN)
    """

    @Test func parsesLsofListeningPorts() {
        #expect(parseLsofOutput(lsofFixture) == [3000, 4321, 5173])
    }

    @Test func lsofParsingIgnoresShortLines() {
        #expect(parseLsofOutput("garbage line\nshort\n") == [])
        #expect(parseLsofOutput("") == [])
    }

    @Test func extractsPortFromAddressForms() {
        #expect(extractPort("*:8080") == 8080)
        #expect(extractPort("127.0.0.1:3000") == 3000)
        #expect(extractPort("[::1]:5173") == 5173)
        #expect(extractPort("no-port") == 0)
        #expect(extractPort("") == 0)
    }

    @Test func parsesNetstatFallback() {
        let netstat = """
        Active Internet connections (including servers)
        Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
        tcp4       0      0  127.0.0.1.3000         *.*                    LISTEN
        tcp46      0      0  *.8080                 *.*                    LISTEN
        tcp4       0      0  192.168.1.5.49152      1.2.3.4.443            ESTABLISHED
        """
        #expect(parseNetstatOutput(netstat) == [3000, 8080])
    }

    @Test func findsFirstPIDSkippingHeader() {
        #expect(firstPID(inLsofOutput: lsofFixture) == 1024)
        #expect(firstPID(inLsofOutput: "COMMAND PID USER\n") == nil)
    }

    @Test func parsesEtimeFormats() {
        #expect(parseEtime("02:03") == 123)
        #expect(parseEtime("01:02:03") == 3723)
        #expect(parseEtime("2-01:02:03") == 2 * 86400 + 3723)
        #expect(parseEtime("45") == 45)
        #expect(parseEtime("") == 0)
    }
}
