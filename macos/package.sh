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

DMG_STAGE="$BUILD_DIR/dmg_stage"
rm -rf "$DMG_STAGE"
mkdir -p "$DMG_STAGE"

cp -R "$APP_BUNDLE" "$DMG_STAGE/"
ln -s /Applications "$DMG_STAGE/Applications"

DMG_PATH="$DIST_DIR/$APP_NAME.dmg"
ZIP_PATH="$DIST_DIR/$APP_NAME-macOS.zip"

rm -f "$DMG_PATH" "$ZIP_PATH"

echo "💿 Creando imagen de disco DMG instalador..."
hdiutil create -volname "$APP_NAME" -srcfolder "$DMG_STAGE" -ov -format UDZO "$DMG_PATH"
ditto -c -k --keepParent "$APP_BUNDLE" "$ZIP_PATH"

if [ -z "$DEVELOPER_ID" ]; then
    echo "⚠️ Variable DEVELOPER_ID no definida."
    echo "Para empaquetar y notarizar oficialmente:"
    echo "  export DEVELOPER_ID=\"Developer ID Application: Tu Nombre (TEAM_ID)\""
    echo "  export NOTARY_PROFILE=\"perfil-notary\""
    echo ""
    echo "✅ DMG local generado en: $DMG_PATH"
    echo "✅ ZIP local generado en: $ZIP_PATH"
    exit 0
fi

echo "🔏 Firmando App Bundle con Hardened Runtime..."
codesign --force --deep --options runtime --timestamp --sign "$DEVELOPER_ID" "$APP_BUNDLE"

# Re-generar DMG con la app firmada
rm -rf "$DMG_STAGE"
mkdir -p "$DMG_STAGE"
cp -R "$APP_BUNDLE" "$DMG_STAGE/"
ln -s /Applications "$DMG_STAGE/Applications"

rm -f "$DMG_PATH"
hdiutil create -volname "$APP_NAME" -srcfolder "$DMG_STAGE" -ov -format UDZO "$DMG_PATH"

echo "🔏 Firmando archivo DMG..."
codesign --force --timestamp --sign "$DEVELOPER_ID" "$DMG_PATH"

if [ -n "$NOTARY_PROFILE" ]; then
    echo "☁️ Enviando DMG a notarizar a Apple Notary Service..."
    xcrun notarytool submit "$DMG_PATH" --keychain-profile "$NOTARY_PROFILE" --wait
    
    echo "📎 Grapando ticket de notarización al DMG (Stapling)..."
    xcrun stapler staple "$DMG_PATH"
    
    echo "🔍 Validando con Gatekeeper..."
    spctl -a -t open --context context:primary-signature -vv "$DMG_PATH" || true
    
    echo ""
    echo "✅ ¡DMG oficial firmado, notarizado y grapado con éxito!"
    echo "📍 Ubicación: $DMG_PATH"
fi
