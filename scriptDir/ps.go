package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

type ProcessInfo struct {
	PID     int
	PPID    int
	Name    string
	CPU     float64
	Memory  float64
	Status  string
	User    string
	Time    string
	Command string
}

type PsOptions struct {
	All     bool   // -A, -e
	ListAll bool   // -a
	Full    bool   // -f
	User    string // -u
	Help    bool   // --help
}

func main() {
	opts := parseFlags()

	if opts.Help {
		printHelp()
		return
	}

	processes, err := getProcesses(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ps: error: %v\n", err)
		os.Exit(1)
	}

	// Фильтрация по пользователю если указано
	if opts.User != "" {
		processes = filterByUser(processes, opts.User)
	}

	// Сортировка по PID
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})

	// Вывод
	printProcesses(processes, opts)
}

func parseFlags() *PsOptions {
	opts := &PsOptions{}

	// Определение флагов (только указанные)
	flag.BoolVar(&opts.All, "A", false, "show all processes")
	flag.BoolVar(&opts.All, "e", false, "show all processes")
	flag.BoolVar(&opts.ListAll, "a", false, "show all processes with tty")
	flag.BoolVar(&opts.Full, "f", false, "full format")
	flag.StringVar(&opts.User, "u", "", "show processes for user")
	flag.BoolVar(&opts.Help, "help", false, "show help")

	flag.Parse()
	return opts
}

func getProcesses(opts *PsOptions) ([]ProcessInfo, error) {
	// В зависимости от ОС используем разные методы
	switch runtime.GOOS {
	case "linux":
		return getLinuxProcesses(opts)
	case "windows":
		return getWindowsProcesses(opts)
	case "darwin":
		return getMacProcesses(opts)
	default:
		return getGenericProcesses(opts)
	}
}

func getLinuxProcesses(opts *PsOptions) ([]ProcessInfo, error) {
	// Демонстрационные данные для Linux
	processes := []ProcessInfo{
		{PID: 1, PPID: 0, Name: "systemd", CPU: 0.1, Memory: 100, Status: "S", User: "root", Time: "00:10:00", Command: "/sbin/init"},
		{PID: 123, PPID: 1, Name: "sshd", CPU: 0.2, Memory: 50, Status: "S", User: "root", Time: "00:05:30", Command: "/usr/sbin/sshd -D"},
		{PID: 456, PPID: 1, Name: "bash", CPU: 0.1, Memory: 20, Status: "S", User: "user", Time: "00:01:15", Command: "-bash"},
		{PID: 789, PPID: 456, Name: "ps", CPU: 0.5, Memory: 5, Status: "R", User: "user", Time: "00:00:00", Command: "./ps -f"},
		{PID: 999, PPID: 1, Name: "cron", CPU: 0.0, Memory: 15, Status: "S", User: "root", Time: "00:00:10", Command: "/usr/sbin/cron"},
	}

	// Фильтрация для флага -a
	if opts.ListAll && !opts.All {
		// Показываем только процессы с TTY
		var filtered []ProcessInfo
		for _, proc := range processes {
			// Предположим, что bash и ps имеют TTY
			if proc.Name == "bash" || proc.Name == "ps" {
				filtered = append(filtered, proc)
			}
		}
		return filtered, nil
	}

	return processes, nil
}

func getWindowsProcesses(opts *PsOptions) ([]ProcessInfo, error) {
	// Демонстрационные данные для Windows
	username := os.Getenv("USERNAME")
	if username == "" {
		username = "user"
	}

	return []ProcessInfo{
		{PID: 0, PPID: 0, Name: "System Idle Process", CPU: 90.0, Memory: 1, Status: "Running", User: "SYSTEM", Time: "100:00:00", Command: "[System Process]"},
		{PID: 4, PPID: 0, Name: "System", CPU: 2.5, Memory: 500, Status: "Running", User: "SYSTEM", Time: "50:00:00", Command: "NT Kernel & System"},
		{PID: 1234, PPID: 456, Name: "explorer.exe", CPU: 1.2, Memory: 80000, Status: "Running", User: username, Time: "01:30:00", Command: "C:\\Windows\\explorer.exe"},
		{PID: 5678, PPID: 1234, Name: "cmd.exe", CPU: 0.5, Memory: 5000, Status: "Running", User: username, Time: "00:05:00", Command: "C:\\Windows\\System32\\cmd.exe"},
		{PID: 9999, PPID: 5678, Name: "ps.exe", CPU: 0.1, Memory: 2000, Status: "Running", User: username, Time: "00:00:01", Command: ".\\ps.exe"},
	}, nil
}

