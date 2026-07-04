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
  MOMENTO_PRE_RELEASE_COMMIT  Commit message for pending local changes. Default: chore: save pending changes before release
  MOMENTO_CODE_SIGN_IDENTITY  Release code signing identity. Default: Developer ID Application
  MOMENTO_NOTARY_PROFILE      notarytool keychain profile used for notarization
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
NOTARY_ZIP="$DIST_DIR/$PROJECT_NAME-$MARKETING_VERSION-notary.zip"
DOWNLOAD_URL="https://github.com/$GITHUB_REPOSITORY/releases/download/$RELEASE_TAG/$DMG_NAME"
RELEASE_URL="https://github.com/$GITHUB_REPOSITORY/releases/tag/$RELEASE_TAG"
CODE_SIGN_IDENTITY="${MOMENTO_CODE_SIGN_IDENTITY:-Developer ID Application}"
NOTARY_PROFILE="${MOMENTO_NOTARY_PROFILE:-}"

[[ "$MARKETING_VERSION" =~ ^[0-9]+(\.[0-9]+){1,2}$ ]] || fail "marketing version must look like 1.0.1"
[[ "$BUILD_NUMBER" =~ ^[0-9]+$ ]] || fail "build number must be a positive integer"

cd "$(dirname "$0")/.."

require_command git
require_command xcodebuild
require_command hdiutil
require_command plutil
require_command xmllint
require_command perl
require_command gh
require_command curl
require_command xcrun
require_command security
require_command codesign
require_command ditto
require_command spctl

[[ "$CODE_SIGN_IDENTITY" == Developer\ ID\ Application* ]] || fail "release signing identity must be Developer ID Application, got: $CODE_SIGN_IDENTITY"
security find-identity -p codesigning -v | grep -F "\"$CODE_SIGN_IDENTITY" >/dev/null || fail "missing code signing identity: $CODE_SIGN_IDENTITY"
[[ -n "$NOTARY_PROFILE" ]] || fail "MOMENTO_NOTARY_PROFILE is required for release notarization"

step "Checking working tree"
CURRENT_BRANCH="$(git branch --show-current)"
[[ -n "$CURRENT_BRANCH" ]] || fail "not on a branch"
git rev-parse --verify --quiet "$RELEASE_TAG" >/dev/null && fail "local tag already exists: $RELEASE_TAG"
git ls-remote --exit-code --tags origin "refs/tags/$RELEASE_TAG" >/dev/null 2>&1 && fail "remote tag already exists: $RELEASE_TAG"
gh auth status --active --hostname github.com >/dev/null
gh release view "$RELEASE_TAG" --repo "$GITHUB_REPOSITORY" >/dev/null 2>&1 && fail "GitHub Release already exists: $RELEASE_TAG"

if [[ -n "$(git status --porcelain=v1)" ]]; then
  step "Committing pending local changes"
  git add --all
  git --no-pager diff --cached --check
  git commit -m "${MOMENTO_PRE_RELEASE_COMMIT:-chore: save pending changes before release}"
fi

printf 'Preparing %s %s (%s)\n' "$PROJECT_NAME" "$MARKETING_VERSION" "$BUILD_NUMBER"

step "Updating Xcode version settings"
perl -0pi -e "s/CURRENT_PROJECT_VERSION = [^;]+;/CURRENT_PROJECT_VERSION = $BUILD_NUMBER;/g; s/MARKETING_VERSION = [^;]+;/MARKETING_VERSION = $MARKETING_VERSION;/g" "$PBXPROJ_FILE"

rm -rf "$DERIVED_DATA" "$DMG_ROOT"
rm -f "$DMG_PATH" "$NOTARY_ZIP"
mkdir -p "$DMG_ROOT"

step "Building Release app"
xcodebuild \
  -project "$PROJECT_FILE" \
  -scheme "$SCHEME_NAME" \
  -configuration Release \
  -destination 'platform=macOS' \
  -derivedDataPath "$DERIVED_DATA" \
  build \
  CODE_SIGN_STYLE=Manual \
  CODE_SIGN_IDENTITY="$CODE_SIGN_IDENTITY"

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

step "Verifying app signature"
codesign --verify --deep --strict --verbose=2 "$APP_PATH"
codesign -dvvv "$APP_PATH" 2>&1 | grep -F "Authority=Developer ID Application:" >/dev/null || fail "release app was not signed with Developer ID Application"

step "Notarizing app"
ditto -c -k --keepParent "$APP_PATH" "$NOTARY_ZIP"
xcrun notarytool submit "$NOTARY_ZIP" --keychain-profile "$NOTARY_PROFILE" --wait
xcrun stapler staple "$APP_PATH"
xcrun stapler validate "$APP_PATH"
rm -f "$NOTARY_ZIP"

step "Creating DMG"
cp -R "$APP_PATH" "$DMG_ROOT/"
ln -s /Applications "$DMG_ROOT/Applications"
hdiutil create -volname "$PROJECT_NAME" -srcfolder "$DMG_ROOT" -ov -format UDZO "$DMG_PATH"

step "Notarizing DMG"
xcrun notarytool submit "$DMG_PATH" --keychain-profile "$NOTARY_PROFILE" --wait
xcrun stapler staple "$DMG_PATH"
xcrun stapler validate "$DMG_PATH"

step "Verifying DMG and Gatekeeper acceptance"
hdiutil verify "$DMG_PATH"
spctl -a -vvv -t exec "$APP_PATH"
spctl -a -vvv -t open --context context:primary-signature "$DMG_PATH"

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

step "Committing release metadata"
git add "$PBXPROJ_FILE" "$APPCAST_FILE"
git --no-pager diff --cached --check
git commit -m "chore: release $MARKETING_VERSION"
git tag "$RELEASE_TAG"

step "Pushing branch and tag"
git push origin "$CURRENT_BRANCH"
git push origin "$RELEASE_TAG"

step "Creating GitHub Release"
gh release create "$RELEASE_TAG" "$DMG_PATH" \
  --repo "$GITHUB_REPOSITORY" \
  --title "$PROJECT_NAME $MARKETING_VERSION" \
  --generate-notes \
  --latest

step "Checking GitHub Pages feed"
if curl -fsSL https://seaony.github.io/Momento/appcast.xml >/dev/null; then
  printf 'GitHub Pages feed is reachable.\n'
else
  printf 'GitHub Pages feed is not updated yet; it may need a short refresh window.\n'
fi

cat <<EOF

Published release:
  Version:  $MARKETING_VERSION
  Build:    $BUILD_NUMBER
  DMG:      $DMG_PATH
  Appcast:  $APPCAST_FILE
  Tag:      $RELEASE_TAG
  Release:  https://github.com/$GITHUB_REPOSITORY/releases/tag/$RELEASE_TAG
EOF
