import re

def modify_file(filepath, search_replace_pairs):
    with open(filepath, 'r') as f:
        content = f.read()

    for search, replace in search_replace_pairs:
        if search in content:
            content = content.replace(search, replace)
        else:
            print(f"Could not find search string in {filepath}:\n{search}\n")

    with open(filepath, 'w') as f:
        f.write(content)

# runner_test.txtar
modify_file("internal/rules/testdata/runner.txtar", [
    (
"""-- expected/output.txt --
Validation issues found:
 - [error] Unknown filesystem type 'unknown' on line 4 (Line: /dev/sda3       /mnt            unknown defaults        0       2)
""",
"""-- expected/output.txt --
Validation issues found:
 - [error] Unknown filesystem type 'unknown' on line 4 (Line: /dev/sda3       /mnt            unknown defaults        0       2)"""
    )
])
