//go:build windows
// +build windows

package limitx

func SetUlimit(ConfigLimit *ConfigLimit) error {
	return nil
}
