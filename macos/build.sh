#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

APP_NAME="CogniBar"
BUILD_DIR="$SCRIPT_DIR/build"
APP_BUNDLE="$BUILD_DIR/$APP_NAME.app"
CONTENTS_DIR="$APP_BUNDLE/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

echo "🔨 Compilando $APP_NAME con swiftc nativo..."

rm -rf "$BUILD_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

cp "Resources/Info.plist" "$CONTENTS_DIR/Info.plist"

SWIFT_SOURCES=(
    "Sources/Theme.swift"
    "Sources/Logo.swift"
    "Sources/Shell.swift"
    "Sources/DBWatcher.swift"
    "Sources/HotKey.swift"
    "Sources/Controller.swift"
    "Sources/ContentView.swift"
    "Sources/main.swift"
)

# Compilar para arm64 (Apple Silicon) y x86_64 (Intel)
ARCHS=("arm64" "x86_64")
TARGET_MACOS="13.0"

for arch in "${ARCHS[@]}"; do
    echo "  • Compilando arquitectura $arch..."
    swiftc -O -swift-version 5 -parse-as-library -target "$arch-apple-macos$TARGET_MACOS" \
        -framework AppKit -framework SwiftUI -framework ServiceManagement -framework Carbon \
        "${SWIFT_SOURCES[@]}" \
        -o "$BUILD_DIR/$APP_NAME-$arch"
done

echo "  • Creando binario universal (Fat Binary)..."
lipo -create "$BUILD_DIR/$APP_NAME-arm64" "$BUILD_DIR/$APP_NAME-x86_64" -output "$MACOS_DIR/$APP_NAME"
rm -f "$BUILD_DIR/$APP_NAME-arm64" "$BUILD_DIR/$APP_NAME-x86_64"

# Firma de código (Ad-hoc para local o Developer ID si está configurado)
if [ -n "$DEVELOPER_ID" ]; then
    echo "🔏 Firmando con Apple Developer ID: $DEVELOPER_ID..."
    codesign --force --options runtime --timestamp --sign "$DEVELOPER_ID" "$APP_BUNDLE"
else
    echo "🔏 Aplicando firma ad-hoc local..."
    codesign --force --sign - "$APP_BUNDLE"
fi

echo "✅ App bundle generado exitosamente en: $APP_BUNDLE"

if [[ "$1" == "--install" || "$1" == "-i" ]]; then
    INSTALL_TARGET="$HOME/Applications/$APP_NAME.app"
    mkdir -p "$HOME/Applications"
    echo "🚀 Instalando en $INSTALL_TARGET..."
    
    # Detener instancia previa si está corriendo
    pkill -x "$APP_NAME" 2>/dev/null || true
    rm -rf "$INSTALL_TARGET"
    cp -R "$APP_BUNDLE" "$INSTALL_TARGET"
    
    open "$INSTALL_TARGET"
    echo "✨ $APP_NAME iniciado en la barra de menús."
fi
