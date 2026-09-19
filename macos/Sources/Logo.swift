import AppKit

public enum CogniLogo {
    public enum PulseState: Equatable {
        case idle
        case activePulse
    }

    /// Renders the 20x18 Status Bar Icon
    public static func statusImage(pulse: PulseState = .idle) -> NSImage {
        let size = NSSize(width: 20, height: 18)
        let img = NSImage(size: size, flipped: false) { rect in
            guard let ctx = NSGraphicsContext.current?.cgContext else { return false }

            ctx.saveGState()

            let isPulsing = (pulse == .activePulse)
            let pulseColor = NSColor(red: 0.0, green: 0.898, blue: 1.0, alpha: 1.0) // electricCyan

            let strokeColor: NSColor = isPulsing ? pulseColor : .labelColor
            ctx.setStrokeColor(strokeColor.cgColor)
            ctx.setLineWidth(1.4)
            ctx.setLineCap(.round)
            ctx.setLineJoin(.round)

            // Left neural arc
            let leftArc = CGMutablePath()
            leftArc.addArc(center: CGPoint(x: 6.5, y: 9.0), radius: 4.5, startAngle: .pi * 0.4, endAngle: .pi * 1.6, clockwise: false)
            ctx.addPath(leftArc)
            ctx.strokePath()

            // Right neural arc
            let rightArc = CGMutablePath()
            rightArc.addArc(center: CGPoint(x: 13.5, y: 9.0), radius: 4.5, startAngle: -.pi * 0.6, endAngle: .pi * 0.6, clockwise: false)
            ctx.addPath(rightArc)
            ctx.strokePath()

            // Connective bridge
            ctx.move(to: CGPoint(x: 6.5, y: 9.0))
            ctx.addLine(to: CGPoint(x: 13.5, y: 9.0))
            ctx.strokePath()

            // Center synaptic nucleus
            let centerRect = CGRect(x: 8.5, y: 7.5, width: 3.0, height: 3.0)
            if isPulsing {
                ctx.setFillColor(pulseColor.cgColor)
                ctx.fillEllipse(in: centerRect.insetBy(dx: -1.0, dy: -1.0))

                // Outer glow ring
                ctx.setStrokeColor(pulseColor.withAlphaComponent(0.4).cgColor)
                ctx.setLineWidth(1.0)
                ctx.strokeEllipse(in: centerRect.insetBy(dx: -3.0, dy: -3.0))
            } else {
                ctx.setFillColor(NSColor.labelColor.cgColor)
                ctx.fillEllipse(in: centerRect)
            }

            ctx.restoreGState()
            return true
        }

        img.isTemplate = (pulse == .idle)
        return img
    }
}