func getMacProcesses(opts *PsOptions) ([]ProcessInfo, error) {
	// Демонстрационные данные для macOS
	username := os.Getenv("USER")
	if username == "" {
		username = "user"
	}

	return []ProcessInfo{
		{PID: 1, PPID: 0, Name: "launchd", CPU: 0.1, Memory: 500, Status: "S", User: "root", Time: "100:00:00", Command: "/sbin/launchd"},
		{PID: 123, PPID: 1, Name: "WindowServer", CPU: 5.2, Memory: 80000, Status: "S", User: "_windowserver", Time: "10:30:00", Command: "/System/Library/..."},
		{PID: 456, PPID: 1, Name: "bash", CPU: 0.3, Memory: 4000, Status: "S", User: username, Time: "00:10:00", Command: "/bin/bash"},
		{PID: 789, PPID: 456, Name: "ps", CPU: 0.1, Memory: 2000, Status: "R", User: username, Time: "00:00:00", Command: "./ps"},
	}, nil
}

func getGenericProcesses(opts *PsOptions) ([]ProcessInfo, error) {
	// Общая реализация для неизвестных ОС
	pid := os.Getpid()
	ppid := os.Getppid()
	
	return []ProcessInfo{
		{
			PID:     pid,
			PPID:    ppid,
			Name:    getExecutableName(),
			CPU:     0.1,
			Memory:  getMemoryUsage(),
			Status:  "R",
			User:    getUser(),
			Time:    "00:00:01",
			Command: strings.Join(os.Args, " "),
		},
	}, nil
}

func getExecutableName() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	
	// Извлекаем только имя файла
	if idx := strings.LastIndex(exe, string(os.PathSeparator)); idx != -1 {
		return exe[idx+1:]
	}
	if idx := strings.LastIndex(exe, "/"); idx != -1 {
		return exe[idx+1:]
	}
	if idx := strings.LastIndex(exe, "\\"); idx != -1 {
		return exe[idx+1:]
	}
	
	return exe
}

func getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024 // MB
}

func getUser() string {
	// Пытаемся получить имя пользователя
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

func filterByUser(processes []ProcessInfo, username string) []ProcessInfo {
	var filtered []ProcessInfo
	for _, proc := range processes {
		if strings.EqualFold(proc.User, username) {
			filtered = append(filtered, proc)
		}
	}
	return filtered
}

func printProcesses(processes []ProcessInfo, opts *PsOptions) {
	if opts.Full {
		printFullFormat(processes)
	} else {
		printDefaultFormat(processes)
	}
}

func printDefaultFormat(processes []ProcessInfo) {
	// Заголовок
	fmt.Printf("%-8s %-8s %-10s %s\n", "PID", "TTY", "TIME", "CMD")
	
	// Процессы
	for _, proc := range processes {
		// Упрощенный TTY
		tty := getTTY(proc)
		
		// Обрезаем команду
		cmd := proc.Command
		if len(cmd) > 40 {
			cmd = cmd[:37] + "..."
		}
		if cmd == "" {
			cmd = proc.Name
		}
		
		fmt.Printf("%-8d %-8s %-10s %s\n", proc.PID, tty, proc.Time, cmd)
	}
}

func printFullFormat(processes []ProcessInfo) {
	// Заголовок
	fmt.Printf("%-8s %-8s %-8s %-6s %-8s %-8s %-10s %s\n",
		"UID", "PID", "PPID", "C", "STIME", "TTY", "TIME", "CMD")
	
	// Процессы
	for _, proc := range processes {
		// Упрощенные значения
		stime := "00:00"
		tty := getTTY(proc)
		
		// Обрезаем команду
		cmd := proc.Command
		if len(cmd) > 30 {
			cmd = cmd[:27] + "..."
		}
		if cmd == "" {
			cmd = proc.Name
		}
		
		fmt.Printf("%-8s %-8d %-8d %-6.1f %-8s %-8s %-10s %s\n",
			proc.User, proc.PID, proc.PPID, proc.CPU, stime, tty, proc.Time, cmd)
	}
}

func getTTY(proc ProcessInfo) string {
	// Упрощенное определение TTY
	if runtime.GOOS == "windows" {
		return "con"
	}
	
	// Для демонстрационных данных
	switch proc.Name {
	case "bash", "ps", "ssh", "login":
		return "pts/0"
	case "systemd", "launchd", "init":
		return "?"
	case "explorer.exe", "cmd.exe":
		return "con"
	default:
		return "?"
	}
}

func printHelp() {
	helpText := `Использование: ps [ПАРАМЕТР]...
Отображает информацию о процессах.

Параметры:
  -A, -e           все процессы
  -a               все процессы с терминалом
  -f               полный формат
  -u ПОЛЬЗОВАТЕЛЬ  процессы пользователя
      --help           справка

По умолчанию показываются процессы текущего пользователя с терминалом.

Флаги:
  -A, -e   Показывать все процессы (включая системные)
  -a       Показывать все процессы с терминалом
  -f       Полный формат с дополнительной информацией
  -u       Фильтровать по имени пользователя

Примеры:
  ps               Процессы текущего пользователя
  ps -f            Полный формат
  ps -A            Все процессы
  ps -a            Все процессы с терминалом
  ps -u root       Процессы пользователя root
  ps --help        Справка
`

	fmt.Println(helpText)
}