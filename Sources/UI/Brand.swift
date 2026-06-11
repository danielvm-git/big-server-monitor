import SwiftUI

/// Brand constants from the design handoff (specs/design-handoff/).
enum Brand {
    /// Indigo brand accent — #4F46E5.
    static let indigo = Color(red: 0x4F / 255, green: 0x46 / 255, blue: 0xE5 / 255)
    static let appName = "BigServerMonitor"
    static let tagline = "Infrastructure Monitoring"
}

/// The BigServerMonitor mark: indigo rounded square with a white pulse
/// waveform over three server-rack bars. Geometry mirrors
/// specs/design-handoff/assets/bigservermonitor-logo.svg (120×120 viewBox).
struct LogoMark: View {
    var size: CGFloat = 28

    var body: some View {
        let s = size / 120

        ZStack {
            RoundedRectangle(cornerRadius: 28 * s, style: .continuous)
                .fill(Brand.indigo)

            PulseLine()
                .stroke(.white, style: StrokeStyle(lineWidth: 3.5 * s, lineCap: .round, lineJoin: .round))

            ForEach(0..<3, id: \.self) { i in
                RoundedRectangle(cornerRadius: 3 * s)
                    .fill(.white.opacity([1.0, 0.72, 0.44][i]))
                    .frame(width: 76 * s, height: 10 * s)
                    .position(x: 60 * s, y: (CGFloat([62, 77, 92][i]) + 5) * s)
            }
        }
        .frame(width: size, height: size)
        .accessibilityLabel(Brand.appName)
    }

    private struct PulseLine: Shape {
        func path(in rect: CGRect) -> Path {
            let s = rect.width / 120
            let points: [(CGFloat, CGFloat)] = [(22, 38), (36, 38), (43, 22), (52, 54), (59, 38), (98, 38)]
            var p = Path()
            p.move(to: CGPoint(x: points[0].0 * s, y: points[0].1 * s))
            for pt in points.dropFirst() {
                p.addLine(to: CGPoint(x: pt.0 * s, y: pt.1 * s))
            }
            return p
        }
    }
}

/// "BigServerMonitor" wordmark — bold, tight tracking, per Logo.html.
struct Wordmark: View {
    var fontSize: CGFloat = 14

    var body: some View {
        Text(Brand.appName)
            .font(.system(size: fontSize, weight: .bold))
            .tracking(-0.02 * fontSize)
    }
}
