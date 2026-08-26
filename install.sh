#!/usr/bin/env bash

set -e

echo "🧠 Cogni — Sistema Universal de Memoria para Agentes de IA"
echo "────────────────────────────────────────────────────────"

HOME_DIR="$HOME"
TARGET_DIR="$HOME_DIR/.cogni"
BIN_INSTALL_DIR="$HOME_DIR/.local/bin"
SRC_CACHE_DIR="$HOME_DIR/.cogni-src"

OS_TYPE="$(uname -s)"

get_vscode_user_dir() {
    case "$OS_TYPE" in
        Darwin) echo "$HOME_DIR/Library/Application Support/Code/User" ;;
        *)      echo "$HOME_DIR/.config/Code/User" ;;
    esac
}

mkdir -p "$TARGET_DIR"
mkdir -p "$BIN_INSTALL_DIR"

if [ -f "go.mod" ] && grep -q "github.com/AdelysAlberto/cogni" go.mod 2>/dev/null; then
    REPO_DIR="$(pwd)"
else
    if [ -d "$SRC_CACHE_DIR/.git" ]; then
        git -C "$SRC_CACHE_DIR" fetch --tags --quiet
        git -C "$SRC_CACHE_DIR" reset --hard --quiet origin/main
    else
        rm -rf "$SRC_CACHE_DIR"
        git clone --quiet https://github.com/AdelysAlberto/cogni-memory.git "$SRC_CACHE_DIR"
    fi
    REPO_DIR="$SRC_CACHE_DIR"
fi

detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    case "$arch" in
        x86_64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l)  arch="arm" ;;
        *)       arch="$arch" ;;
    esac
    echo "${os}_${arch}"
}

try_install_go() {
    if command -v snap &>/dev/null; then
        sudo snap install go --classic &>/dev/null && return 0
    fi
    if command -v apt-get &>/dev/null; then
        sudo apt-get update -qq && sudo apt-get install -y golang-go &>/dev/null && return 0
    fi
    if command -v dnf &>/dev/null; then
        sudo dnf install -y golang &>/dev/null && return 0
    fi
    if command -v pacman &>/dev/null; then
        sudo pacman -S --noconfirm go &>/dev/null && return 0
    fi
    if command -v brew &>/dev/null; then
        brew install go &>/dev/null && return 0
    fi
    return 1
}

PLATFORM="$(detect_platform)"
LATEST_RELEASE_URL="https://github.com/AdelysAlberto/cogni-memory/releases/latest/download/cogni_${PLATFORM}"

if curl -fsSL --head "$LATEST_RELEASE_URL" &>/dev/null; then
    curl -fsSL "$LATEST_RELEASE_URL" -o "$BIN_INSTALL_DIR/cogni"
    chmod +x "$BIN_INSTALL_DIR/cogni"
    echo "✔ Binario listo en ~/.local/bin/cogni"
elif command -v go &>/dev/null; then
    VERSION_TAG="$(git -C "$REPO_DIR" describe --tags --abbrev=0 2>/dev/null || echo "v2.0.12")"
    (cd "$REPO_DIR" && go build -ldflags="-s -w -X github.com/AdelysAlberto/cogni/internal/cli.Version=${VERSION_TAG}" -o "$BIN_INSTALL_DIR/cogni" ./cmd/cogni)
    chmod +x "$BIN_INSTALL_DIR/cogni"
    echo "✔ Binario compilado e instalado en ~/.local/bin/cogni ($VERSION_TAG)"
elif [ -f "$REPO_DIR/bin/cogni" ]; then
    cp "$REPO_DIR/bin/cogni" "$BIN_INSTALL_DIR/cogni"
    chmod +x "$BIN_INSTALL_DIR/cogni"
    echo "✔ Binario listo en ~/.local/bin/cogni"
elif try_install_go; then
    export PATH="$PATH:/snap/bin:/usr/local/go/bin"
    if command -v go &>/dev/null; then
        VERSION_TAG="$(git -C "$REPO_DIR" describe --tags --abbrev=0 2>/dev/null || echo "v2.0.12")"
        (cd "$REPO_DIR" && go build -ldflags="-s -w -X github.com/AdelysAlberto/cogni/internal/cli.Version=${VERSION_TAG}" -o "$BIN_INSTALL_DIR/cogni" ./cmd/cogni)
        chmod +x "$BIN_INSTALL_DIR/cogni"
        echo "✔ Binario compilado e instalado en ~/.local/bin/cogni ($VERSION_TAG)"
    else
        echo "⚠️ Go instalado pero requiere reiniciar la terminal."
        exit 1
    fi
