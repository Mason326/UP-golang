package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID     int
	PPID    int
	User    string
	TTY     string
	Stat    string
	Time    string
	Command string
	VSZ     int
	RSS     int
	CPU     float64
	Memory  float64
}

type PsOptions struct {
	All     bool
	ListAll bool
	Full    bool
	User    string
	Help    bool
}

func main() {
	opts := parseFlags()

	if opts.Help {
		printHelp()
		return
	}

	processes, err := getRealProcesses(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ps: error: %v\n", err)
		os.Exit(1)
	}

	// Фильтрация по пользователю если указано
	if opts.User != "" {
		processes = filterByUser(processes, opts.User)
	}

	// Вывод
	printProcesses(processes, opts)
}

func parseFlags() *PsOptions {
	opts := &PsOptions{}

	flag.BoolVar(&opts.All, "A", false, "show all processes")
	flag.BoolVar(&opts.All, "e", false, "show all processes")
	flag.BoolVar(&opts.ListAll, "a", false, "show all processes with tty")
	flag.BoolVar(&opts.Full, "f", false, "full format")
	flag.StringVar(&opts.User, "u", "", "show processes for user")
	flag.BoolVar(&opts.Help, "help", false, "show help")

	flag.Parse()
	return opts
}

func getRealProcesses(opts *PsOptions) ([]ProcessInfo, error) {
	var processes []ProcessInfo

	// Читаем директорию /proc
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("cannot read /proc: %v", err)
	}

	currentUser, _ := user.Current()
	currentUsername := ""
	if currentUser != nil {
		currentUsername = currentUser.Username
	}

	// Получаем текущий TTY
	currentTTY := getCurrentTTY()

	for _, file := range files {
		// Проверяем, является ли имя директории числом (PID)
		pid, err := strconv.Atoi(file.Name())
		if err != nil || !file.IsDir() {
			continue
		}

		// Читаем информацию о процессе
		proc, err := readProcessInfo(pid)
		if err != nil {
			// Пропускаем процессы, которые завершились
			continue
		}

		// Применяем фильтры
		if !shouldIncludeProcess(proc, opts, currentUsername, currentTTY) {
			continue
		}

		processes = append(processes, proc)
	}

	return processes, nil
}

func shouldIncludeProcess(proc ProcessInfo, opts *PsOptions, currentUser, currentTTY string) bool {
	if opts.All {
		return true
	}

	if opts.ListAll {
		// -a: все процессы с TTY (любого терминала)
		return proc.TTY != "?"
	}

	if opts.User != "" {
		return proc.User == opts.User
	}

	// По умолчанию: только процессы текущего пользователя в ТЕКУЩЕМ TTY
	if proc.User != currentUser {
		return false
	}
	
	// Сравниваем TTY процесса с текущим TTY
	// Нормализуем TTY для сравнения
	procTTY := normalizeTTY(proc.TTY)
	currentTTYNormalized := normalizeTTY(currentTTY)
	
	return procTTY == currentTTYNormalized
}

func normalizeTTY(tty string) string {
	if tty == "" || tty == "?" {
		return tty
	}
	// Убираем префикс /dev/ если есть
	tty = strings.TrimPrefix(tty, "/dev/")
	// Убираем префикс pts/ для сравнения
	tty = strings.TrimPrefix(tty, "pts/")
	// Убираем префикс tty для сравнения
	tty = strings.TrimPrefix(tty, "tty")
	return tty
}

func getCurrentTTY() string {
	// Пробуем несколько способов определить текущий TTY
	
	// 1. Через /proc/self/fd/0 (самый надежный способ)
	if link, err := os.Readlink("/proc/self/fd/0"); err == nil {
		if strings.Contains(link, "/dev/pts/") || strings.Contains(link, "/dev/tty") {
			return filepath.Base(link)
		}
	}
	
	// 2. Через команду tty
	if output, err := exec.Command("tty").Output(); err == nil {
		tty := strings.TrimSpace(string(output))
		if strings.HasPrefix(tty, "/dev/") {
			return filepath.Base(tty)
		}
		return tty
	}
	
	// 3. Через переменные окружения
	if tty := os.Getenv("SSH_TTY"); tty != "" {
		return filepath.Base(tty)
	}
	
	// 4. Если мы в SSH сессии
	if os.Getenv("SSH_CONNECTION") != "" {
		// Пытаемся получить pts из ps
		cmd := exec.Command("ps", "-o", "tty=", "-p", strconv.Itoa(os.Getpid()))
		if output, err := cmd.Output(); err == nil {
			tty := strings.TrimSpace(string(output))
			if tty != "?" && tty != "" {
				return tty
			}
		}
	}
	
	// 5. Fallback - пытаемся угадать
	return "pts/0"
}

