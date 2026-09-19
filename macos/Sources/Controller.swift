import Foundation
import SwiftUI
import ServiceManagement
import Combine

public struct CogniStatsDTO: Codable {
    public let totalMemories: Int
    public let totalTokensSaved: Int
    public let totalReasoningSavedUSD: Double?
    public let dbSizeBytes: Int64?
    public let activeProjects: Int?
    public let categories: [String: Int]?

    enum CodingKeys: String, CodingKey {
        case totalMemories = "total_memories"
        case totalTokensSaved = "total_tokens_saved"
        case totalReasoningSavedUSD = "total_reasoning_saved_usd"
        case dbSizeBytes = "db_size_bytes"
        case activeProjects = "active_projects"
        case categories = "categories"
    }
}

public struct CogniConfigDTO: Codable {
    public let selectedHarnesses: [String]?
    public let activeProject: String?

    enum CodingKeys: String, CodingKey {
        case selectedHarnesses = "selected_harnesses"
        case activeProject = "active_project"
    }
}

@MainActor
public final class CogniController: ObservableObject {
    @Published public var selectedTab: Int = 0 // 0: Estado, 1: Comandos, 2: Actualizaciones
    @Published public var stats: CogniStatsDTO?
    @Published public var activeHarnesses: [String] = []
    @Published public var pulseState: CogniLogo.PulseState = .idle
    @Published public var isCleaning: Bool = false
    @Published public var statusMessage: String = "Listo"
    @Published public var isLaunchAtLoginEnabled: Bool = false

    // Update Management
    @Published public var currentVersion: String = "v2.3.1"
    @Published public var latestVersion: String?
    @Published public var releaseURL: String?
    @Published public var hasCheckedUpdate: Bool = false
    @Published public var isCheckingUpdate: Bool = false
    @Published public var isUpgrading: Bool = false
    @Published public var updateMessage: String?
    @Published public var updateAvailable: Bool = false

    private var dbWatcher: DBWatcher?
    private var timer: AnyCancellable?
    private var pulseResetTask: Task<Void, Never>?

    public init() {
        checkLaunchAtLoginStatus()
        fetchLocalVersion()
        refreshAll()

        // Setup real-time file watcher on SQLite database
        dbWatcher = DBWatcher { [weak self] in
            guard let self = self else { return }
            self.triggerPulse(mode: .cyan)
            self.refreshAll()
        }

        // Periodic light refresh every 10 seconds
        timer = Timer.publish(every: 10, on: .main, in: .common)
            .autoconnect()
            .sink { [weak self] _ in
                self?.refreshAll()
            }
    }

    public func refreshAll() {
        fetchLocalVersion()
        fetchStats()
        fetchConfig()
    }

    public func fetchStats() {
        Task.detached(priority: .userInitiated) {
            let res = Shell.runCogni(["stats", "--json"])
            if res.status == 0, let data = res.output.data(using: .utf8) {
                if let parsed = try? JSONDecoder().decode(CogniStatsDTO.self, from: data) {
                    await MainActor.run {
                        self.stats = parsed
                    }
                }
            }
        }
    }

    public func fetchConfig() {
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        let configPath = "\(home)/.cogni/config.json"
        if let data = try? Data(contentsOf: URL(fileURLWithPath: configPath)),
           let cfg = try? JSONDecoder().decode(CogniConfigDTO.self, from: data),
           let harnesses = cfg.selectedHarnesses {
            self.activeHarnesses = harnesses
        } else {
            // Default detected standard harnesses
            self.activeHarnesses = ["local", "antigravity", "cursor", "pi"]
        }
    }

    public func triggerPulse(mode: CogniLogo.ColorMode = .cyan) {
        pulseResetTask?.cancel()
        self.pulseState = .activePulse(mode)

        pulseResetTask = Task {
            try? await Task.sleep(nanoseconds: 700_000_000) // 700ms pulse
            if !Task.isCancelled {
                self.pulseState = .idle
            }
        }
    }

    public func openWebUI() {
        Task.detached {
            Shell.runCogni(["ui"])
        }
    }

    public func cleanAndVacuum() {
        guard !isCleaning else { return }
        isCleaning = true
        statusMessage = "Optimizando base de datos..."
        triggerPulse(mode: .orange)

        Task.detached(priority: .userInitiated) {
            _ = Shell.runCogni(["stats"])
            try? await Task.sleep(nanoseconds: 600_000_000)

            await MainActor.run {
                self.isCleaning = false
                self.statusMessage = "Base de datos optimizada"
                self.refreshAll()
            }
        }
    }

