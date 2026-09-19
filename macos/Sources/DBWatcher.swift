import Foundation

public final class DBWatcher {
    private var sources: [DispatchSourceFileSystemObject] = []
    private var fileDescriptors: [Int32] = []
    private let queue = DispatchQueue(label: "com.cogni.dbwatcher", qos: .utility)
    private var onDatabaseChange: (() -> Void)?

    public init(onChange: @escaping () -> Void) {
        self.onDatabaseChange = onChange
        startWatching()
    }

    deinit {
        stopWatching()
    }

    public func startWatching() {
        stopWatching()

        let home = FileManager.default.homeDirectoryForCurrentUser.path
        let globalDir = "\(home)/.cogni"
        let globalDB = "\(globalDir)/memory.db"

        // Ensure directory exists so we can watch it even if DB is recreated
        try? FileManager.default.createDirectory(atPath: globalDir, withIntermediateDirectories: true)

        watchPath(globalDir)
        if FileManager.default.fileExists(atPath: globalDB) {
            watchPath(globalDB)
        }
    }

    private func watchPath(_ path: String) {
        let fd = open(path, O_EVTONLY)
        guard fd >= 0 else { return }

        fileDescriptors.append(fd)
        let source = DispatchSource.makeFileSystemObjectSource(
            fileDescriptor: fd,
            eventMask: [.write, .extend, .attrib, .link, .rename],
            queue: queue
        )

        source.setEventHandler { [weak self] in
            DispatchQueue.main.async {
                self?.onDatabaseChange?()
            }
        }

        source.setCancelHandler {
            close(fd)
        }

        source.resume()
        sources.append(source)
    }

    public func stopWatching() {
        for source in sources {
            source.cancel()
        }
        sources.removeAll()
        fileDescriptors.removeAll()
    }
}
