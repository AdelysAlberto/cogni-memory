import SwiftUI
import AppKit

public struct ContentView: View {
    @ObservedObject var controller: CogniController
    var onQuit: () -> Void

    @State private var copiedCommand: String?

    public init(controller: CogniController, onQuit: @escaping () -> Void) {
        self.controller = controller
        self.onQuit = onQuit
    }

    public var body: some View {
        VStack(spacing: 10) {
            // MARK: - Header
            headerView

            // MARK: - Navigation Tabs
            navigationTabsView

            // MARK: - Tab Content
            Group {
                switch controller.selectedTab {
                case 0:
                    statusTabView
                case 1:
                    commandsTabView
                case 2:
                    updatesTabView
                default:
                    statusTabView
                }
            }
            .transition(.opacity)

            // MARK: - Footer
            footerView
        }
        .padding(12)
        .frame(width: 320)
        .background(CogniTheme.bgDeep)
        .foregroundColor(CogniTheme.textPrimary)
        .animation(.easeInOut(duration: 0.18), value: controller.selectedTab)
    }

    // MARK: - Header View
    private var headerView: some View {
        HStack(alignment: .center, spacing: 8) {
            ZStack {
                Circle()
                    .fill(CogniTheme.electricCyan.opacity(0.12))
                    .frame(width: 26, height: 26)
                Image(nsImage: CogniLogo.statusImage(pulse: .idle))
                    .renderingMode(.template)
                    .foregroundColor(CogniTheme.electricCyan)
            }

            VStack(alignment: .leading, spacing: 1) {
                HStack(spacing: 5) {
                    Text("Cogni Memory")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundColor(CogniTheme.textPrimary)
                    Text(controller.currentVersion)
                        .font(.system(size: 9, weight: .semibold, design: .monospaced))
                        .padding(.horizontal, 4)
                        .padding(.vertical, 1)
                        .background(CogniTheme.border)
                        .foregroundColor(CogniTheme.textSecondary)
                        .cornerRadius(3)
                }
                Text("Universal Agent Engine")
                    .font(.system(size: 9.5, weight: .medium))
                    .foregroundColor(CogniTheme.textDim)
            }

            Spacer()

            // Active Pill
            HStack(spacing: 4) {
                Circle()
                    .fill(CogniTheme.electricCyan)
                    .frame(width: 5, height: 5)
                Text("ACTIVO")
                    .font(.system(size: 8.5, weight: .bold))
                    .foregroundColor(CogniTheme.electricCyan)
            }
            .padding(.horizontal, 6)
            .padding(.vertical, 2.5)
            .background(CogniTheme.electricCyan.opacity(0.12))
            .cornerRadius(10)
            .overlay(
                RoundedRectangle(cornerRadius: 10)
                    .stroke(CogniTheme.electricCyan.opacity(0.25), lineWidth: 1)
            )
        }
    }

