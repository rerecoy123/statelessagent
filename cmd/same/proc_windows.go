//go:build windows

package main

import "syscall"

func backgroundProcessSysProcAttr() *syscall.SysProcAttr {
	// CREATE_NEW_PROCESS_GROUP
	return &syscall.SysProcAttr{CreationFlags: 0x00000200}
}
