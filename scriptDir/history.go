package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type HistoryEntry struct {
	ID      int
	Command string
}

type HistoryConfig struct {
	Limit  int
	Delete int
	Clear  bool
	Help   bool
}

func main() {
	config := parseFlags()

	if config.Help {
		printHelp()
		return
	}

	if config.Delete > 0 {
		deleteHistoryEntry(config.Delete)
		return
	}

	if config.Clear {
		clearHistory()
		return
	}

	entries := loadHistoryEntries()
	displayHistory(entries, config)
}

func parseFlags() HistoryConfig {
	var config HistoryConfig

	flag.IntVar(&config.Limit, "n", 0, "number of history entries to show")
	flag.IntVar(&config.Delete, "d", 0, "delete entry by number")
	flag.BoolVar(&config.Clear, "c", false, "clear history file")
	flag.BoolVar(&config.Help, "help", false, "show help")

	flag.Parse()
	return config
}

func getHistoryFiles() []string {
	var files []string

	// Получаем домашнюю директорию
	home, err := os.UserHomeDir()
	if err != nil {
		return files
	}

	// Основные файлы истории
	possibleFiles := []string{
		filepath.Join(home, ".bash_history"),
		filepath.Join(home, ".zsh_history"),
		filepath.Join(home, ".history"),
	}

	// Добавляем HISTFILE если установлен
	if histfile := os.Getenv("HISTFILE"); histfile != "" {
		possibleFiles = append([]string{histfile}, possibleFiles...)
	}

	// Проверяем существование файлов
	for _, file := range possibleFiles {
		if _, err := os.Stat(file); err == nil {
			files = append(files, file)
		}
	}

	return files
}

func getMainHistoryFile() string {
	files := getHistoryFiles()
	if len(files) > 0 {
		return files[0]
	}

	// Если файлов нет, создаем стандартный
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Проверяем, какая оболочка используется
	shell := os.Getenv("SHELL")
	defaultFile := ""

	if strings.Contains(shell, "bash") {
		defaultFile = filepath.Join(home, ".bash_history")
	} else if strings.Contains(shell, "zsh") {
		defaultFile = filepath.Join(home, ".zsh_history")
	} else {
		defaultFile = filepath.Join(home, ".history")
	}

	return defaultFile
}

func loadHistoryEntries() []HistoryEntry {
	historyFile := getMainHistoryFile()
	if historyFile == "" {
		return []HistoryEntry{}
	}

	return readHistoryFile(historyFile, 1)
}

func readHistoryFile(filename string, startID int) []HistoryEntry {
	file, err := os.Open(filename)
	if err != nil {
		return []HistoryEntry{}
	}
	defer file.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)
	currentID := startID

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Пропускаем строки с таймстемпами bash
		if strings.HasPrefix(line, "#") && len(line) > 1 {
			// Проверяем, является ли это числом (таймстемпом)
			if _, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
				continue
			}
		}

		// Обрабатываем zsh формат: ": timestamp;command"
		if strings.HasPrefix(line, ": ") {
			// Ищем точку с запятой
			if idx := strings.Index(line, ";"); idx != -1 {
				line = line[idx+1:]
			} else {
				continue
			}
		}

		entry := HistoryEntry{
			ID:      currentID,
			Command: line,
		}

		entries = append(entries, entry)
		currentID++
	}

	return entries
}

func displayHistory(entries []HistoryEntry, config HistoryConfig) {
	if len(entries) == 0 {
		fmt.Println("No history entries found")
		fmt.Println("\nTo populate history:")
		fmt.Println("1. Run commands in your shell")
		fmt.Println("2. For bash: run 'history -w' to write to file")
		fmt.Println("3. For zsh: run 'fc -W' to write to file")
		return
	}

	// Применяем limit
	displayEntries := entries
	if config.Limit > 0 {
		start := len(entries) - config.Limit
		if start < 0 {
			start = 0
		}
		displayEntries = entries[start:]
	}

	// Выводим историю
	fmt.Printf("Command History (showing %d of %d):\n", len(displayEntries), len(entries))
	fmt.Println("===================================")

	// Определяем максимальную ширину для номера
	maxID := len(displayEntries)
	if len(displayEntries) > 0 {
		maxID = displayEntries[len(displayEntries)-1].ID
	}
	idWidth := len(strconv.Itoa(maxID))

	for _, entry := range displayEntries {
		fmt.Printf("%*d  %s\n", idWidth, entry.ID, entry.Command)
	}
}