func readProcessInfo(pid int) (ProcessInfo, error) {
	proc := ProcessInfo{PID: pid}

	// Читаем /proc/[pid]/stat
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statData, err := ioutil.ReadFile(statPath)
	if err != nil {
		return proc, err
	}

	// Парсим /proc/[pid]/stat
	statStr := string(statData)
	firstParen := strings.Index(statStr, "(")
	lastParen := strings.LastIndex(statStr, ")")
	
	if firstParen == -1 || lastParen == -1 {
		return proc, fmt.Errorf("invalid stat format")
	}

	comm := statStr[firstParen+1 : lastParen]
	fields := strings.Fields(statStr[lastParen+2:])
	
	if len(fields) < 20 {
		return proc, fmt.Errorf("not enough fields in stat")
	}

	proc.Stat = fields[0]
	ppid, _ := strconv.Atoi(fields[1])
	proc.PPID = ppid
	
	ttyNr, _ := strconv.ParseUint(fields[4], 10, 64)
	proc.TTY = getTTYFromNumber(int(ttyNr))
	
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	totalTime := utime + stime
	
	proc.Time = formatCPUTime(totalTime)
	
	vsz, _ := strconv.Atoi(fields[20])
	proc.VSZ = vsz
	rss, _ := strconv.Atoi(fields[21])
	proc.RSS = rss * 4096

	// Читаем команду
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdlineData, err := ioutil.ReadFile(cmdlinePath)
	if err == nil && len(cmdlineData) > 0 {
		cmdline := strings.ReplaceAll(string(cmdlineData), "\x00", " ")
		cmdline = strings.TrimSpace(cmdline)
		if cmdline == "" {
			proc.Command = fmt.Sprintf("[%s]", comm)
		} else {
			proc.Command = cmdline
		}
	} else {
		proc.Command = fmt.Sprintf("[%s]", comm)
	}

	// Получаем пользователя
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	statusData, err := ioutil.ReadFile(statusPath)
	if err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(statusData))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Uid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					uid, _ := strconv.Atoi(fields[1])
					if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
						proc.User = u.Username
					} else {
						proc.User = fields[1]
					}
				}
				break
			}
		}
	}

	proc.CPU = 0.0 // Упрощаем - не считаем CPU%
	proc.Memory = float64(proc.RSS) / 1024 / 1024

	return proc, nil
}

func getTTYFromNumber(ttyNr int) string {
	if ttyNr == 0 {
		return "?"
	}
	
	major := (ttyNr >> 8) & 0xFF
	minor := ttyNr & 0xFF
	
	if major == 4 {
		return fmt.Sprintf("tty%d", minor)
	} else if major == 136 {
		return fmt.Sprintf("pts/%d", minor)
	} else if major == 3 {
		return fmt.Sprintf("tty%d", minor)
	}
	
	// Для консольных tty
	if ttyNr < 64 {
		return fmt.Sprintf("tty%d", ttyNr)
	}
	
	return "?"
}

func formatCPUTime(clockTicks uint64) string {
	seconds := clockTicks / 100
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func filterByUser(processes []ProcessInfo, username string) []ProcessInfo {
	var filtered []ProcessInfo
	for _, proc := range processes {
		if proc.User == username {
			filtered = append(filtered, proc)
		}
	}
	return filtered
}

func printProcesses(processes []ProcessInfo, opts *PsOptions) {
	if len(processes) == 0 {
		return
	}

	if opts.Full {
		printFullFormat(processes)
	} else {
		printDefaultFormat(processes)
	}
}

func printDefaultFormat(processes []ProcessInfo) {
	// Определяем максимальную ширину для PID
	maxPID := 0
	for _, proc := range processes {
		if proc.PID > maxPID {
			maxPID = proc.PID
		}
	}
	
	pidWidth := len(strconv.Itoa(maxPID))
	if pidWidth < 5 {
		pidWidth = 5
	}
	
	// Заголовок
	fmt.Printf("%*s %-8s %-10s %s\n", pidWidth, "PID", "TTY", "TIME", "CMD")
	
	// Процессы
	for _, proc := range processes {
		// Обрезаем длинные команды
		cmd := proc.Command
		if len(cmd) > 50 {
			cmd = cmd[:47] + "..."
		}
		
		fmt.Printf("%*d %-8s %-10s %s\n", 
			pidWidth, proc.PID, 
			proc.TTY, 
			proc.Time, 
			cmd)
	}
}

func printFullFormat(processes []ProcessInfo) {
	// Определяем максимальную ширину для PID
	maxPID := 0
	for _, proc := range processes {
		if proc.PID > maxPID {
			maxPID = proc.PID
		}
	}
	
	pidWidth := len(strconv.Itoa(maxPID))
	if pidWidth < 5 {
		pidWidth = 5
	}
	
	// Заголовок
	fmt.Printf("%-8s %*s %-8s %-4s %-8s %-8s %-10s %s\n",
		"UID", pidWidth, "PID", "PPID", "C", "STIME", "TTY", "TIME", "CMD")
	
	// Процессы
	for _, proc := range processes {
		stime := "00:00"
		if len(proc.Time) >= 5 {
			stime = proc.Time[:5]
		}
		
		cmd := proc.Command
		if len(cmd) > 30 {
			cmd = cmd[:27] + "..."
		}
		
		fmt.Printf("%-8s %*d %-8d %-4.1f %-8s %-8s %-10s %s\n",
			proc.User,
			pidWidth, proc.PID,
			proc.PPID,
			proc.CPU,
			stime,
			proc.TTY,
			proc.Time,
			cmd)
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

Поведение по умолчанию:
  ps               процессы текущего пользователя в текущем терминале
  ps -A или ps -e  все процессы
  ps -a            все процессы с терминалом
  ps -u user       процессы указанного пользователя
  ps -f            полный формат с дополнительной информацией

Примеры:
  ps               # Процессы текущего пользователя в текущем терминале
  ps -f            # Полный формат
  ps -A            # Все процессы
  ps -a            # Все процессы с терминалом
  ps -u root       # Процессы пользователя root
  ps -u user -f    # Полный формат для процессов пользователя user
  ps --help        # Справка
`

	fmt.Println(helpText)
}
