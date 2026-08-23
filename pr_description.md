🎯 **What:** The vulnerability fixed
Command injection via an unsanitized environment variable (`EDITOR`, `VISUAL`, `SUDO_EDITOR`). An attacker could provide arbitrary commands or shell commands and have them executed without sandboxing via `exec.Command` in the tool's context.

⚠️ **Risk:** The potential impact if left unfixed
If left unfixed, this enables privilege escalation when `vifstab` is executed with higher privileges (e.g., via `sudo`). An attacker can gain arbitrary code execution or a root shell since the editor process runs as the elevated user.

🛡️ **Solution:** How the fix addresses the vulnerability
Added a strict `validateEditor` function that validates the configured editor against a known allowlist (`vi`, `vim`, `nvim`, `nano`, `pico`, `emacs`, `mcedit`, `ed`). It extracts the base name of the provided editor, ensuring paths cannot bypass the list. Additionally, it explicitly rejects any arguments in the editor configuration to completely prevent shell injections, flag manipulations, or appending multiple commands.