func clearHistory() {
	historyFile := getMainHistoryFile()
	if historyFile == "" {
		fmt.Println("Error: Cannot determine history file location")
		return
	}

	// Спрашиваем подтверждение
	fmt.Printf("Are you sure you want to clear %s? (y/N): ", historyFile)
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Clear cancelled.")
		return
	}

	// Показываем сколько записей будет удалено
	entries := loadHistoryEntries()
	entryCount := len(entries)
	
	// Создаем пустой файл
	file, err := os.Create(historyFile)
	if err != nil {
		fmt.Printf("Error clearing file: %v\n", err)
		return
	}
	file.Close()

	fmt.Printf("History file cleared: %d entries removed\n", entryCount)
	fmt.Println("File:", historyFile)
	
	// Предупреждение о истории в памяти shell
	fmt.Println("\nNote: This only clears the history file.")
	fmt.Println("To clear current shell session history, run:")
	fmt.Println("  bash: 'history -c'")
	fmt.Println("  zsh:  'fc -p' or start a new shell")
}

func deleteHistoryEntry(entryNum int) {
	historyFile := getMainHistoryFile()
	if historyFile == "" {
		fmt.Println("Error: Cannot determine history file location")
		return
	}

	// Читаем текущую историю
	entries := readHistoryFile(historyFile, 1)

	if entryNum <= 0 || entryNum > len(entries) {
		fmt.Printf("Error: Entry %d not found (history has %d entries)\n",
			entryNum, len(entries))
		return
	}

	// Показываем удаляемую команду
	entryToDelete := entries[entryNum-1]
	fmt.Printf("Deleting entry %d: %s\n", entryNum, entryToDelete.Command)

	// Спрашиваем подтверждение
	fmt.Print("Are you sure? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Deletion cancelled.")
		return
	}

	// Читаем исходный файл полностью
	srcFile, err := os.Open(historyFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer srcFile.Close()

	// Создаем временный файл
	tmpFile := historyFile + ".tmp"
	dstFile, err := os.Create(tmpFile)
	if err != nil {
		fmt.Printf("Error creating temp file: %v\n", err)
		srcFile.Close()
		return
	}
	defer dstFile.Close()

	scanner := bufio.NewScanner(srcFile)
	writer := bufio.NewWriter(dstFile)
	currentEntry := 1
	var skipNext bool

	for scanner.Scan() {
		line := scanner.Text()

		// Пропускаем связанные строки (таймстемпы)
		if skipNext {
			skipNext = false
			continue
		}

		// Проверяем таймстемп bash
		if strings.HasPrefix(line, "#") && len(line) > 1 {
			if _, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
				// Это таймстемп
				if currentEntry == entryNum {
					// Пропускаем таймстемп и следующую команду
					skipNext = true
					currentEntry++
					continue
				}
			}
		}

		// Проверяем zsh формат
		if strings.HasPrefix(line, ": ") {
			if currentEntry == entryNum {
				// Пропускаем эту запись zsh
				currentEntry++
				continue
			}
		}

		// Обычная команда
		if !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ": ") && strings.TrimSpace(line) != "" {
			if currentEntry == entryNum {
				// Пропускаем эту команду
				currentEntry++
				continue
			}
			currentEntry++
		}

		writer.WriteString(line + "\n")
	}

	writer.Flush()
	srcFile.Close()
	dstFile.Close()

	// Заменяем оригинальный файл временным
	if err := os.Rename(tmpFile, historyFile); err != nil {
		fmt.Printf("Error replacing file: %v\n", err)
		return
	}

	fmt.Printf("Entry %d deleted successfully\n", entryNum)

	// Показываем обновленную историю
	newEntries := currentEntry - 1
	fmt.Printf("History now has %d entries\n", newEntries)
}

func printHelp() {
	helpText := `Использование: history [КЛЮЧ]...
Отображает историю команд из файлов истории.

Ключи:
  -n ЧИСЛО       показать только последние ЧИСЛО команд
  -d НОМЕР       удалить запись истории с указанным номером
  -c             очистить файл истории (удалить все записи)
      --help     показать эту справку и выйти

Примеры:
  history           показать всю историю команд из файла
  history -n 10     показать последние 10 команд
  history -d 5      удалить 5-ю команду из файла истории
  history -c        очистить файл истории (требует подтверждения)
`
	fmt.Println(helpText)
}
