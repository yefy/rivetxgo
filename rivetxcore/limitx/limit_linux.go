//go:build linux
// +build linux

package limitx

import (
	"github.com/yefy/log4go/ee"
	"github.com/yefy/log4go/log4"
	"golang.org/x/sys/unix"
	"syscall"
)

func SetUlimit(ConfigLimit *ConfigLimit) error {
	if ConfigLimit == nil {
		ConfigLimit = &config.ConfigLimit{
			OpenFileLimitSoft: 512 * 1024,             //524288
			OpenFileLimitHard: 512 * 1024,             //524288
			MemlockRlimitCurr: 1 * 1024 * 1024 * 1024, // 1G  1073741824
			MemlockRlimitMax:  2 * 1024 * 1024 * 1024, // 2G  2147483648
		}
	}
	log4.Info("Current unix.RLIM_INFINITY:%v", uint64(unix.RLIM_INFINITY))

	if ConfigLimit.OpenFileLimitSoft > 0 && ConfigLimit.OpenFileLimitHard > 0 {
		var rLimit syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			return ee.New(err, "Error getting rlimit")
		}

		log4.Info("Current Limit:%v", rLimit)
		if ConfigLimit.OpenFileLimitSoft > rLimit.Cur || ConfigLimit.OpenFileLimitHard > rLimit.Max {
			OpenFileLimitSoft := ConfigLimit.OpenFileLimitSoft
			if OpenFileLimitSoft < rLimit.Cur {
				OpenFileLimitSoft = rLimit.Cur
			}

			OpenFileLimitHard := ConfigLimit.OpenFileLimitHard
			if OpenFileLimitHard < rLimit.Max {
				OpenFileLimitHard = rLimit.Max
			}

			if OpenFileLimitSoft > OpenFileLimitHard {
				OpenFileLimitSoft = OpenFileLimitHard
			}

			rLimit.Cur = OpenFileLimitSoft
			rLimit.Max = OpenFileLimitHard
			err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
			if err != nil {
				rLimit.Cur = 65535
				rLimit.Max = 65535
				err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
				if err != nil {
					return ee.New(err, "Error setting rlimit")
				}
			}

			log4.Info("New Limit:%v", rLimit)
		}
	}

	if ConfigLimit.MemlockRlimitCurr > 0 && ConfigLimit.MemlockRlimitMax > 0 {
		var rLimit unix.Rlimit
		err := unix.Getrlimit(unix.RLIMIT_MEMLOCK, &rLimit)
		if err != nil {
			return ee.New(err, "Error getting RLIMIT_MEMLOCK")
		}

		log4.Info("Current RLIMIT_MEMLOCK:%v", rLimit)
		if ConfigLimit.MemlockRlimitCurr > rLimit.Cur || ConfigLimit.MemlockRlimitMax > rLimit.Max {
			MemlockRlimitCurr := ConfigLimit.MemlockRlimitCurr
			if MemlockRlimitCurr < rLimit.Cur {
				MemlockRlimitCurr = rLimit.Cur
			}

			MemlockRlimitMax := ConfigLimit.MemlockRlimitMax
			if MemlockRlimitMax < rLimit.Max {
				MemlockRlimitMax = rLimit.Max
			}

			if MemlockRlimitCurr > MemlockRlimitMax {
				MemlockRlimitCurr = MemlockRlimitMax
			}

			rLimit.Cur = MemlockRlimitCurr
			rLimit.Max = MemlockRlimitMax
			err = unix.Setrlimit(unix.RLIMIT_MEMLOCK, &rLimit)
			if err != nil {
				rLimit.Cur = rLimit.Cur / 2
				rLimit.Max = rLimit.Max / 2
				err = unix.Setrlimit(unix.RLIMIT_MEMLOCK, &rLimit)
				if err != nil {
					return ee.New(err, "Error setting RLIMIT_MEMLOCK")
				}
			}

			log4.Info("New RLIMIT_MEMLOCK:%v", rLimit)
		}
	}

	return nil
}
