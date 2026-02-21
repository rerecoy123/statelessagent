//go:build !windows

package main

import "syscall"

func backgroundProcessSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
