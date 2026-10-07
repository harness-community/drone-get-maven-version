#!/bin/sh
# Release gate: static checks on the Windows images and the pipelines that
# publish them. Run by .harness/validate.yaml and .harness/publish.yaml.

set -eu

fail=0
err() { echo "ERROR: $*" >&2; fail=1; }

for ltsc in ltsc2019 ltsc2022 ltsc2025; do
  df="docker/Dockerfile.windows.amd64.$ltsc"
  [ -f "$df" ] || { err "$df is missing"; continue; }

  grep -Eq '^ARG BASE_IMAGE=harness/ci-base@sha256:[0-9a-f]{64}$' "$df" \
    || err "$df: BASE_IMAGE must be harness/ci-base pinned by sha256 digest"

  tag=$(sed -n 's/^ARG IMAGE_VERSION=\(.*\)$/\1/p' "$df")
  case "$tag" in
    "windows-$ltsc"-r[1-9]*) ;;
    *) err "$df: IMAGE_VERSION '$tag' must look like windows-$ltsc-rN"; continue ;;
  esac

  grep -q "$tag" .harness/publish.yaml 2>/dev/null \
    || err ".harness/publish.yaml must publish $tag (IMAGE_VERSION in $df)"

  grep -q '^ENTRYPOINT \["C:\\\\app\\\\drone-maven.exe"\]$' "$df" \
    || err "$df: ENTRYPOINT must be exec form C:\\app\\drone-maven.exe"
  grep -q 'io.harness.release.status="candidate"' "$df" \
    || err "$df: missing io.harness.release.status=\"candidate\" label"
  grep -Eq 'servercore|java-build|java-ci-images' "$df" \
    && err "$df: must derive from harness/ci-base only"
  grep -Eq -- 'choco(\.exe)?.* install .*--version|install \$pkg .*--version' "$df" \
    && err "$df: Maven and Java must stay unpinned"
done

[ -f docker/Dockerfile.windows.amd64 ] \
  && err "docker/Dockerfile.windows.amd64 must be removed; windows-amd64 is a legacy tag only"

if ls docker | grep -Eq '1809|-amd64\.ltsc'; then
  err "no 1809 or '-amd64' suffixed names allowed under docker/"
fi

grep -q godotenv go.mod && err "go.mod must not require godotenv"

if git ls-files 2>/dev/null | grep -Eq '(^release/|\.exe$)'; then
  err "binaries (release/, *.exe) must not be committed"
fi

[ "$fail" -eq 0 ] && echo "release checks passed"
exit "$fail"
