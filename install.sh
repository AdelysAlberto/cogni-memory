#!/usr/bin/env bash

set -e

echo "🧠 ========================================================"
echo "   [Cogni] Instalador Universal de Memoria para Agentes"
echo "   Cognitive Omniscient Grid for Networked Intelligence"
echo "   \"Así como el Byte es la unidad de datos,"
echo "    Cogni es la unidad de conocimiento sintético de tu agente.\""
echo "========================================================"

HOME_DIR="$HOME"
TARGET_DIR="$HOME_DIR/.cogni"
BIN_INSTALL_DIR="$HOME_DIR/.local/bin"
SRC_CACHE_DIR="$HOME_DIR/.cogni-src"

# Detectar sistema operativo una sola vez para todos los módulos
OS_TYPE="$(uname -s)"

# Retorna el directorio de usuario de VS Code según el OS
# Linux:  ~/.config/Code/User/
# macOS:  ~/Library/Application Support/Code/User/
get_vscode_user_dir() {
    case "$OS_TYPE" in
        Darwin) echo "$HOME_DIR/Library/Application Support/Code/User" ;;
        *)      echo "$HOME_DIR/.config/Code/User" ;;
    esac
}

mkdir -p "$TARGET_DIR"
mkdir -p "$BIN_INSTALL_DIR"

# Determinar si estamos dentro del repositorio clonado o ejecutando vía curl
if [ -f "go.mod" ] && grep -q "github.com/AdelysAlberto/cogni" go.mod 2>/dev/null; then
    REPO_DIR="$(pwd)"
else
    echo "📥 Descargando código fuente de Cogni..."
    if [ -d "$SRC_CACHE_DIR/.git" ]; then
        git -C "$SRC_CACHE_DIR" fetch --tags --quiet
        git -C "$SRC_CACHE_DIR" reset --hard --quiet origin/main
    else
        rm -rf "$SRC_CACHE_DIR"
        git clone --quiet https://github.com/AdelysAlberto/cogni-memory.git "$SRC_CACHE_DIR"
    fi
    REPO_DIR="$SRC_CACHE_DIR"
fi

# Detectar OS y arquitectura para descarga de binario precompilado
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

# Intentar instalar Go automáticamente según la distro
try_install_go() {
    echo "🔍 Go no encontrado. Intentando instalarlo automáticamente..."

    # Usar if/then en lugar de && para que set -e no interrumpa el script si falla
    if command -v snap &>/dev/null; then
        echo "📦 Instalando Go vía snap (se puede requerir contraseña)..."
        if sudo snap install go --classic 2>/dev/null; then return 0; fi
    fi

    if command -v apt-get &>/dev/null; then
        echo "📦 Instalando Go vía apt (se puede requerir contraseña)..."
        if sudo apt-get update -qq && sudo apt-get install -y golang-go 2>/dev/null; then return 0; fi
    fi

    if command -v dnf &>/dev/null; then
        echo "📦 Instalando Go vía dnf (se puede requerir contraseña)..."
        if sudo dnf install -y golang 2>/dev/null; then return 0; fi
    fi

    if command -v pacman &>/dev/null; then
        echo "📦 Instalando Go vía pacman (se puede requerir contraseña)..."
        if sudo pacman -S --noconfirm go 2>/dev/null; then return 0; fi
    fi

    if command -v brew &>/dev/null; then
        echo "📦 Instalando Go vía Homebrew..."
        if brew install go 2>/dev/null; then return 0; fi
    fi

    return 1
}

# 1. Intentar descargar binario precompilado desde GitHub Releases
PLATFORM="$(detect_platform)"
LATEST_RELEASE_URL="https://github.com/AdelysAlberto/cogni-memory/releases/latest/download/cogni_${PLATFORM}"

echo "🔍 Buscando binario precompilado para $PLATFORM..."
if curl -fsSL --head "$LATEST_RELEASE_URL" &>/dev/null; then
    echo "⬇️  Descargando binario precompilado para $PLATFORM..."
    curl -fsSL "$LATEST_RELEASE_URL" -o "$BIN_INSTALL_DIR/cogni"
    chmod +x "$BIN_INSTALL_DIR/cogni"
    echo "✅ Binario instalado en $BIN_INSTALL_DIR/cogni"

# 2. Compilar si Go está disponible
elif command -v go &>/dev/null; then
    VERSION_TAG="$(git -C "$REPO_DIR" describe --tags --abbrev=0 2>/dev/null || echo "v2.0.3")"
    echo "🔨 Compilando binario de Cogni en Go ($VERSION_TAG)..."
    (cd "$REPO_DIR" && go build -ldflags="-s -w -X github.com/AdelysAlberto/cogni/internal/cli.Version=${VERSION_TAG}" -o "$BIN_INSTALL_DIR/cogni" ./cmd/cogni)
    chmod +x "$BIN_INSTALL_DIR/cogni"
    echo "✅ Binario instalado en $BIN_INSTALL_DIR/cogni"

