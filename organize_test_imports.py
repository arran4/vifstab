import os

with open('internal/rules/runner_test.go', 'r') as f:
    data = f.read()

data = data.replace('github.com/arran4/vifstab/internal/rules/builtin/all', 'github.com/arran4/vifstab/internal/rules/builtin/all')

with open('internal/rules/runner_test.go', 'w') as f:
    f.write(data)
