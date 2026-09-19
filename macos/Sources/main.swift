import AppKit
import SwiftUI

@MainActor
public final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var popover: NSPopover!
    private var controller: CogniController!
    private var hotKey: GlobalHotKey!


    public func applicationDidFinishLaunching(_ notification: Notification) {
        controller = CogniController()

        // Configure Status Item
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        if let button = statusItem.button {
            button.image = CogniLogo.statusImage(pulse: .idle)
            button.target = self
            button.action = #selector(statusItemClicked(_:))
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }

        // Configure Popover with dynamic sizing matching SwiftUI intrinsic content
        let popover = NSPopover()
        let hosting = NSHostingController(
            rootView: ContentView(controller: controller, onQuit: {
                NSApp.terminate(nil)
            })
        )
        hosting.sizingOptions = [.preferredContentSize]
        popover.contentViewController = hosting
        popover.behavior = .transient
        popover.animates = true
        self.popover = popover

        // Register Global HotKey (⌥⌘C)
        hotKey = GlobalHotKey { [weak self] in
            self?.togglePopover()
        }
        hotKey.register()
    }

    @objc private func statusItemClicked(_ sender: NSStatusBarButton) {
        guard let event = NSApp.currentEvent else {
            togglePopover()
            return
        }

        if event.type == .rightMouseUp || event.modifierFlags.contains(.control) {
            showContextMenu(at: sender)
        } else {
            togglePopover()
        }
    }

    private func togglePopover() {
        guard let button = statusItem.button else { return }
        if popover.isShown {
            popover.performClose(nil)
        } else {
            controller.fetchConfig()
            let edge: NSRectEdge = button.isFlipped ? .maxY : .minY
            popover.show(relativeTo: button.bounds, of: button, preferredEdge: edge)
            if let window = popover.contentViewController?.view.window {
                window.makeKeyAndOrderFront(nil)
            }
            NSApp.activate(ignoringOtherApps: true)
        }
    }

    private func showContextMenu(at button: NSStatusBarButton) {
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

        let location = NSPoint(x: 0, y: button.bounds.height + 4)
        menu.popUp(positioning: nil, at: location, in: button)
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
