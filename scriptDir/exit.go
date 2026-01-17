package main

import (
	"os"
	"syscall"
)

func main() {
    pid := os.Getppid()
    syscall.Kill(pid, syscall.SIGHUP) 	
}