else
    echo "❌ Revisa que Go >= 1.22 esté disponible e inténtalo de nuevo."
    exit 1
fi

SKILL_SOURCE="$REPO_DIR/SKILL.md"
RULE_SOURCE="$REPO_DIR/rules/cogni.rules.md"

echo ""
echo "🤖 Selecciona el entorno o Harness de IA que utilizas:"
echo "  1) Gemini Antigravity    (~/.gemini/)"
echo "  2) Cursor IDE            (~/.cursor/)"
echo "  3) Claude Code / Desktop (~/.claude/)"
echo "  4) OpenCode              (~/.config/opencode/)"
echo "  5) Agentes Estándar      (~/.agents/)"
echo "  6) GitHub Copilot        (VS Code / Copilot Chat)"
echo "  7) Hermes CLI            (~/.hermes/)"
echo "  8) TODOS los entornos    (Recomendado)"
echo "  9) Omitir skill"
echo ""

HARNESS_CHOICE="${HARNESS_CHOICE:-}"
if [ -z "$HARNESS_CHOICE" ]; then
    if [ -t 0 ]; then
        read -p "Ingresa tu opción (1-9) [por defecto: 8]: " HARNESS_CHOICE || true
    elif [ -r /dev/tty ]; then
        read -p "Ingresa tu opción (1-9) [por defecto: 8]: " HARNESS_CHOICE < /dev/tty 2>/dev/null || true
    fi
fi

if [ -z "$HARNESS_CHOICE" ]; then
    echo "ℹ️ Modo no interactivo detectado: Seleccionando opción 8 (TODOS por defecto)..."
    HARNESS_CHOICE=8
fi

install_copilot_instructions() {
    local vscode_dir prompts_dir copilot_instruction_file
    vscode_dir="$(get_vscode_user_dir)"
    prompts_dir="$vscode_dir/prompts"
    copilot_instruction_file="$prompts_dir/cogni-copilot.instructions.md"

    mkdir -p "$prompts_dir"
    cat > "$copilot_instruction_file" <<'EOF'
---
description: "Cogni enforcement for GitHub Copilot when CLI is available"
applyTo: "**"
---

# Cogni Enforcement (Copilot)

When `cogni` CLI is available in PATH:

1. Before any non-trivial bugfix, run at least one targeted lookup:
    - `cogni search --query "<error-or-module-keyword>"`
2. For high-signal outcomes (bugfix, architecture, decision, discovery, config, pattern, preference), persist memory before final response:
    - `cogni save ...` or `cogni update --id ...`
3. Do not replace Cogni persistence with internal memory-only systems (e.g., `memory.create`) as final storage.
4. End-user response must include one of these 1-line confirmations:
    - `🧠 Memoria Recuperada: [project] "title/topic" (Tags: #tag1, #tag2)`
    - `💾 Memoria Guardada: [project] "title" (Category: #category, Tags: #tag1, #tag2, #tag3)`
5. If save/update fails, disclose explicitly:
    - `⚠️ Memoria No Guardada: <reason> (Attempted: <command>)`

If `cogni` CLI is not available, explain it and provide the exact install/enable step.
EOF
}

HARNESS_FLAG=""
case $HARNESS_CHOICE in
    1) HARNESS_FLAG="antigravity" ;;
    2) HARNESS_FLAG="cursor" ;;
    3) HARNESS_FLAG="claude" ;;
    4) HARNESS_FLAG="opencode" ;;
    5) HARNESS_FLAG="local" ;;
    6) HARNESS_FLAG="copilot" ;;
    7) HARNESS_FLAG="hermes" ;;
    8) HARNESS_FLAG="all" ;;
    9) HARNESS_FLAG="none" ;;
    *) HARNESS_FLAG="all" ;;
esac

if [ "$HARNESS_FLAG" = "copilot" ] || [ "$HARNESS_FLAG" = "all" ]; then
    install_copilot_instructions
fi

"$BIN_INSTALL_DIR/cogni" init --harness "$HARNESS_FLAG"

