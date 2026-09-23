# Platform Abstraction Layer (PAL) & Regression Prevention Invariants

## 1. Platform Abstraction Layer (PAL) & Ports and Adapters
- All OS-specific operations (browser launching, desktop notifications, system tray integration, window management) MUST reside in `internal/platform/`.
- High-level business logic (CLI handlers, storage engines, server routes, MCP tools) MUST NEVER import `os/exec` directly to invoke OS-specific binaries (`open`, `xdg-open`, `rundll32`).
- Always consume `platform.OpenURL(url)` and `platform.LaunchTray(args, fallback)`.

## 2. Strict Target File Segregation (Go Build Tags)
- OS-specific implementations must use explicit build tags and filename conventions:
  - `platform_darwin.go`: `//go:build darwin`
  - `platform_linux.go`: `//go:build linux`
  - `platform_windows.go`: `//go:build windows`
- Modifying Linux desktop integration MUST NOT alter or destabilize macOS or Windows binaries.

## 3. Regression Prevention & Zero-Polling Invariant
- **Headless Detection**: Linux desktop integrations must explicitly verify graphical session presence (`DISPLAY` / `WAYLAND_DISPLAY`) before invoking UI tools. If headless, print friendly terminal instructions and never fail silently.
- **Zero-Polling on Idle**: System trays and background widgets must remain in 0% CPU idle state. Data queries are strictly **On-Demand Pull** (triggered by user clicks or menu interactions) or event-driven via kernel `inotify`. Background polling loops (`time.Sleep` / `setInterval` querying database) are strictly prohibited.
- **Deterministic Verification**: Any platform change must pass cross-compilation tests for `darwin`, `linux`, and `windows` before being committed.
