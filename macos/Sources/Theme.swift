import SwiftUI
import AppKit

// MARK: - Cogni Design Tokens
public struct CogniTheme {
    // Dark Palette Tokens
    public static let bgDeep = Color(hex: 0x0B0F17)        // Asfalto Profundo
    public static let bgCard = Color(hex: 0x111827)        // Card Surface
    public static let bgCardHover = Color(hex: 0x1A2234)   // Card Hover
    public static let border = Color(hex: 0x1E293B)        // Line Border
    public static let borderActive = Color(hex: 0x00E5FF).opacity(0.3)
    
    public static let electricCyan = Color(hex: 0x00E5FF)  // Active status / Token metrics / Synapse Pulse
    public static let cobaltRoyal = Color(hex: 0x0055FF)   // Secondary brand accent / Harnesses
    public static let volcanicOrange = Color(hex: 0xFF6B00)// Action / Alerts / Clean
    
    public static let textPrimary = Color(hex: 0xF8FAFC)   // Crisp White
    public static let textSecondary = Color(hex: 0x94A3B8) // Muted Info
    public static let textDim = Color(hex: 0x64748B)       // Dim Subtitle
}

extension Color {
    init(hex: UInt, alpha: Double = 1.0) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xff) / 255,
            green: Double((hex >> 08) & 0xff) / 255,
            blue: Double((hex >> 00) & 0xff) / 255,
            opacity: alpha
        )
    }
}
