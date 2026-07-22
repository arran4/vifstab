#!/bin/bash
echo "package builtin" > internal/rules/builtin/imports.go
echo "" >> internal/rules/builtin/imports.go
echo "import (" >> internal/rules/builtin/imports.go
find internal/rules/builtin -mindepth 1 -type d | while read dir; do
    if [ -f "$dir/$(basename $dir).go" ]; then
        pkg="github.com/arran4/vifstab/${dir}"
        echo "	_ \"${pkg}\"" >> internal/rules/builtin/imports.go
    fi
done
echo ")" >> internal/rules/builtin/imports.go
