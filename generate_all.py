import os

base_dir = "internal/rules/builtin"
imports = []

for root, dirs, files in os.walk(base_dir):
    for dir in dirs:
        # Ignore testdata and all
        if dir in ['testdata', 'all'] or root.endswith('testdata'):
            continue

        full_path = os.path.join(root, dir)
        # Check if it has go files and not a leaf directory we don't want (like we only want the actual fs dirs)
        has_go = any(f.endswith('.go') for f in os.listdir(full_path))
        if has_go:
            import_path = full_path.replace(base_dir, "github.com/arran4/vifstab/internal/rules/builtin")
            imports.append(f'\t_ "{import_path}"')

os.makedirs(os.path.join(base_dir, "all"), exist_ok=True)
with open(os.path.join(base_dir, "all", "imports.go"), "w") as f:
    f.write('package all\n\nimport (\n')
    for imp in imports:
        f.write(imp + '\n')
    f.write(')\n')
