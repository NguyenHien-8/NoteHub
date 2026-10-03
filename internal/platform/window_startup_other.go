//go:build !windows

package platform

func prepareWindowForShow() func() { return func() {} }
