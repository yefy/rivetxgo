package limitx

type ConfigLimit struct {
	OpenFileLimitSoft uint64 `yaml:"open_file_limit_soft"`
	OpenFileLimitHard uint64 `yaml:"open_file_limit_hard"`
	MemlockRlimitCurr uint64 `yaml:"memlock_rlimit_curr"`
	MemlockRlimitMax  uint64 `yaml:"memlock_rlimit_max"`
}
