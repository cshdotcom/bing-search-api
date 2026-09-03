#!/usr/bin/env bash
# 交叉编译并打包全平台发行版到 dist/
# 用法: bash build_release.sh [版本号]   (默认取 git tag,无 tag 则为 dev)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

BINARY="bing-search-api"
VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo dev)}"
DIST="dist"
LDFLAGS="-s -w -X main.version=${VERSION}"

PLATFORMS=(
  "linux amd64 tar.gz"
  "linux arm64 tar.gz"
  "linux 386 tar.gz"
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "windows amd64 zip"
  "windows 386 zip"
  "windows arm64 zip"
)

rm -rf "$DIST"
mkdir -p "$DIST"

for spec in "${PLATFORMS[@]}"; do
  set -- $spec
  GOOS="$1"; GOARCH="$2"; EXT="$3"
  NAME="${BINARY}_${VERSION}_${GOOS}_${GOARCH}"
  STAGE="$(mktemp -d)/${NAME}"
  mkdir -p "$STAGE"

  BIN="$BINARY"
  [ "$GOOS" = "windows" ] && BIN="${BINARY}.exe"

  echo "==> ${NAME}"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "$LDFLAGS" -o "${STAGE}/${BIN}" .
  cp README.md LICENSE "$STAGE/"

  if [ "$EXT" = "zip" ]; then
    (cd "$(dirname "$STAGE")" && zip -r -q "${ROOT}/${DIST}/${NAME}.zip" "${NAME}") \
      || (cd "$(dirname "$STAGE")" && python3 -m zipfile -c "${ROOT}/${DIST}/${NAME}.zip" "${NAME}")
  else
    (cd "$(dirname "$STAGE")" && tar -czf "${ROOT}/${DIST}/${NAME}.tar.gz" "${NAME}")
  fi
  rm -rf "$(dirname "$STAGE")"
done

(cd "$DIST" && sha256sum *.tar.gz *.zip > SHA256SUMS)

echo
echo "发行包已生成:"
ls -lh "$DIST"
