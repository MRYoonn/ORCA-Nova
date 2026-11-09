#!/usr/bin/env bash
set -euo pipefail

TAG="${1:-v1.6.0-rc.2}"
EXPECTED_COMMIT="${2:-9a358c0bd4b18fb755e657decbcee188ed59764e}"
EXPECTED_STABLE="${3:-v1.5.1}"
REPO="MRYoonn/ORCA-Nova"
API="https://api.github.com/repos/$REPO"
ROOT="https://github.com/$REPO/releases/download/$TAG"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl --fail --silent --show-error --location --retry 4 --retry-all-errors \
  --connect-timeout 25 --max-time 180 "$API/releases/tags/$TAG" -o "$TMP/release.json"
curl --fail --silent --show-error --location --retry 4 --retry-all-errors \
  --connect-timeout 25 --max-time 180 "$API/releases/latest" -o "$TMP/latest.json"

jq -e --arg tag "$TAG" --arg commit "$EXPECTED_COMMIT" \
  '.tag_name == $tag and .prerelease == true and .draft == false and .target_commitish == $commit' \
  "$TMP/release.json" >/dev/null
jq -e --arg stable "$EXPECTED_STABLE" \
  '.tag_name == $stable and .prerelease == false' "$TMP/latest.json" >/dev/null

jq -e '[.assets[].name] | sort == ["ORCA_Nova_Portable_Windows.zip","SHA256SUMS.txt"]' \
  "$TMP/release.json" >/dev/null
jq -e '.assets[] | select(.name == "ORCA_Nova_Portable_Windows.zip") | .size > 2000000' \
  "$TMP/release.json" >/dev/null

cd "$TMP"
for file in ORCA_Nova_Portable_Windows.zip SHA256SUMS.txt; do
  curl --fail --silent --show-error --location --retry 4 --retry-all-errors \
    --connect-timeout 25 --max-time 240 "$ROOT/$file" -o "$file"
done

sha256sum -c SHA256SUMS.txt
unzip -tqq ORCA_Nova_Portable_Windows.zip
unzip -Z1 ORCA_Nova_Portable_Windows.zip > entries.txt
grep -Fxq 'ORCA_Nova.exe' entries.txt
grep -Fxq 'ORCA_Nova_UI.exe' entries.txt
if grep -Eiq '(^|[/\\])orca(\\.exe)?$' entries.txt; then
  echo "Unexpected redistributed ORCA executable" >&2
  exit 1
fi
if grep -Eq '(^[/\\]|(^|[/\\])\.\.([/\\]|$)|^[A-Za-z]:)' entries.txt; then
  echo "Unsafe ZIP entry path detected" >&2
  exit 1
fi

printf 'RESULT: PASS\nTAG: %s\nSOURCE: %s\nLATEST_STABLE: %s\n' "$TAG" "$EXPECTED_COMMIT" "$EXPECTED_STABLE"
printf 'ZIP_SHA256: '; sha256sum ORCA_Nova_Portable_Windows.zip | cut -d' ' -f1
printf 'ZIP_SIZE: '; wc -c < ORCA_Nova_Portable_Windows.zip
printf 'ZIP_ENTRIES: '; wc -l < entries.txt
