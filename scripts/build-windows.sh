#!/usr/bin/env bash
# Кросс-сборка Sweepy для Windows из-под Linux.
#
# Вариант 1 (рекомендуется, нужен Docker):
#   go install github.com/fyne-io/fyne-cross@latest
#   fyne-cross windows -ldflags="-s -w" -app-id com.sweepy.app -icon Icon.png .
#
# Вариант 2 (без Docker, нужен mingw-w64):
#   sudo apt install gcc-mingw-w64-x86-64
#   bash scripts/build-windows.sh
set -euo pipefail

VERSION="${VERSION:-1.0.0}"
CC="${CC:-x86_64-w64-mingw32-gcc}"

mkdir -p dist
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="$CC" \
  go build -ldflags "-s -w -H=windowsgui -X sweepy/internal/build.Version=${VERSION}" \
  -o "dist/sweepy-windows-amd64.exe" .

echo "Готово: dist/sweepy-windows-amd64.exe"
