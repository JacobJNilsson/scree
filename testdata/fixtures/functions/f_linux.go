//go:build linux

package functions

func platform() string { return "linux" }

var hook = func() string { return "linux" }
