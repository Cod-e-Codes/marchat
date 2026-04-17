#!/usr/bin/env bash
# Download release zips from GitHub, compute SHA256, and write Homebrew / Scoop /
# winget / AUR packaging files under OUTPUT_DIR (default: packaging-out).
# Requires: bash 4+, curl, sha256sum.
set -euo pipefail

RELEASE_TAG="${RELEASE_TAG:?set RELEASE_TAG to the git tag (e.g. v1.0.0)}"
GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-Cod-e-Codes/marchat}"
OUTPUT_DIR="${OUTPUT_DIR:-packaging-out}"

if [[ "$RELEASE_TAG" != v* ]]; then
  echo "RELEASE_TAG must start with v (got: $RELEASE_TAG)" >&2
  exit 1
fi

VER="${RELEASE_TAG#v}"
PKGVER="${VER//-/.}"
BASE_URL="https://github.com/${GITHUB_REPOSITORY}/releases/download/${RELEASE_TAG}"
RELEASE_DATE_UTC="${RELEASE_DATE_UTC:-$(date -u +%Y-%m-%d)}"

mkdir -p "${OUTPUT_DIR}/aur" "${OUTPUT_DIR}/winget"

declare -A HASH
for plat in linux-amd64 linux-arm64 windows-amd64 darwin-amd64 darwin-arm64; do
  zip_name="marchat-${RELEASE_TAG}-${plat}.zip"
  tmp_zip="/tmp/${zip_name}"
  echo "Downloading ${zip_name}..."
  curl -fsSL "${BASE_URL}/${zip_name}" -o "${tmp_zip}"
  HASH[${plat}]=$(sha256sum "${tmp_zip}" | awk '{print $1}')
done

WINGET_WIN_HASH=$(echo "${HASH[windows-amd64]}" | tr '[:lower:]' '[:upper:]')

# --- Homebrew formula ---
cat > "${OUTPUT_DIR}/marchat.rb" << HBEOF
class Marchat < Formula
  desc "Terminal chat with WebSockets, optional E2E encryption, and plugins"
  homepage "https://github.com/${GITHUB_REPOSITORY}"
  version "${VER}"
  license "MIT"

  on_macos do
    on_arm do
      url "${BASE_URL}/marchat-${RELEASE_TAG}-darwin-arm64.zip"
      sha256 "${HASH[darwin-arm64]}"
    end
    on_intel do
      url "${BASE_URL}/marchat-${RELEASE_TAG}-darwin-amd64.zip"
      sha256 "${HASH[darwin-amd64]}"
    end
  end

  on_linux do
    on_arm do
      url "${BASE_URL}/marchat-${RELEASE_TAG}-linux-arm64.zip"
      sha256 "${HASH[linux-arm64]}"
    end
    on_intel do
      url "${BASE_URL}/marchat-${RELEASE_TAG}-linux-amd64.zip"
      sha256 "${HASH[linux-amd64]}"
    end
  end

  def install
    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "marchat-client-darwin-arm64" => "marchat-client"
        bin.install "marchat-server-darwin-arm64" => "marchat-server"
      else
        bin.install "marchat-client-darwin-amd64" => "marchat-client"
        bin.install "marchat-server-darwin-amd64" => "marchat-server"
      end
    elsif OS.linux?
      if Hardware::CPU.arm?
        bin.install "marchat-client-linux-arm64" => "marchat-client"
        bin.install "marchat-server-linux-arm64" => "marchat-server"
      else
        bin.install "marchat-client-linux-amd64" => "marchat-client"
        bin.install "marchat-server-linux-amd64" => "marchat-server"
      end
    end
  end

  test do
    ENV["MARCHAT_DOCTOR_NO_NETWORK"] = "1"
    system "#{bin}/marchat-client", "-doctor-json"
  end
end
HBEOF