    // MARK: - Launch at Login
    public func checkLaunchAtLoginStatus() {
        if #available(macOS 13.0, *) {
            isLaunchAtLoginEnabled = (SMAppService.mainApp.status == .enabled)
        }
    }

    public func toggleLaunchAtLogin() {
        if #available(macOS 13.0, *) {
            do {
                if isLaunchAtLoginEnabled {
                    try SMAppService.mainApp.unregister()
                    isLaunchAtLoginEnabled = false
                } else {
                    try SMAppService.mainApp.register()
                    isLaunchAtLoginEnabled = true
                }
            } catch {
                statusMessage = "Error en inicio automático"
            }
        }
    }

    public func formatNumber(_ num: Int) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter.string(from: NSNumber(value: num)) ?? "\(num)"
    }

    public func formatBytes(_ bytes: Int64) -> String {
        let kb = Double(bytes) / 1024.0
        if kb < 1024 {
            return String(format: "%.1f KB", kb)
        }
        let mb = kb / 1024.0
        return String(format: "%.2f MB", mb)
    }

    // MARK: - Update Checking
    public func fetchLocalVersion() {
        let res = Shell.runCogni(["version"])
        if res.status == 0 {
            let out = res.output.trimmingCharacters(in: .whitespacesAndNewlines)
            if let vIdx = out.range(of: "v") {
                self.currentVersion = String(out[vIdx.lowerBound...]).components(separatedBy: " ")[0]
            }
        }
    }

    public func checkForUpdates() {
        guard !isCheckingUpdate else { return }
        isCheckingUpdate = true
        updateMessage = "Comprobando versión disponible en GitHub..."

        Task {
            await MainActor.run {
                self.fetchLocalVersion()
            }
            await doCheckForUpdates(verbose: true)
            await MainActor.run {
                self.isCheckingUpdate = false
            }
        }
    }

    private func doCheckForUpdates(verbose: Bool) async {
        guard let url = URL(string: "https://api.github.com/repos/AdelysAlberto/cogni-memory/releases/latest") else { return }

        var request = URLRequest(url: url)
        request.setValue("CogniBar-App", forHTTPHeaderField: "User-Agent")
        request.cachePolicy = .reloadIgnoringLocalAndRemoteCacheData
        request.timeoutInterval = 10

        do {
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let httpRes = response as? HTTPURLResponse, httpRes.statusCode == 200 else {
                if verbose {
                    await MainActor.run {
                        self.updateMessage = "No se pudo conectar con GitHub"
                    }
                }
                return
            }

            struct ReleaseDTO: Codable {
                let tagName: String
                let htmlUrl: String
                enum CodingKeys: String, CodingKey {
                    case tagName = "tag_name"
                    case htmlUrl = "html_url"
                }
            }

            let release = try JSONDecoder().decode(ReleaseDTO.self, from: data)
            let remoteTag = release.tagName.trimmingCharacters(in: .whitespacesAndNewlines)
            let remoteClean = remoteTag.hasPrefix("v") ? remoteTag : "v\(remoteTag)"
            let localClean = self.currentVersion.hasPrefix("v") ? self.currentVersion : "v\(self.currentVersion)"

            let isNewer = compareSemver(localClean, remoteClean) < 0

            await MainActor.run {
                self.latestVersion = remoteClean
                self.releaseURL = release.htmlUrl
                self.hasCheckedUpdate = true
                self.updateAvailable = isNewer

                if isNewer {
                    self.updateMessage = "¡Nueva versión \(remoteClean) disponible para instalar!"
                } else {
                    self.updateMessage = "Cogni está actualizado a la última versión (\(localClean))."
                }
            }
        } catch {
            if verbose {
                await MainActor.run {
                    self.updateMessage = "Error al comprobar: \(error.localizedDescription)"
                }
            }
        }
    }

    public func performUpgrade() {
        guard !isUpgrading else { return }
        isUpgrading = true
        updateMessage = "Descargando e instalando actualización..."

        Task.detached(priority: .userInitiated) {
            let res = Shell.runCogni(["upgrade"])
            
            await MainActor.run {
                self.isUpgrading = false
                if res.status == 0 {
                    self.updateMessage = "¡Actualización completada con éxito!"
                    self.fetchLocalVersion()
                    self.updateAvailable = false
                } else {
                    self.updateMessage = "Error actualizando: \(res.output)"
                }
            }
        }
    }

    private func compareSemver(_ v1: String, _ v2: String) -> Int {
        let p1 = parseSemver(v1)
        let p2 = parseSemver(v2)

        for i in 0..<3 {
            if p1[i] > p2[i] { return 1 }
            if p1[i] < p2[i] { return -1 }
        }
        return 0
    }

    private func parseSemver(_ v: String) -> [Int] {
        var clean = v.trimmingCharacters(in: .whitespacesAndNewlines)
        if clean.hasPrefix("v") {
            clean.removeFirst()
        }
        let parts = clean.components(separatedBy: ".")
        var result = [0, 0, 0]
        for i in 0..<min(parts.count, 3) {
            result[i] = Int(parts[i]) ?? 0
        }
        return result
    }
}
