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
		"beneath": NoValueOption{}, "bind": NoValueOption{}, "exclusive": NoValueOption{},
		"fake": NoValueOption{}, "fork": NoValueOption{}, "help": NoValueOption{},
		"internal-only": NoValueOption{}, "loud": NoValueOption{}, "make-private": NoValueOption{},
		"make-rprivate": NoValueOption{}, "make-rshared": NoValueOption{}, "make-rslave": NoValueOption{},
		"make-runbindable": NoValueOption{}, "make-shared": NoValueOption{}, "make-slave": NoValueOption{},
		"make-unbindable": NoValueOption{}, "no-canonicalize": NoValueOption{}, "no-mtab": NoValueOption{},
		"nomand": NoValueOption{}, "nosymfollow": NoValueOption{}, "onlyonce": NoValueOption{},
		"options-source-force": NoValueOption{}, "rbind": NoValueOption{}, "read-only": NoValueOption{},
		"read-write": NoValueOption{}, "remount": NoValueOption{}, "show-labels": NoValueOption{},
		"silent": NoValueOption{}, "verbose": NoValueOption{}, "version": NoValueOption{},
		"X-": NoValueOption{}, "x-": NoValueOption{},

		// Valued generic options marked for future detailed validation
		"context":        TODOOption{Comment: "validate SELinux context string"},
		"defcontext":     TODOOption{Comment: "validate SELinux default context string"},
		"rootcontext":    TODOOption{Comment: "validate SELinux root context string"},
		"fscontext":      TODOOption{Comment: "validate SELinux fs context string"},
		"fstab":          TODOOption{Comment: "validate fstab file path"},
		"map-groups":     TODOOption{Comment: "validate group mapping file/string"},
		"map-users":      TODOOption{Comment: "validate user mapping file/string"},
		"mkdir":          TODOOption{Comment: "validate octal mode for auto mkdir"},
		"namespace":      TODOOption{Comment: "validate namespace string"},
		"options":        TODOOption{Comment: "validate nested mount options string"},
		"options-mode":   TODOOption{Comment: "validate options mode string"},
		"options-source": TODOOption{Comment: "validate options source path"},
		"source":         TODOOption{Comment: "validate mount source path/device"},
		"target":         TODOOption{Comment: "validate mount target directory"},
		"target-prefix":  TODOOption{Comment: "validate target prefix path"},
		"test-opts":      TODOOption{Comment: "validate test options string"},
		"types":          TODOOption{Comment: "validate filesystem types list"},
		"uuid":           TODOOption{Comment: "validate UUID format"},
	}
	return opts
}