    // MARK: - Navigation Tabs
    private var navigationTabsView: some View {
        HStack(spacing: 4) {
            tabButton(title: "Estado", icon: "chart.bar.fill", index: 0)
            tabButton(title: "Comandos", icon: "terminal.fill", index: 1)
            tabButton(title: "Actualizar", icon: "arrow.triangle.2.circlepath", index: 2, hasBadge: controller.updateAvailable)
        }
        .padding(3)
        .background(CogniTheme.bgCard)
        .cornerRadius(8)
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(CogniTheme.border, lineWidth: 1)
        )
    }

    private func tabButton(title: String, icon: String, index: Int, hasBadge: Bool = false) -> some View {
        let isSelected = controller.selectedTab == index
        return Button(action: {
            controller.selectedTab = index
        }) {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 10, weight: .semibold))
                Text(title)
                    .font(.system(size: 10, weight: .semibold))

                if hasBadge {
                    Circle()
                        .fill(CogniTheme.electricCyan)
                        .frame(width: 5, height: 5)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 5)
            .background(isSelected ? CogniTheme.bgCardHover : Color.clear)
            .foregroundColor(isSelected ? CogniTheme.electricCyan : CogniTheme.textSecondary)
            .cornerRadius(6)
        }
        .buttonStyle(PlainButtonStyle())
    }

    // MARK: - TAB 0: Estado
    private var statusTabView: some View {
        VStack(spacing: 8) {
            brandHeroCardView
            actionsCardView
            harnessesCardView
        }
    }

    private var brandHeroCardView: some View {
        HStack(spacing: 10) {
            ZStack {
                RoundedRectangle(cornerRadius: 8)
                    .fill(
                        LinearGradient(
                            colors: [CogniTheme.electricCyan.opacity(0.16), CogniTheme.cobaltRoyal.opacity(0.16)],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
                    .frame(width: 36, height: 36)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(CogniTheme.electricCyan.opacity(0.3), lineWidth: 1)
                    )
                Image(nsImage: CogniLogo.statusImage(pulse: .idle))
                    .renderingMode(.template)
                    .foregroundColor(CogniTheme.electricCyan)
            }

            VStack(alignment: .leading, spacing: 2) {
                Text("Memoria Contextual Activa")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundColor(CogniTheme.textPrimary)

                Text("Indexación BM25 + Recuerdo Semántico")
                    .font(.system(size: 9, weight: .medium))
                    .foregroundColor(CogniTheme.textDim)

                if let project = controller.activeProject, !project.isEmpty {
                    HStack(spacing: 4) {
                        Text("Proyecto:")
                            .font(.system(size: 8.5, weight: .medium))
                            .foregroundColor(CogniTheme.textDim)
                        Text(project)
                            .font(.system(size: 9, weight: .semibold, design: .monospaced))
                            .foregroundColor(CogniTheme.electricCyan)
                            .lineLimit(1)
                    }
                    .padding(.top, 1)
                }
            }

            Spacer()
        }
        .padding(10)
        .background(CogniTheme.bgCard)
        .cornerRadius(10)
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(CogniTheme.border, lineWidth: 1)
        )
    }

    private var actionsCardView: some View {
        HStack(spacing: 8) {
            Button(action: { controller.openWebUI() }) {
                HStack(spacing: 5) {
                    Image(systemName: "globe")
                        .font(.system(size: 11, weight: .bold))
                    Text("Abrir Web UI")
                        .font(.system(size: 11, weight: .bold))
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 7)
                .background(CogniTheme.electricCyan.opacity(0.15))
                .foregroundColor(CogniTheme.electricCyan)
                .cornerRadius(7)
                .overlay(
                    RoundedRectangle(cornerRadius: 7)
                        .stroke(CogniTheme.electricCyan.opacity(0.35), lineWidth: 1)
                )
            }
            .buttonStyle(PlainButtonStyle())

            Button(action: { controller.cleanAndVacuum() }) {
                HStack(spacing: 5) {
                    Image(systemName: "sparkles")
                        .font(.system(size: 11, weight: .bold))
                    Text(controller.isCleaning ? "Limpiando..." : "Optimizar")
                        .font(.system(size: 11, weight: .bold))
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 7)
                .background(CogniTheme.volcanicOrange.opacity(0.15))
                .foregroundColor(CogniTheme.volcanicOrange)
                .cornerRadius(7)
                .overlay(
                    RoundedRectangle(cornerRadius: 7)
                        .stroke(CogniTheme.volcanicOrange.opacity(0.35), lineWidth: 1)
                )
            }
            .buttonStyle(PlainButtonStyle())
            .disabled(controller.isCleaning)
        }
    }

    private var harnessesCardView: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("ARNESES DE IA CONECTADOS")
                .font(.system(size: 8.5, weight: .bold))
                .foregroundColor(CogniTheme.textDim)

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
                        .padding(.vertical, 3.5)
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
        .padding(9)
        .background(CogniTheme.bgCard)
        .cornerRadius(10)
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(CogniTheme.border, lineWidth: 1)
        )
    }

    // MARK: - TAB 1: Comandos
    private var commandsTabView: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("COMANDOS FRECUENTES")
                .font(.system(size: 9, weight: .bold))
                .foregroundColor(CogniTheme.textDim)

            VStack(spacing: 6) {
                commandRow(name: "Buscar memoria (BM25)", cmd: "cogni search \"<query>\"")
                commandRow(name: "Recuperar contexto activo", cmd: "cogni context")
                commandRow(name: "Guardar memoria estructurada", cmd: "cogni save --title \"...\" --what \"...\"")
                commandRow(name: "Abrir Dashboard Web", cmd: "cogni ui")
                commandRow(name: "Configurar arneses de IA", cmd: "cogni init")
                commandRow(name: "Actualizar Cogni CLI", cmd: "cogni upgrade")
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

    private func commandRow(name: String, cmd: String) -> some View {
        let isCopied = (copiedCommand == cmd)
        return HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(name)
                    .font(.system(size: 9, weight: .medium))
                    .foregroundColor(CogniTheme.textSecondary)
                Text(cmd)
                    .font(.system(size: 10, weight: .semibold, design: .monospaced))
                    .foregroundColor(CogniTheme.electricCyan)
                    .lineLimit(1)
            }

            Spacer()

            Button(action: {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(cmd, forType: .string)
                copiedCommand = cmd
                DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
                    if copiedCommand == cmd { copiedCommand = nil }
                }
            }) {
                Image(systemName: isCopied ? "checkmark.circle.fill" : "doc.on.doc")
                    .font(.system(size: 11))
                    .foregroundColor(isCopied ? CogniTheme.electricCyan : CogniTheme.textDim)
                    .padding(5)
                    .background(CogniTheme.bgCardHover)
                    .cornerRadius(5)
            }
            .buttonStyle(PlainButtonStyle())
        }
        .padding(.vertical, 3)
        .padding(.horizontal, 6)
        .background(CogniTheme.bgDeep.opacity(0.6))
        .cornerRadius(6)
    }

    // MARK: - TAB 2: Actualizaciones
    private var updatesTabView: some View {
        VStack(spacing: 12) {
            VStack(spacing: 6) {
                ZStack {
                    Circle()
                        .fill(controller.updateAvailable ? CogniTheme.electricCyan.opacity(0.15) : CogniTheme.cobaltRoyal.opacity(0.15))
                        .frame(width: 40, height: 40)
                    Image(systemName: controller.updateAvailable ? "arrow.down.circle.fill" : (controller.hasCheckedUpdate ? "checkmark.seal.fill" : "arrow.triangle.2.circlepath"))
                        .font(.system(size: 20))
                        .foregroundColor(controller.updateAvailable ? CogniTheme.electricCyan : CogniTheme.cobaltRoyal)
                }

                Text(controller.updateAvailable ? "¡Nueva Versión Disponible!" : (controller.hasCheckedUpdate ? "Cogni está al día" : "Comprobación de Versión"))
                    .font(.system(size: 13, weight: .bold))
                    .foregroundColor(CogniTheme.textPrimary)

                HStack(spacing: 12) {
                    VStack {
                        Text("Versión Actual")
                            .font(.system(size: 9, weight: .medium))
                            .foregroundColor(CogniTheme.textDim)
                        Text(controller.currentVersion)
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundColor(CogniTheme.textSecondary)
                    }

                    if let latest = controller.latestVersion {
                        Image(systemName: "arrow.right")
                            .font(.system(size: 10))
                            .foregroundColor(CogniTheme.textDim)

                        VStack {
                            Text("Última Versión")
                                .font(.system(size: 9, weight: .medium))
                                .foregroundColor(CogniTheme.textDim)
                            Text(latest)
                                .font(.system(size: 11, weight: .bold, design: .monospaced))
                                .foregroundColor(controller.updateAvailable ? CogniTheme.electricCyan : CogniTheme.textSecondary)
                        }
                    }
                }
                .padding(.vertical, 6)
                .padding(.horizontal, 12)
                .background(CogniTheme.bgDeep.opacity(0.6))
                .cornerRadius(6)
            }

            if let msg = controller.updateMessage {
                Text(msg)
                    .font(.system(size: 10, weight: .medium))
                    .foregroundColor(controller.updateAvailable ? CogniTheme.electricCyan : CogniTheme.textDim)
                    .multilineTextAlignment(.center)
            } else if !controller.hasCheckedUpdate {
                Text("Presiona \"Comprobar Actualización\" para verificar si existe una nueva versión en GitHub.")
                    .font(.system(size: 10, weight: .medium))
                    .foregroundColor(CogniTheme.textDim)
                    .multilineTextAlignment(.center)
            }

            // Action Buttons
            if controller.updateAvailable {
                VStack(spacing: 6) {
                    Button(action: { controller.performUpgrade() }) {
                        HStack(spacing: 6) {
                            if controller.isUpgrading {
                                ProgressView()
                                    .controlSize(.small)
                            } else {
                                Image(systemName: "arrow.down.to.line")
                                    .font(.system(size: 11, weight: .bold))
                            }
                            Text(controller.isUpgrading ? "Actualizando..." : "Actualizar a \(controller.latestVersion ?? "nueva versión")")
                                .font(.system(size: 11, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 8)
                        .background(CogniTheme.electricCyan)
                        .foregroundColor(CogniTheme.bgDeep)
                        .cornerRadius(6)
                    }
                    .buttonStyle(PlainButtonStyle())
                    .disabled(controller.isUpgrading)

                    Button(action: { controller.checkForUpdates() }) {
                        Text("Volver a Comprobar")
                            .font(.system(size: 9, weight: .medium))
                            .foregroundColor(CogniTheme.textDim)
                    }
                    .buttonStyle(PlainButtonStyle())
                    .disabled(controller.isCheckingUpdate || controller.isUpgrading)
                }
            } else {
                Button(action: { controller.checkForUpdates() }) {
                    HStack(spacing: 5) {
                        if controller.isCheckingUpdate {
                            ProgressView()
                                .controlSize(.small)
                        } else {
                            Image(systemName: "arrow.clockwise")
                                .font(.system(size: 11, weight: .semibold))
                        }
                        Text(controller.isCheckingUpdate ? "Comprobando..." : "Comprobar Actualización")
                            .font(.system(size: 11, weight: .semibold))
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 7)
                    .background(CogniTheme.bgCardHover)
                    .foregroundColor(CogniTheme.textPrimary)
                    .cornerRadius(6)
                    .overlay(
                        RoundedRectangle(cornerRadius: 6)
                            .stroke(CogniTheme.border, lineWidth: 1)
                    )
                }
                .buttonStyle(PlainButtonStyle())
                .disabled(controller.isCheckingUpdate)
            }

            if let urlStr = controller.releaseURL, let url = URL(string: urlStr) {
                Link(destination: url) {
                    Text("Ver notas de la versión en GitHub →")
                        .font(.system(size: 9, weight: .medium))
                        .foregroundColor(CogniTheme.textDim)
                }
            }
        }
        .padding(12)
        .background(CogniTheme.bgCard)
        .cornerRadius(10)
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(CogniTheme.border, lineWidth: 1)
        )
    }

    // MARK: - Footer
    private var footerView: some View {
        HStack {
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
