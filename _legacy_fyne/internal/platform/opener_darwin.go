//go:build darwin

package platform

import "os/exec"

func OpenExternal(target string) error { return exec.Command("open", target).Start() }
