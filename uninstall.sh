#!/usr/bin/env bash

set -e

echo "🗑️ Desinstalando Cogni..."

HOME_DIR="$HOME"
BIN_PATH="$HOME_DIR/.local/bin/cogni"
VSCODE_PROMPTS_LINUX="$HOME_DIR/.config/Code/User/prompts/cogni-copilot.instructions.md"
VSCODE_PROMPTS_MACOS="$HOME_DIR/Library/Application Support/Code/User/prompts/cogni-copilot.instructions.md"

if [ -f "$BIN_PATH" ]; then
    rm -f "$BIN_PATH"
    echo "  -> Binario eliminado de $BIN_PATH"
fi

echo "  -> Limpiando skills en arneses de IA..."
rm -rf "$HOME_DIR/.gemini/config/skills/cogni" "$HOME_DIR/.gemini/config/skills/agent-memory"
rm -rf "$HOME_DIR/.cursor/skills/cogni" "$HOME_DIR/.cursor/skills/agent-memory"
rm -rf "$HOME_DIR/.claude/skills/cogni" "$HOME_DIR/.claude/skills/agent-memory"
rm -rf "$HOME_DIR/.config/opencode/skills/cogni" "$HOME_DIR/.agents/skills/cogni"
rm -rf "$HOME_DIR/.copilot/skills/cogni"
rm -rf "$HOME_DIR/.hermes/skills/cogni"
rm -f "$HOME_DIR/.gemini/config/rules/cogni.rules.md" "$HOME_DIR/.cursor/rules/cogni.rules.md" "$HOME_DIR/.claude/rules/cogni.rules.md" "$HOME_DIR/.config/opencode/rules/cogni.rules.md" "$HOME_DIR/.agents/rules/cogni.rules.md" "$HOME_DIR/.hermes/rules/cogni.rules.md"
rm -f "$VSCODE_PROMPTS_LINUX" "$VSCODE_PROMPTS_MACOS"

echo ""
read -p "¿Deseas eliminar también las bases de datos de memoria en ~/.cogni? (s/N): " REMOVE_DB || true

if [[ "$REMOVE_DB" =~ ^[sSyY]$ ]]; then
    rm -rf "$HOME_DIR/.cogni"
    echo "  -> Base de datos global ~/.cogni eliminada."
fi

echo "✅ Desinstalación de Cogni completada."
