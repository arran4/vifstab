#!/usr/bin/env bash
# Tests GoReleaser's locally loaded snapshot images, never registry images.
set -euo pipefail
cd "$(dirname "$0")/.."
tmp=$(mktemp -d)
cid=''
cleanup() {
  if [[ -n "$cid" ]]; then docker rm -f "$cid" >/dev/null || true; fi
  rm -rf -- "$tmp"
}
trap cleanup EXIT
mkdir -m 755 "$tmp/fixtures"
: > "$tmp/fixtures/valid.fstab"
printf 'UUID=fixture /fixture ext4 invalid-option 0 2\n' > "$tmp/fixtures/invalid.fstab"
chmod 644 "$tmp/fixtures/"*.fstab

for arch in amd64 arm64; do
  image="ghcr.io/arran4/vifstab:snapshot-$arch"
  test "$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$image")" = "linux/$arch"
  test "$(docker image inspect --format '{{.Config.User}}' "$image")" = '65532:65532'
  test "$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.source"}}' "$image")" = 'https://github.com/arran4/vifstab'
  test "$(docker image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$image")" = "$(git rev-parse HEAD)"
  mapfile -t binaries < <(find dist -type f -name vifstab -path "*/vifstab_linux_$arch*/*")
  test "${#binaries[@]}" -eq 1
  cid=$(docker create --platform "linux/$arch" --pull never "$image" --help)
  docker cp "$cid:/usr/local/bin/vifstab" "$tmp/$arch-vifstab"
  cmp "${binaries[0]}" "$tmp/$arch-vifstab"
  for notice in LICENSE THIRD_PARTY_NOTICES.md; do
    docker cp "$cid:/usr/share/doc/vifstab/$notice" "$tmp/$notice"
    cmp "$notice" "$tmp/$notice"
  done
  docker rm "$cid" >/dev/null
  cid=''
done

# Execute the native amd64 image with no host configuration or devices.
image=ghcr.io/arran4/vifstab:snapshot-amd64
run=(docker run --pull never --rm --network none --cap-drop ALL
     --security-opt no-new-privileges --read-only)
mount=(--mount "type=bind,src=$tmp/fixtures,dst=/fixtures,readonly")
"${run[@]}" "$image" --help > "$tmp/help"
grep -q verify "$tmp/help"
"${run[@]}" "$image" version > "$tmp/version"
"$tmp/amd64-vifstab" version > "$tmp/native-version"
cmp "$tmp/native-version" "$tmp/version"
grep -F "$(git rev-parse HEAD)" "$tmp/version"

"${run[@]}" "${mount[@]}" "$image" verify --static --file /fixtures/valid.fstab > "$tmp/valid"
grep -F 'Validation successful.' "$tmp/valid"
status=0
"${run[@]}" "${mount[@]}" "$image" verify --static --file /fixtures/invalid.fstab > "$tmp/invalid" 2>&1 || status=$?
test "$status" -eq 1
grep -F 'Unknown ext4 mount option' "$tmp/invalid"

printf '# pipe\n' > "$tmp/expected"
"${run[@]}" -i "$image" format --file - < "$tmp/expected" > "$tmp/formatted"
cmp "$tmp/expected" "$tmp/formatted"
"${run[@]}" "${mount[@]}" "$image" verify --static --file /fixtures/valid.fstab --capture - > "$tmp/case.txtar"
grep -F -- '-- manifest.json --' "$tmp/case.txtar"
"${run[@]}" -i "$image" verify --snapshot - < "$tmp/case.txtar" > "$tmp/replay"
grep -F 'Validation successful.' "$tmp/replay"
test ! -s "$tmp/fixtures/valid.fstab"
printf 'Both architectures contain the release binaries and notices; native isolated runtime checks passed.\n'
