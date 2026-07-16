//go:build !windows

package output

func enableVT() bool { return true }
