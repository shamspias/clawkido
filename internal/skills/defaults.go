package skills

// RegisterDefaults registers all built-in skills into the registry.
// Call this once at startup from main.go.
func RegisterDefaults(r *Registry) {
	// Read-only skills (safe).
	r.Register(TimeSkill{})
	r.Register(HTTPGetSkill{})
	r.Register(FileReadSkill{})
	r.Register(DirListSkill{})
	r.Register(EnvSkill{})
	r.Register(NewSysInfoSkill())

	// Write skills (caution).
	r.Register(ShellSkill{})

	// Destructive skills (confirmation required).
	r.Register(MemoryResetSkill{})
	r.Register(FileWriteSkill{})
	r.Register(FileDeleteSkill{})

	// Meta skills (self-referencing).
	r.Register(NewHelpSkill(r))
}
