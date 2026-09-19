import SwiftUI

public struct ContentView: View {
    @ObservedObject var controller: CogniController
    var onQuit: () -> Void

    public init(controller: CogniController, onQuit: @escaping () -> Void) {
        self.controller = controller
        self.onQuit = onQuit
    }

    public var body: some View {
        VStack(spacing: 12) {
            // MARK: - Header
            headerView

            // MARK: - Hero Card: Token Savings & Actions
            heroCardView

            // MARK: - Storage & Memory Breakdown
            storageCardView

            // MARK: - Active Harnesses
            harnessesCardView

            // MARK: - Footer
            footerView
        }
        .padding(14)
        .frame(width: 320)
        .background(CogniTheme.bgDeep)
        .foregroundColor(CogniTheme.textPrimary)
    }

    // MARK: - Header View
    private var headerView: some View {
        HStack(alignment: .center, spacing: 8) {
            // Neural Icon
            ZStack {
                Circle()
                    .fill(CogniTheme.electricCyan.opacity(0.12))
                    .frame(width: 28, height: 28)
                Image(nsImage: CogniLogo.statusImage(pulse: controller.pulseState))
                    .renderingMode(.template)
                    .foregroundColor(CogniTheme.electricCyan)
            }

            VStack(alignment: .leading, spacing: 1) {
                Text("Cogni Memory")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundColor(CogniTheme.textPrimary)
                Text("Universal Agent Engine")
                    .font(.system(size: 10, weight: .medium))
                    .foregroundColor(CogniTheme.textDim)
            }

            Spacer()

            // Active Pill
            HStack(spacing: 4) {
                Circle()
                    .fill(CogniTheme.electricCyan)
                    .frame(width: 6, height: 6)
                Text("ACTIVO")
                    .font(.system(size: 9, weight: .bold))
                    .foregroundColor(CogniTheme.electricCyan)
            }
            .padding(.horizontal, 7)
            .padding(.vertical, 3)
            .background(CogniTheme.electricCyan.opacity(0.12))
            .cornerRadius(12)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(CogniTheme.electricCyan.opacity(0.25), lineWidth: 1)
            )
        }
    }

    // MARK: - Hero Card
    private var heroCardView: some View {
        VStack(spacing: 10) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("TOKENS AHORRADOS")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundColor(CogniTheme.textDim)
                    
                    let tokens = controller.stats?.totalTokensSaved ?? 0
                    Text(controller.formatNumber(tokens))
                        .font(.system(size: 20, weight: .heavy, design: .rounded))
                        .foregroundColor(CogniTheme.electricCyan)
                }
                Spacer()

                if let usd = controller.stats?.totalReasoningSavedUSD, usd > 0 {
                    VStack(alignment: .trailing, spacing: 2) {
                        Text("RAZONAMIENTO")
                            .font(.system(size: 9, weight: .bold))
                            .foregroundColor(CogniTheme.textDim)
                        Text(String(format: "$%.3f", usd))
                            .font(.system(size: 14, weight: .bold, design: .monospaced))
                            .foregroundColor(CogniTheme.textPrimary)
                    }
                }
            }

            Divider()
                .background(CogniTheme.border)

            // Action Buttons
            HStack(spacing: 8) {
                Button(action: {
                    controller.openWebUI()
                }) {
                    HStack(spacing: 5) {
                        Image(systemName: "globe")
                            .font(.system(size: 11, weight: .semibold))
                        Text("Abrir Web UI")
                            .font(.system(size: 11, weight: .semibold))
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 6)
                    .background(CogniTheme.electricCyan.opacity(0.15))
                    .foregroundColor(CogniTheme.electricCyan)
                    .cornerRadius(6)
                    .overlay(
                        RoundedRectangle(cornerRadius: 6)
                            .stroke(CogniTheme.electricCyan.opacity(0.3), lineWidth: 1)
                    )
                }
                .buttonStyle(PlainButtonStyle())

                Button(action: {
                    controller.cleanAndVacuum()
                }) {
                    HStack(spacing: 5) {
                        Image(systemName: "sparkles")
                            .font(.system(size: 11, weight: .semibold))
                        Text(controller.isCleaning ? "Limpiando..." : "Optimizar")
                            .font(.system(size: 11, weight: .semibold))
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 6)
                    .background(CogniTheme.volcanicOrange.opacity(0.15))
                    .foregroundColor(CogniTheme.volcanicOrange)
                    .cornerRadius(6)
                    .overlay(
                        RoundedRectangle(cornerRadius: 6)
                            .stroke(CogniTheme.volcanicOrange.opacity(0.3), lineWidth: 1)
                    )
                }
                .buttonStyle(PlainButtonStyle())
                .disabled(controller.isCleaning)
            }
        }
        .padding(10)
        .background(CogniTheme.bgCard)
        .cornerRadius(10)
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(CogniTheme.border, lineWidth: 1)
        )
    }

    // MARK: - Storage & Memory Breakdown
    private var storageCardView: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Base de Datos")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundColor(CogniTheme.textSecondary)
                Spacer()
                if let sizeBytes = controller.stats?.dbSizeBytes {
                    Text(controller.formatBytes(sizeBytes))
                        .font(.system(size: 10, weight: .medium, design: .monospaced))
                        .foregroundColor(CogniTheme.textDim)
                }
            }

            HStack(spacing: 12) {
                let total = controller.stats?.totalMemories ?? 0
                VStack(alignment: .leading, spacing: 1) {
                    Text("\(total)")
                        .font(.system(size: 15, weight: .bold, design: .rounded))
                        .foregroundColor(CogniTheme.textPrimary)
                    Text("Firmas activas")
                        .font(.system(size: 9, weight: .medium))
                        .foregroundColor(CogniTheme.textDim)
                }

                Spacer()

                if let cats = controller.stats?.categories, !cats.isEmpty {
                    HStack(spacing: 4) {
                        ForEach(Array(cats.keys.prefix(3)), id: \.self) { cat in
                            Text(cat)
                                .font(.system(size: 8, weight: .bold))
                                .padding(.horizontal, 5)
                                .padding(.vertical, 2)
                                .background(CogniTheme.cobaltRoyal.opacity(0.18))
                                .foregroundColor(CogniTheme.textSecondary)
                                .cornerRadius(4)
                        }
                    }
                }
            }
        }
        .padding(10)
        .background(CogniTheme.bgCard)
        .cornerRadius(10)
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(CogniTheme.border, lineWidth: 1)
        )
    }

    // MARK: - Active Harnesses
    private var harnessesCardView: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Arneses de IA Conectados")
                .font(.system(size: 10, weight: .bold))
                .foregroundColor(CogniTheme.textSecondary)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 5) {
                    ForEach(controller.activeHarnesses, id: \.self) { h in
                        HStack(spacing: 4) {
                            Circle()
                                .fill(CogniTheme.electricCyan)
                                .frame(width: 4, height: 4)
                            Text(h.capitalized)
                                .font(.system(size: 9, weight: .semibold))
                                .foregroundColor(CogniTheme.textPrimary)
                        }
                        .padding(.horizontal, 7)
                        .padding(.vertical, 3)
                        .background(CogniTheme.bgCardHover)
                        .cornerRadius(6)
                        .overlay(
                            RoundedRectangle(cornerRadius: 6)
                                .stroke(CogniTheme.border, lineWidth: 1)
                        )
                    }
                }
            }
        }
        .padding(10)
        .background(CogniTheme.bgCard)
        .cornerRadius(10)
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(CogniTheme.border, lineWidth: 1)
        )
    }

    // MARK: - Footer View
    private var footerView: some View {
        HStack {
            // Global Hotkey Badge
            HStack(spacing: 3) {
                Text("Atajo:")
                    .font(.system(size: 9, weight: .medium))
                    .foregroundColor(CogniTheme.textDim)
                Text("⌥⌘C")
                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                    .padding(.horizontal, 4)
                    .padding(.vertical, 1)
                    .background(CogniTheme.border)
                    .foregroundColor(CogniTheme.textSecondary)
                    .cornerRadius(3)
            }

            Spacer()

            // Quit Button
            Button(action: onQuit) {
                Text("Salir")
                    .font(.system(size: 10, weight: .medium))
                    .foregroundColor(CogniTheme.textDim)
            }
            .buttonStyle(PlainButtonStyle())
        }
        .padding(.horizontal, 4)
        .padding(.top, 2)
    }
}
