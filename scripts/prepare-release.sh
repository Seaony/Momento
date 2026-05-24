#!/usr/bin/env bash
set -euo pipefail

PROJECT_NAME="Momento"
SCHEME_NAME="Momento"
PROJECT_FILE="Momento.xcodeproj"
PBXPROJ_FILE="$PROJECT_FILE/project.pbxproj"
DIST_DIR="dist"
DERIVED_DATA="$DIST_DIR/ReleaseBuild"
DMG_ROOT="$DIST_DIR/dmg-root"
APPCAST_FILE="appcast.xml"
GITHUB_REPOSITORY="${MOMENTO_RELEASE_REPOSITORY:-Seaony/Momento}"

export GIT_PAGER=cat
export PAGER=cat

usage() {
  cat <<EOF
Usage:
  scripts/prepare-release.sh <marketing-version> <build-number>

Example:
  scripts/prepare-release.sh 1.0.1 2

Environment:
  MOMENTO_RELEASE_REPOSITORY  GitHub repository used in appcast URLs. Default: Seaony/Momento
  MOMENTO_RELEASE_TAG         Release tag used in appcast URLs. Default: <marketing-version>
EOF
}

fail() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

step() {
  printf '\n==> %s\n' "$1"
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 2 ]]; then
  usage
  exit 64
fi

MARKETING_VERSION="$1"
BUILD_NUMBER="$2"
RELEASE_TAG="${MOMENTO_RELEASE_TAG:-$MARKETING_VERSION}"
DMG_NAME="$PROJECT_NAME-$MARKETING_VERSION.dmg"
DMG_PATH="$DIST_DIR/$DMG_NAME"
DOWNLOAD_URL="https://github.com/$GITHUB_REPOSITORY/releases/download/$RELEASE_TAG/$DMG_NAME"
RELEASE_URL="https://github.com/$GITHUB_REPOSITORY/releases/tag/$RELEASE_TAG"

[[ "$MARKETING_VERSION" =~ ^[0-9]+(\.[0-9]+){1,2}$ ]] || fail "marketing version must look like 1.0.1"
[[ "$BUILD_NUMBER" =~ ^[0-9]+$ ]] || fail "build number must be a positive integer"

cd "$(dirname "$0")/.."

require_command git
require_command xcodebuild
require_command hdiutil
require_command plutil
require_command xmllint
require_command perl

step "Checking working tree"
git --no-pager diff --quiet || fail "working tree has unstaged changes; commit or stash them first"
git --no-pager diff --cached --quiet || fail "working tree has staged changes; commit or unstage them first"

printf 'Preparing %s %s (%s)\n' "$PROJECT_NAME" "$MARKETING_VERSION" "$BUILD_NUMBER"

step "Updating Xcode version settings"
perl -0pi -e "s/CURRENT_PROJECT_VERSION = [^;]+;/CURRENT_PROJECT_VERSION = $BUILD_NUMBER;/g; s/MARKETING_VERSION = [^;]+;/MARKETING_VERSION = $MARKETING_VERSION;/g" "$PBXPROJ_FILE"

rm -rf "$DERIVED_DATA" "$DMG_ROOT"
rm -f "$DMG_PATH"
mkdir -p "$DMG_ROOT"

step "Building Release app"
xcodebuild \
  -project "$PROJECT_FILE" \
  -scheme "$SCHEME_NAME" \
  -configuration Release \
  -destination 'platform=macOS' \
  -derivedDataPath "$DERIVED_DATA" \
  build

APP_PATH="$DERIVED_DATA/Build/Products/Release/$PROJECT_NAME.app"
INFO_PLIST="$APP_PATH/Contents/Info.plist"
SPARKLE_BIN="$DERIVED_DATA/SourcePackages/artifacts/sparkle/Sparkle/bin"

[[ -d "$APP_PATH" ]] || fail "built app was not found at $APP_PATH"
[[ -x "$SPARKLE_BIN/sign_update" ]] || fail "Sparkle sign_update was not found at $SPARKLE_BIN/sign_update"

