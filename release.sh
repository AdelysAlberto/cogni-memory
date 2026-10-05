#!/usr/bin/env bash

set -euo pipefail

TYPE="${1:-patch}"

if [[ "$TYPE" != "patch" && "$TYPE" != "minor" && "$TYPE" != "major" ]]; then
    echo "❌ Tipo de versión inválido: $TYPE"
    echo "Uso: ./release.sh [patch|minor|major]"
    exit 1
fi

REPOSITORY="AdelysAlberto/cogni-memory"

if ! command -v gh &>/dev/null; then
    echo "Error: GitHub CLI (gh) es obligatorio para publicar los binarios."
    echo "Instalen gh: https://cli.github.com/ y ejecuten: gh auth login"
    exit 1
fi

if ! gh auth status --hostname github.com &>/dev/null; then
    echo "Error: falta autenticacion de GitHub. Ejecuten: gh auth login"
    exit 1
fi

# Obtener última tag de git
LATEST_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo "v2.0.3")"
VERSION="${LATEST_TAG#v}"

IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

case "$TYPE" in
    patch)
        PATCH=$((PATCH + 1))
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
esac

NEW_TAG="v${MAJOR}.${MINOR}.${PATCH}"

echo "🏷️ Incrementando versión: ${LATEST_TAG} -> ${NEW_TAG}"

# Asegurar working tree limpio o hacer commit si hay cambios pendientes
if [[ -n $(git status --porcelain) ]]; then
    echo "📦 Guardando cambios detectados en git..."
    git add .
    git commit -m "chore: release ${NEW_TAG}"
fi

echo "🔨 Compilando binarios multiplataforma con ldflags ${NEW_TAG}..."
mkdir -p bin

build_binary() {
    local os="$1"
    local arch="$2"
    local output="bin/cogni_${os}_${arch}"
    echo "  • Compilando ${output}..."
    GOOS="$os" GOARCH="$arch" go build -ldflags="-s -w -X github.com/AdelysAlberto/cogni/internal/cli.Version=${NEW_TAG}" -o "$output" ./cmd/cogni
    if [[ "$os" == "darwin" ]] && command -v codesign &>/dev/null; then
        codesign -s - -f "$output" 2>/dev/null || true
    fi
}

build_binary "darwin" "arm64"
build_binary "darwin" "amd64"
build_binary "linux" "amd64"
build_binary "linux" "arm64"

# Crear alias local bin/cogni para la plataforma actual
PLATFORM_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
PLATFORM_ARCH="$(uname -m)"
case "$PLATFORM_ARCH" in
    x86_64) PLATFORM_ARCH="amd64" ;;
    aarch64|arm64) PLATFORM_ARCH="arm64" ;;
esac
cp "bin/cogni_${PLATFORM_OS}_${PLATFORM_ARCH}" bin/cogni

echo "📌 Creando git tag ${NEW_TAG}..."
git tag -a "${NEW_TAG}" -m "Release ${NEW_TAG}"

echo "🚀 Subiendo cambios y tag a GitHub..."
git push origin main
git push origin "${NEW_TAG}"

echo "📦 Subiendo Release borrador a GitHub y adjuntando binarios multiplataforma..."
ASSETS=(bin/cogni_darwin_arm64 bin/cogni_darwin_amd64 bin/cogni_linux_amd64 bin/cogni_linux_arm64)
if [[ -f "macos/dist/CogniBar.dmg" ]]; then
    ASSETS+=(macos/dist/CogniBar.dmg)
fi
gh release create "${NEW_TAG}" "${ASSETS[@]}" --repo "$REPOSITORY" --verify-tag --draft --title "${NEW_TAG}" --notes "Release ${NEW_TAG}"
gh release edit "${NEW_TAG}" --repo "$REPOSITORY" --draft=false --latest

echo "✅ ¡Release ${NEW_TAG} publicado con éxito!"
