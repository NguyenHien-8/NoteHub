//go:build linux

package platform

import "os/exec"

func OpenExternal(target string) error { return exec.Command("xdg-open", target).Start() }
