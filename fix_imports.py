files = ["internal/editor/editor.go", "internal/editor/verify.go", "internal/rules/runner_test.go"]

for f in files:
    with open(f, "r") as file:
        data = file.read()

    # We want exactly one of each. So let's first remove them all
    lines = data.split('\n')
    new_lines = []

    builtin_found = False
    all_found = False

    in_import_block = False

    for line in lines:
        if 'import (' in line:
            in_import_block = True
            new_lines.append(line)
            new_lines.append('\t_ "github.com/arran4/vifstab/internal/rules/builtin"')
            new_lines.append('\t_ "github.com/arran4/vifstab/internal/rules/builtin/all"')
            continue

        if in_import_block and ')' in line:
            in_import_block = False
            new_lines.append(line)
            continue

        if in_import_block and '_ "github.com/arran4/vifstab/internal/rules/builtin"' in line:
            continue

        if in_import_block and '_ "github.com/arran4/vifstab/internal/rules/builtin/all"' in line:
            continue

        # Clean up double blank lines in imports
        if in_import_block and line.strip() == "":
            continue

        new_lines.append(line)

    with open(f, "w") as file:
        file.write('\n'.join(new_lines))
