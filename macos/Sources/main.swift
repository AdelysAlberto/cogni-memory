import AppKit
import SwiftUI
import Combine

@MainActor
public final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var popover: NSPopover!
    private var controller: CogniController!
    private var hotKey: GlobalHotKey!
    private var cancellables = Set<AnyCancellable>()

    public func applicationDidFinishLaunching(_ notification: Notification) {
        controller = CogniController()

        // Configure Status Item
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        if let button = statusItem.button {
            button.image = CogniLogo.statusImage(pulse: .idle)
            button.target = self
            button.action = #selector(statusItemClicked)
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }

        // Configure Popover
        let popover = NSPopover()
        popover.contentSize = NSSize(width: 320, height: 360)
        popover.behavior = .transient
        popover.animates = true
        popover.contentViewController = NSHostingController(
            rootView: ContentView(controller: controller, onQuit: {
                NSApp.terminate(nil)
            })
        )
        self.popover = popover

        // Observe pulse state to update status item icon
        controller.$pulseState
            .sink { [weak self] state in
                guard let self = self, let button = self.statusItem.button else { return }
                button.image = CogniLogo.statusImage(pulse: state)
            }
            .store(in: &cancellables)

        // Register Global HotKey (⌥⌘C)
        hotKey = GlobalHotKey { [weak self] in
            self?.togglePopover()
        }
        hotKey.register()
    }

    @objc private func statusItemClicked() {
        guard let event = NSApp.currentEvent else { return }
        if event.type == .rightMouseUp {
            showContextMenu()
        } else {
            togglePopover()
        }
    }

    private func togglePopover() {
        guard let button = statusItem.button else { return }
        if popover.isShown {
            popover.performClose(nil)
        } else {
            controller.refreshAll()
            popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
            popover.contentViewController?.view.window?.makeKey()
        }
    }

    private func showContextMenu() {
        let menu = NSMenu()

        let headerItem = NSMenuItem(title: "Cogni Memory Engine", action: nil, keyEquivalent: "")
        headerItem.isEnabled = false
        menu.addItem(headerItem)
        menu.addItem(NSMenuItem.separator())

        let openUIItem = NSMenuItem(title: "Abrir Web UI", action: #selector(contextOpenUI), keyEquivalent: "u")
        openUIItem.target = self
        menu.addItem(openUIItem)

        let cleanItem = NSMenuItem(title: "Optimizar Base de Datos", action: #selector(contextClean), keyEquivalent: "o")
        cleanItem.target = self
        menu.addItem(cleanItem)

        menu.addItem(NSMenuItem.separator())

        let loginItem = NSMenuItem(
            title: "Abrir al Iniciar Sesión",
            action: #selector(contextToggleLaunchAtLogin),
            keyEquivalent: ""
        )
        loginItem.target = self
        loginItem.state = controller.isLaunchAtLoginEnabled ? .on : .off
        menu.addItem(loginItem)

        menu.addItem(NSMenuItem.separator())

        let quitItem = NSMenuItem(title: "Salir de Cogni", action: #selector(contextQuit), keyEquivalent: "q")
        quitItem.target = self
        menu.addItem(quitItem)

        statusItem.menu = menu
        statusItem.button?.performClick(nil)
        statusItem.menu = nil // Restore left click behavior
    }

    @objc private func contextOpenUI() {
        controller.openWebUI()
    }

    @objc private func contextClean() {
        controller.cleanAndVacuum()
    }

    @objc private func contextToggleLaunchAtLogin() {
        controller.toggleLaunchAtLogin()
    }

    @objc private func contextQuit() {
        NSApp.terminate(nil)
    }
}

// MARK: - Main Entry Point
@main
struct CogniBarApp {
    @MainActor
    static func main() {
        let app = NSApplication.shared
        let delegate = AppDelegate()
        app.delegate = delegate
        app.setActivationPolicy(.accessory) // Runs purely in Menu Bar without Dock icon
        app.run()
    }
}
