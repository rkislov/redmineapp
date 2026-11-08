#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
case "$(go env GOOS)" in
 darwin)
  export MACOSX_DEPLOYMENT_TARGET=12.0
  export CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=12.0"
  export CGO_CXXFLAGS="${CGO_CXXFLAGS:-} -mmacosx-version-min=12.0"
  export CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=12.0"
  bundle="${REDMINE_APP_DIR:-dist/redmineapp.app}"
  go build -trimpath -ldflags='-s -w' -o "$bundle/Contents/MacOS/redmineapp" .
  mkdir -p "$bundle/Contents/Resources"
  cp packaging/Info.plist "$bundle/Contents/Info.plist"
  cp LICENSE NOTICE THIRD_PARTY_NOTICES.txt "$bundle/Contents/Resources/"
  mkdir -p dist/redmineapp.iconset
  for size in 16 32 128 256 512; do sips -z "$size" "$size" assets/icon.png --out "dist/redmineapp.iconset/icon_${size}x${size}.png" >/dev/null; done
  cp assets/icon.png dist/redmineapp.iconset/icon_512x512@2x.png
  iconutil -c icns dist/redmineapp.iconset -o "$bundle/Contents/Resources/redmineapp.icns"
  codesign --force --deep --sign - "$bundle"
  ditto -c -k --keepParent "$bundle" "dist/redmineapp-macos-$(go env GOARCH).zip"
  hdiutil create -volname redmineapp -srcfolder "$bundle" -ov -format UDZO "dist/redmineapp-macos-$(go env GOARCH).dmg"
  ;;
 linux)
  go build -trimpath -ldflags='-s -w' -o dist/redmineapp .
  stage="dist/linux-$(go env GOARCH)"
  mkdir -p "$stage/usr/bin" "$stage/usr/share/applications" "$stage/usr/share/icons/hicolor/256x256/apps" "$stage/usr/share/doc/redmineapp"
  cp LICENSE NOTICE THIRD_PARTY_NOTICES.txt "$stage/usr/share/doc/redmineapp/"
  cp dist/redmineapp "$stage/usr/bin/"
  cp packaging/redmineapp.desktop "$stage/usr/share/applications/"
  cp assets/icon.png "$stage/usr/share/icons/hicolor/256x256/apps/redmineapp.png"
  tar -C "$stage" -czf "dist/redmineapp-linux-$(go env GOARCH).tar.gz" usr
  if command -v dpkg-deb >/dev/null; then
   mkdir -p "$stage/DEBIAN"
   cat > "$stage/DEBIAN/control" <<CONTROL
Package: redmineapp
Version: 0.1.3
Architecture: $(go env GOARCH)
Maintainer: redmineapp contributors
Depends: libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libxxf86vm1, libwayland-client0, libwayland-cursor0, libwayland-egl1, libxkbcommon0, libegl1
Description: Native cross-platform Redmine desktop client written in Go
CONTROL
   dpkg-deb --build --root-owner-group "$stage" "dist/redmineapp-linux-$(go env GOARCH).deb"
  fi
  ;;
 windows)
  go build -trimpath -ldflags='-s -w -H=windowsgui' -o dist/redmineapp.exe .
  if command -v iscc >/dev/null; then iscc packaging/windows.iss; fi
  ;;
 *) echo 'Unsupported platform' >&2; exit 1;;
esac
