//go:build windows

package platform

import "os/exec"

func OpenExternal(target string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
}
