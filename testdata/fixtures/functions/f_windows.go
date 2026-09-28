//go:build windows

package functions

func platform() string { return "windows" }

var hook = func() string { return "windows" }