# --- Scoop manifest ---
SC_JSON="${OUTPUT_DIR}/marchat.json"
cat > "${SC_JSON}" << SCEOF
{
  "version": "${VER}",
  "description": "Terminal chat with WebSockets, optional E2E encryption, and plugins",
  "homepage": "https://github.com/${GITHUB_REPOSITORY}",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "${BASE_URL}/marchat-${RELEASE_TAG}-windows-amd64.zip",
      "hash": "${HASH[windows-amd64]}"
    }
  },
  "bin": [
    ["marchat-client-windows-amd64.exe", "marchat-client"],
    ["marchat-server-windows-amd64.exe", "marchat-server"]
  ],
  "checkver": "github/${GITHUB_REPOSITORY}",
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/${GITHUB_REPOSITORY}/releases/download/v\$version/marchat-v\$version-windows-amd64.zip"
      }
    }
  }
}
SCEOF

# --- winget (installer + version; locale copied from repo template) ---
cat > "${OUTPUT_DIR}/winget/Cod-e-Codes.Marchat.installer.yaml" << WGEOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.installer.1.6.0.schema.json
PackageIdentifier: Cod-e-Codes.Marchat
PackageVersion: ${VER}
Platform:
  - Windows.Desktop
MinimumOSVersion: 10.0.17763.0
InstallerType: zip
NestedInstallerType: portable
Installers:
  - Architecture: x64
    InstallerUrl: ${BASE_URL}/marchat-${RELEASE_TAG}-windows-amd64.zip
    InstallerSha256: ${WINGET_WIN_HASH}
    NestedInstallerFiles:
      - RelativeFilePath: marchat-client-windows-amd64.exe
        PortableCommandAlias: marchat-client
      - RelativeFilePath: marchat-server-windows-amd64.exe
        PortableCommandAlias: marchat-server
    ReleaseDate: ${RELEASE_DATE_UTC}
ManifestType: installer
ManifestVersion: 1.6.0
WGEOF

cat > "${OUTPUT_DIR}/winget/Cod-e-Codes.Marchat.yaml" << WGVF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.version.1.6.0.schema.json
PackageIdentifier: Cod-e-Codes.Marchat
PackageVersion: ${VER}
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
WGVF

LOCALE_SRC=""
if [[ -d packaging/winget/manifests/c/Cod-e-Codes/Marchat ]]; then
  LOCALE_SRC=$(find packaging/winget/manifests/c/Cod-e-Codes/Marchat -name 'Cod-e-Codes.Marchat.locale.en-US.yaml' -print -quit || true)
fi
if [[ -n "${LOCALE_SRC}" && -f "${LOCALE_SRC}" ]]; then
  cp "${LOCALE_SRC}" "${OUTPUT_DIR}/winget/Cod-e-Codes.Marchat.locale.en-US.yaml"
  sed -i "s/^PackageVersion:.*/PackageVersion: ${VER}/" "${OUTPUT_DIR}/winget/Cod-e-Codes.Marchat.locale.en-US.yaml"
  sed -i "s|^ReleaseNotesUrl:.*|ReleaseNotesUrl: https://github.com/${GITHUB_REPOSITORY}/releases/tag/${RELEASE_TAG}|" \
    "${OUTPUT_DIR}/winget/Cod-e-Codes.Marchat.locale.en-US.yaml"
else
  cat > "${OUTPUT_DIR}/winget/Cod-e-Codes.Marchat.locale.en-US.yaml" << WGLOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.defaultLocale.1.6.0.schema.json
PackageIdentifier: Cod-e-Codes.Marchat
PackageVersion: ${VER}
PackageLocale: en-US
Publisher: Cod-e-Codes
PublisherUrl: https://github.com/Cod-e-Codes
PackageName: marchat
PackageUrl: https://github.com/${GITHUB_REPOSITORY}
License: MIT
LicenseUrl: https://github.com/${GITHUB_REPOSITORY}/blob/main/LICENSE
ShortDescription: Lightweight terminal chat with WebSockets and optional E2E encryption
Description: |
  marchat is a self-hosted terminal chat with real-time messaging over WebSockets, optional E2E encryption, and a plugin ecosystem. This package installs the official release build of marchat-client and marchat-server.
