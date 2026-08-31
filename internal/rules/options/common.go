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
		"group":   NoValueOption{},
		"bind": NoValueOption{}, "rbind": NoValueOption{}, "remount": NoValueOption{},
		"nosymfollow": NoValueOption{}, "silent": NoValueOption{}, "loud": NoValueOption{},
		"nomand": NoValueOption{},

		"private": NoValueOption{}, "rprivate": NoValueOption{},
		"slave": NoValueOption{}, "rslave": NoValueOption{},
		"shared": NoValueOption{}, "rshared": NoValueOption{},
		"unbindable": NoValueOption{}, "runbindable": NoValueOption{},

		"X-": NoValueOption{}, "x-": NoValueOption{},

		// Valued generic options marked for future detailed validation
		"context":        SELinuxContextOption{},
		"defcontext":     SELinuxContextOption{},
		"rootcontext":    SELinuxContextOption{},
		"fscontext":      SELinuxContextOption{},

		"comment":        StringOption{},
		"X-mount.mkdir":  OptionalOctalOption{},
		"x-mount.mkdir":  OptionalOctalOption{},
	}
	return opts
}
