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

		// Valued generic options
		"context": StringOption{}, "defcontext": StringOption{}, "rootcontext": StringOption{},
		"fscontext": StringOption{}, "fstab": StringOption{}, "map-groups": StringOption{},
		"map-users": StringOption{}, "mkdir": StringOption{}, "namespace": StringOption{},
		"options": StringOption{}, "options-mode": StringOption{}, "options-source": StringOption{},
		"source": StringOption{}, "target": StringOption{}, "target-prefix": StringOption{},
		"test-opts": StringOption{}, "types": StringOption{}, "uuid": StringOption{},
	}
	return opts
}
