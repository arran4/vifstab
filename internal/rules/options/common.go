package options

// CommonOptions returns standard mount options common to Linux filesystems.
func CommonOptions() map[string]OptionValidator {
	opts := map[string]OptionValidator{
		// Flags / No value options
		"defaults": NoValueOption{}, "rw": NoValueOption{}, "ro": NoValueOption{},
		"noatime": NoValueOption{}, "relatime": NoValueOption{}, "suid": NoValueOption{},
		"nosuid": NoValueOption{}, "exec": NoValueOption{}, "noexec": NoValueOption{},
		"auto": NoValueOption{}, "noauto": NoValueOption{}, "user": NoValueOption{},
		"nouser": NoValueOption{}, "sync": NoValueOption{}, "async": NoValueOption{},
		"dirsync": NoValueOption{}, "nodev": NoValueOption{}, "lazytime": NoValueOption{},
		"iversion": NoValueOption{}, "nofail": NoValueOption{}, "strictatime": NoValueOption{},
		"nodiratime": NoValueOption{}, "users": NoValueOption{}, "_netdev": NoValueOption{},
		"dev": NoValueOption{}, "atime": NoValueOption{}, "diratime": NoValueOption{},
		"noiversion": NoValueOption{}, "nolazytime": NoValueOption{}, "owner": NoValueOption{},
		"group": NoValueOption{},
		"bind":  NoValueOption{},

		"loud": NoValueOption{},

		"nomand": NoValueOption{}, "nosymfollow": NoValueOption{},
		"rbind":   NoValueOption{},
		"remount": NoValueOption{},
		"silent":  NoValueOption{},
		"X-":      NoValueOption{}, "x-": NoValueOption{},

		// Valued generic options marked for future detailed validation
		"context":     TODOOption{Comment: "validate SELinux context string"},
		"defcontext":  TODOOption{Comment: "validate SELinux default context string"},
		"rootcontext": TODOOption{Comment: "validate SELinux root context string"},
		"fscontext":   TODOOption{Comment: "validate SELinux fs context string"},

		"X-mount.mkdir": OptionalOctalOption{},
		"x-mount.mkdir": OptionalOctalOption{},
	}
	return opts
}