APP_MARKETING_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$INFO_PLIST")"
APP_BUILD_NUMBER="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "$INFO_PLIST")"
MINIMUM_SYSTEM_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "$INFO_PLIST")"

[[ "$APP_MARKETING_VERSION" == "$MARKETING_VERSION" ]] || fail "built app version is $APP_MARKETING_VERSION, expected $MARKETING_VERSION"
[[ "$APP_BUILD_NUMBER" == "$BUILD_NUMBER" ]] || fail "built app build is $APP_BUILD_NUMBER, expected $BUILD_NUMBER"

step "Creating DMG"
cp -R "$APP_PATH" "$DMG_ROOT/"
ln -s /Applications "$DMG_ROOT/Applications"
hdiutil create -volname "$PROJECT_NAME" -srcfolder "$DMG_ROOT" -ov -format UDZO "$DMG_PATH"

step "Verifying DMG and code signature"
hdiutil verify "$DMG_PATH"
codesign --verify --deep --strict --verbose=2 "$APP_PATH"

step "Signing update archive for Sparkle"
SIGN_OUTPUT="$("$SPARKLE_BIN/sign_update" "$DMG_PATH")"
ED_SIGNATURE="$(printf '%s\n' "$SIGN_OUTPUT" | sed -n 's/.*sparkle:edSignature="\([^"]*\)".*/\1/p')"
CONTENT_LENGTH="$(printf '%s\n' "$SIGN_OUTPUT" | sed -n 's/.*length="\([^"]*\)".*/\1/p')"

[[ -n "$ED_SIGNATURE" ]] || fail "failed to parse Sparkle edSignature"
[[ -n "$CONTENT_LENGTH" ]] || fail "failed to parse DMG content length"

"$SPARKLE_BIN/sign_update" --verify "$DMG_PATH" "$ED_SIGNATURE"

PUB_DATE="$(LC_ALL=C date -u '+%a, %d %b %Y %H:%M:%S +0000')"

step "Writing appcast"
cat > "$APPCAST_FILE" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0" xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle">
  <channel>
    <title>Momento Updates</title>
    <link>https://github.com/$GITHUB_REPOSITORY</link>
    <description>Release updates for Momento.</description>
    <language>zh-CN</language>
    <item>
      <title>Momento $MARKETING_VERSION</title>
      <link>$RELEASE_URL</link>
      <sparkle:version>$BUILD_NUMBER</sparkle:version>
      <sparkle:shortVersionString>$MARKETING_VERSION</sparkle:shortVersionString>
      <sparkle:minimumSystemVersion>$MINIMUM_SYSTEM_VERSION</sparkle:minimumSystemVersion>
      <pubDate>$PUB_DATE</pubDate>
      <enclosure
        url="$DOWNLOAD_URL"
        sparkle:edSignature="$ED_SIGNATURE"
        length="$CONTENT_LENGTH"
        type="application/octet-stream" />
    </item>
  </channel>
</rss>
EOF

step "Validating release files"
xmllint --noout "$APPCAST_FILE"
git --no-pager diff --check

rm -rf "$DMG_ROOT"

cat <<EOF

Prepared release artifacts:
  Version:  $MARKETING_VERSION
  Build:    $BUILD_NUMBER
  DMG:      $DMG_PATH
  Appcast:  $APPCAST_FILE
  Tag:      $RELEASE_TAG

Next steps:
  git add $PBXPROJ_FILE $APPCAST_FILE
  git commit -m "chore: release $MARKETING_VERSION"
  git tag $RELEASE_TAG
  git push origin master
  git push origin $RELEASE_TAG

Then create a GitHub Release for tag "$RELEASE_TAG" and upload:
  $DMG_PATH

After GitHub Pages refreshes, verify:
  curl -fsSL https://seaony.github.io/Momento/appcast.xml
EOF
