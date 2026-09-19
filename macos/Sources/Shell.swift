import Foundation

public enum Shell {
    public struct Result {
        public let status: Int32
        public let output: String
    }

    /// Resolves the absolute path to the cogni binary
    public static func resolveCogniPath() -> String {
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        let candidates = [
            "\(home)/.local/bin/cogni",
            "/usr/local/bin/cogni",
            "/opt/homebrew/bin/cogni",
            "/usr/bin/cogni"
        ]

        for path in candidates {
            if FileManager.default.isExecutableFile(atPath: path) {
                return path
            }
        }

        // Fallback to searching PATH via which
        let whichRes = run("/usr/bin/which", ["cogni"])
        if whichRes.status == 0 && !whichRes.output.isEmpty {
            return whichRes.output
        }

        return "\(home)/.local/bin/cogni"
    }

    @discardableResult
    public static func run(_ tool: String, _ args: [String]) -> Result {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: tool)
        process.arguments = args
        
        var env = ProcessInfo.processInfo.environment
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        env["PATH"] = "\(home)/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
        process.environment = env

        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = pipe

        do {
            try process.run()
        } catch {
            return Result(status: -1, output: error.localizedDescription)
        }

        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()

        let output = String(decoding: data, as: UTF8.self).trimmingCharacters(in: .whitespacesAndNewlines)
        return Result(status: process.terminationStatus, output: output)
    }

    public static func runCogni(_ args: [String]) -> Result {
        let cogniBin = resolveCogniPath()
        return run(cogniBin, args)
    }
}