# 3. Copiar binario incluido en el repo si existe
elif [ -f "$REPO_DIR/bin/cogni" ]; then
    echo "📦 Copiando binario precompilado del repositorio..."
    cp "$REPO_DIR/bin/cogni" "$BIN_INSTALL_DIR/cogni"
    chmod +x "$BIN_INSTALL_DIR/cogni"

# 4. Intentar instalar Go automáticamente y compilar
elif try_install_go; then
    # Refrescar PATH por si snap/apt lo añadió
    export PATH="$PATH:/snap/bin:/usr/local/go/bin"
    if command -v go &>/dev/null; then
        VERSION_TAG="$(git -C "$REPO_DIR" describe --tags --abbrev=0 2>/dev/null || echo "v2.0.3")"
        echo "🔨 Compilando binario de Cogni en Go ($VERSION_TAG)..."
        (cd "$REPO_DIR" && go build -ldflags="-s -w -X github.com/AdelysAlberto/cogni/internal/cli.Version=${VERSION_TAG}" -o "$BIN_INSTALL_DIR/cogni" ./cmd/cogni)
        chmod +x "$BIN_INSTALL_DIR/cogni"
        echo "✅ Binario instalado en $BIN_INSTALL_DIR/cogni"
    else
        echo "⚠️  Go fue instalado pero requiere reiniciar la terminal."
        echo "   Ejecuta de nuevo el instalador tras abrir una nueva terminal."
        exit 1
    fi

# 5. Nada funcionó — guiar al usuario
else
    echo ""
    echo "❌ No se pudo instalar Cogni automáticamente."
    echo ""
    echo "   Cogni requiere Go >= 1.22. Instálalo con uno de estos métodos:"
    echo ""
    echo "   Ubuntu/Debian:   sudo snap install go --classic"
    echo "                 o  sudo apt install golang-go"
    echo "   Fedora/RHEL:     sudo dnf install golang"
    echo "   Arch Linux:      sudo pacman -S go"
    echo "   macOS:           brew install go"
    echo "   Manual:          https://go.dev/dl/"
    echo ""
    echo "   Luego vuelve a ejecutar:"
    echo "   bash <(curl -fsSL https://raw.githubusercontent.com/AdelysAlberto/cogni-memory/main/install.sh)"
    echo ""
    exit 1
fi

SKILL_SOURCE="$REPO_DIR/SKILL.md"

echo ""
echo "🤖 Selecciona el entorno o Harness de IA que utilizas:"
echo "1) Gemini Antigravity    (~/.gemini/config/skills/cogni/)"
echo "2) Cursor IDE            (~/.cursor/skills/cogni/)"
echo "3) OpenCode              (~/.config/opencode/skills/ & ~/.agents/skills/)"
echo "4) Agentes Estándar      (~/.agents/skills/cogni/)"
echo "5) GitHub Copilot        (~/.agents/skills/cogni/)  [VS Code / Copilot Chat]"
echo "6) Hermes CLI            (~/.hermes/skills/cogni/)"
echo "7) Instalar en TODOS los entornos detectados (Recomendado)"
echo "8) Omitir instalación de Skill"
echo ""

HARNESS_CHOICE=""
if [ -t 0 ]; then
    read -p "Ingresa tu opción (1-8) [por defecto: 7]: " HARNESS_CHOICE || true
elif [ -r /dev/tty ]; then
    read -p "Ingresa tu opción (1-8) [por defecto: 7]: " HARNESS_CHOICE < /dev/tty 2>/dev/null || true
fi

if [ -z "$HARNESS_CHOICE" ]; then
    echo "ℹ️ Modo no interactivo detectado. Seleccionando opción 7 (TODOS los entornos por defecto)..."
    HARNESS_CHOICE=7
fi

# ─── Módulo: Gemini Antigravity ─────────────────────────────────────────────
# Skills:  ~/.gemini/config/skills/<name>/SKILL.md
install_antigravity() {
    local skills_dir="$HOME_DIR/.gemini/config/skills"
    echo "  -> [Gemini Antigravity] Skills: $skills_dir/cogni/"
    mkdir -p "$skills_dir/cogni" "$skills_dir/agent-memory"
    cp -f "$SKILL_SOURCE" "$skills_dir/cogni/SKILL.md"
    cp -f "$SKILL_SOURCE" "$skills_dir/agent-memory/SKILL.md"
}

