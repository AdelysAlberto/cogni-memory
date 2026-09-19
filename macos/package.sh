#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

APP_NAME="CogniBar"
BUILD_DIR="$SCRIPT_DIR/build"
APP_BUNDLE="$BUILD_DIR/$APP_NAME.app"
DIST_DIR="$SCRIPT_DIR/dist"

# Asegurar compilación fresca
./build.sh

mkdir -p "$DIST_DIR"

if [ -z "$DEVELOPER_ID" ]; then
    echo "⚠️ Variable DEVELOPER_ID no definida."
    echo "Para empaquetar y notarizar oficialmente:"
    echo "  export DEVELOPER_ID=\"Developer ID Application: Tu Nombre (TEAM_ID)\""
    echo "  export NOTARY_PROFILE=\"perfil-notary\""
    echo ""
    echo "📦 Creando ZIP con firma local..."
    ZIP_PATH="$DIST_DIR/$APP_NAME-macOS.zip"
    ditto -c -k --keepParent "$APP_BUNDLE" "$ZIP_PATH"
    echo "✅ Archivo generado en: $ZIP_PATH"
    exit 0
fi

echo "🔏 Firmando con Hardened Runtime para Notarización de Apple..."
codesign --force --deep --options runtime --timestamp --sign "$DEVELOPER_ID" "$APP_BUNDLE"

ZIP_PATH="$DIST_DIR/$APP_NAME-macOS.zip"
ditto -c -k --keepParent "$APP_BUNDLE" "$ZIP_PATH"

if [ -n "$NOTARY_PROFILE" ]; then
    echo "☁️ Enviando a notarizar a Apple Notary Service..."
    xcrun notarytool submit "$ZIP_PATH" --keychain-profile "$NOTARY_PROFILE" --wait
    
    echo "📎 Grapando ticket de notarización (Stapling)..."
    xcrun stapler staple "$APP_BUNDLE"
    
    # Re-empaquetar con el ticket grapado
    ditto -c -k --keepParent "$APP_BUNDLE" "$ZIP_PATH"
    echo "✅ Notarización y grapado completados exitosamente en: $ZIP_PATH"
fi
