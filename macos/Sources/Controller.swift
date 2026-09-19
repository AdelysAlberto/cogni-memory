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
    @Published public var stats: CogniStatsDTO?
    @Published public var activeHarnesses: [String] = []
    @Published public var pulseState: CogniLogo.PulseState = .idle
    @Published public var isCleaning: Bool = false
    @Published public var statusMessage: String = "Listo"
    @Published public var isLaunchAtLoginEnabled: Bool = false

    private var dbWatcher: DBWatcher?
    private var timer: AnyCancellable?
    private var pulseResetTask: Task<Void, Never>?

    public init() {
        checkLaunchAtLoginStatus()
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
}