# ─── Módulo: Cursor IDE ──────────────────────────────────────────────────────
# Skills:  ~/.cursor/skills/<name>/SKILL.md
install_cursor() {
    local skills_dir="$HOME_DIR/.cursor/skills"
    echo "  -> [Cursor IDE] Skills: $skills_dir/cogni/"
    mkdir -p "$skills_dir/cogni"
    cp -f "$SKILL_SOURCE" "$skills_dir/cogni/SKILL.md"
}

# ─── Módulo: OpenCode ────────────────────────────────────────────────────────
# Skills:  ~/.config/opencode/skills/<name>/SKILL.md
#          ~/.agents/skills/<name>/SKILL.md
install_opencode() {
    local skills_dir1="$HOME_DIR/.config/opencode/skills"
    local skills_dir2="$HOME_DIR/.agents/skills"
    echo "  -> [OpenCode] Skills: $skills_dir1/cogni/ y $skills_dir2/cogni/"
    mkdir -p "$skills_dir1/cogni" "$skills_dir2/cogni"
    cp -f "$SKILL_SOURCE" "$skills_dir1/cogni/SKILL.md"
    cp -f "$SKILL_SOURCE" "$skills_dir2/cogni/SKILL.md"
}

# ─── Módulo: Agentes Estándar / Agentic CLI ──────────────────────────────────
# Skills:  ~/.agents/skills/<name>/SKILL.md
install_agents_std() {
    local skills_dir="$HOME_DIR/.agents/skills"
    echo "  -> [Agentes Estándar] Skills: $skills_dir/cogni/"
    mkdir -p "$skills_dir/cogni"
    cp -f "$SKILL_SOURCE" "$skills_dir/cogni/SKILL.md"
}

# ─── Módulo: GitHub Copilot (VS Code) ────────────────────────────────────────
# Skills:      ~/.agents/skills/<name>/SKILL.md          (runtime principal)
# Skills (compat): ~/.copilot/skills/<name>/SKILL.md     (legacy)
# Instrucciones: <vscode_user_dir>/prompts/cogni-copilot.instructions.md
install_copilot() {
    local skills_dir="$HOME_DIR/.agents/skills"
    local legacy_skills_dir="$HOME_DIR/.copilot/skills"
    local vscode_dir
    local prompts_dir
    local copilot_instruction_file
    vscode_dir="$(get_vscode_user_dir)"
    prompts_dir="$vscode_dir/prompts"
    copilot_instruction_file="$prompts_dir/cogni-copilot.instructions.md"

    echo "  -> [GitHub Copilot] OS: $OS_TYPE"
    echo "     Skills:    $skills_dir/cogni/SKILL.md"
    echo "     Skills(L): $legacy_skills_dir/cogni/SKILL.md"
    echo "     Rules:     $copilot_instruction_file"

    mkdir -p "$skills_dir/cogni" "$legacy_skills_dir/cogni" "$prompts_dir"
    cp -f "$SKILL_SOURCE" "$skills_dir/cogni/SKILL.md"
    cp -f "$SKILL_SOURCE" "$legacy_skills_dir/cogni/SKILL.md"

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

# ─── Módulo: Hermes CLI ──────────────────────────────────────────────────────
# Skills:  ~/.hermes/skills/<name>/SKILL.md
install_hermes() {
    local skills_dir="$HOME_DIR/.hermes/skills"
    echo "  -> [Hermes CLI] Skills: $skills_dir/cogni/"
    mkdir -p "$skills_dir/cogni"
    cp -f "$SKILL_SOURCE" "$skills_dir/cogni/SKILL.md"
}

case $HARNESS_CHOICE in
    1) install_antigravity ;;
    2) install_cursor ;;
    3) install_opencode ;;
    4) install_agents_std ;;
    5) install_copilot ;;
    6) install_hermes ;;
    7|*) 
        echo "🚀 Registrando en todos los arneses de IA..."
        install_antigravity
        install_cursor
        install_opencode
        install_agents_std
        install_copilot
        install_hermes
        ;;
    8) echo "⏭️ Instalación de skill omitida." ;;
esac

# Inicializar base de datos global y configurar MCP Servers
echo ""
echo "🗄️ Inicializando almacenamiento SQLite y configurando Servidores MCP..."
"$BIN_INSTALL_DIR/cogni" init --all

echo ""
echo "✅ [Cogni] ¡Instalación completada con éxito!"
echo "💡 Ahora puedes ejecutar desde cualquier terminal:"
echo "   - cogni search --query \"...\""
echo "   - cogni save --title \"...\" --summary \"...\""
echo "   - cogni ui          (Abre el Dashboard visual en navegador)"
echo "   - cogni upgrade     (Verifica y actualiza a la última versión)"
echo "   - cogni share       (Exporta memorias en Markdown)"
echo "   - cogni stats       (Métricas de tokens ahorrados)"
echo ""
