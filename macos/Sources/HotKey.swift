import AppKit
import Carbon.HIToolbox

public final class GlobalHotKey {
    private var hotKeyRef: EventHotKeyRef?
    private var eventHandler: EventHandlerRef?
    private let onTrigger: () -> Void

    public init(onTrigger: @escaping () -> Void) {
        self.onTrigger = onTrigger
    }

    deinit {
        unregister()
    }

    @discardableResult
    public func register() -> Bool {
        unregister()

        var eventType = EventTypeSpec(
            eventClass: OSType(kEventClassKeyboard),
            eventKind: UInt32(kEventHotKeyPressed)
        )

        let selfPtr = Unmanaged.passUnretained(self).toOpaque()

        let installStatus = InstallEventHandler(
            GetApplicationEventTarget(),
            { _, event, userData in
                guard let userData = userData else { return noErr }
                let hotKey = Unmanaged<GlobalHotKey>.fromOpaque(userData).takeUnretainedValue()
                DispatchQueue.main.async {
                    hotKey.onTrigger()
                }
                return noErr
            },
            1,
            &eventType,
            selfPtr,
            &eventHandler
        )

        guard installStatus == noErr else { return false }

        // Key: 'C' (kVK_ANSI_C = 0x08), Modifiers: Option + Command
        let hotKeyID = EventHotKeyID(signature: OSType(0x434F474E), id: 1) // 'COGN'
        let regStatus = RegisterEventHotKey(
            UInt32(kVK_ANSI_C),
            UInt32(cmdKey | optionKey),
            hotKeyID,
            GetApplicationEventTarget(),
            0,
            &hotKeyRef
        )

        return regStatus == noErr
    }

    public func unregister() {
        if let ref = hotKeyRef {
            UnregisterEventHotKey(ref)
            hotKeyRef = nil
        }
        if let handler = eventHandler {
            RemoveEventHandler(handler)
            eventHandler = nil
        }
    }
}