Moniker: marchat
Tags:
  - chat
  - websocket
  - terminal
  - tui
  - cli
ReleaseNotesUrl: https://github.com/${GITHUB_REPOSITORY}/releases/tag/${RELEASE_TAG}
ManifestType: defaultLocale
ManifestVersion: 1.6.0
WGLOF
fi

# --- AUR PKGBUILD ---
cat > "${OUTPUT_DIR}/aur/PKGBUILD" << PKGEOF
# Maintainer: Cody Marsengill <cod.e.codes.dev@gmail.com>
pkgname=marchat-bin
pkgver=${PKGVER}
pkgrel=1
_pkgtag=${RELEASE_TAG}
pkgdesc='Terminal chat with WebSockets (official release binaries)'
arch=('x86_64' 'aarch64')
url='https://github.com/${GITHUB_REPOSITORY}'
license=('MIT')
options=('!strip')
depends=('glibc')
source_x86_64=("marchat-\${_pkgtag}-linux-amd64.zip::https://github.com/${GITHUB_REPOSITORY}/releases/download/\${_pkgtag}/marchat-\${_pkgtag}-linux-amd64.zip")
source_aarch64=("marchat-\${_pkgtag}-linux-arm64.zip::https://github.com/${GITHUB_REPOSITORY}/releases/download/\${_pkgtag}/marchat-\${_pkgtag}-linux-arm64.zip")
sha256sums_x86_64=('${HASH[linux-amd64]}')
sha256sums_aarch64=('${HASH[linux-arm64]}')

package() {
  if [[ \$CARCH == x86_64 ]]; then
    install -Dm755 "\$srcdir/marchat-client-linux-amd64" "\$pkgdir/usr/bin/marchat-client"
    install -Dm755 "\$srcdir/marchat-server-linux-amd64" "\$pkgdir/usr/bin/marchat-server"
  else
    install -Dm755 "\$srcdir/marchat-client-linux-arm64" "\$pkgdir/usr/bin/marchat-client"
    install -Dm755 "\$srcdir/marchat-server-linux-arm64" "\$pkgdir/usr/bin/marchat-server"
  fi
}
PKGEOF

TAB=$'\t'
SRC_X86_NAME="marchat-${RELEASE_TAG}-linux-amd64.zip"
SRC_ARM_NAME="marchat-${RELEASE_TAG}-linux-arm64.zip"
SRC_X86_URL="https://github.com/${GITHUB_REPOSITORY}/releases/download/${RELEASE_TAG}/${SRC_X86_NAME}"
SRC_ARM_URL="https://github.com/${GITHUB_REPOSITORY}/releases/download/${RELEASE_TAG}/${SRC_ARM_NAME}"
{
  echo "pkgbase = marchat-bin"
  echo "${TAB}pkgdesc = Terminal chat with WebSockets (official release binaries)"
  echo "${TAB}pkgver = ${PKGVER}"
  echo "${TAB}pkgrel = 1"
  echo "${TAB}url = https://github.com/${GITHUB_REPOSITORY}"
  echo "${TAB}arch = x86_64"
  echo "${TAB}arch = aarch64"
  echo "${TAB}license = MIT"
  echo "${TAB}depends = glibc"
  echo "${TAB}options = !strip"
  echo "${TAB}source_x86_64 = ${SRC_X86_NAME}::${SRC_X86_URL}"
  echo "${TAB}sha256sums_x86_64 = ${HASH[linux-amd64]}"
  echo "${TAB}source_aarch64 = ${SRC_ARM_NAME}::${SRC_ARM_URL}"
  echo "${TAB}sha256sums_aarch64 = ${HASH[linux-arm64]}"
  echo ""
  echo "pkgname = marchat-bin"
} > "${OUTPUT_DIR}/aur/.SRCINFO"

echo "Wrote manifests under ${OUTPUT_DIR}/"
ls -la "${OUTPUT_DIR}"
ls -la "${OUTPUT_DIR}/aur"
ls -la "${OUTPUT_DIR}/winget"
