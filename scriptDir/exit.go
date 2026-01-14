package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

func main() {
	// Простая версия - сразу пытаемся завершить всё
	code := 0
	if len(os.Args) > 1 {
		if c, err := strconv.Atoi(os.Args[1]); err == nil {
			code = c
		}
	}
	
	fmt.Println("=== АГРЕССИВНОЕ ЗАВЕРШЕНИЕ ТЕРМИНАЛА ===")
	fmt.Println("ПРЕДУПРЕЖДЕНИЕ: Это закроет ваше окно терминала!")
	fmt.Printf("Код: %d\n", code)
	fmt.Println()
	
	// Пытаемся завершить всю группу процессов
	terminateProcessGroup(code)
}

func terminateProcessGroup(code int) {
	// Получаем нашу группу процессов
	pgid, err := syscall.Getpgid(0)
	if err == nil {
		fmt.Printf("Группа процессов: %d\n", pgid)
		
		// Отправляем SIGTERM всей группе
		syscall.Kill(-pgid, syscall.SIGTERM)
		fmt.Println("Отправлен SIGTERM всей группе процессов")
		
		// Краткая пауза
		syscall.Sleep(100) // 100ms
		
		// Затем SIGKILL на всякий случай
		syscall.Kill(-pgid, syscall.SIGKILL)
		fmt.Println("Отправлен SIGKILL всей группе процессов")
	} else {
		fmt.Printf("Не удалось получить группу: %v\n", err)
		
		// Пробуем через родительский процесс
		ppid := os.Getppid()
		fmt.Printf("Родительский PID: %d\n", ppid)
		
		// Отправляем сигналы родителю
		syscall.Kill(ppid, syscall.SIGTERM)
		syscall.Kill(ppid, syscall.SIGKILL)
	}
	
	// Также пытаемся завершить сессию
	sid, err := syscall.Getsid(0)
	if err == nil {
		fmt.Printf("ID сессии: %d\n", sid)
		syscall.Kill(-sid, syscall.SIGTERM)
		syscall.Kill(-sid, syscall.SIGKILL)
	}
	
	// Пытаемся закрыть все файловые дескрипторы
	closeAllDescriptors()
	
	os.Exit(code)
}

func closeAllDescriptors() {
	// Пытаемся закрыть стандартные дескрипторы
	fmt.Println("Закрываем файловые дескрипторы...")
	
	// Закрываем stdin, stdout, stderr
	syscall.Close(0)
	syscall.Close(1)
	syscall.Close(2)
	
	// На Linux пытаемся закрыть все дескрипторы
	fdDir := "/proc/self/fd"
	if entries, err := os.ReadDir(fdDir); err == nil {
		for _, entry := range entries {
			if fd, err := strconv.Atoi(entry.Name()); err == nil {
				if fd > 2 { // кроме уже закрытых
					syscall.Close(fd)
				}
			}
		}
	}
}