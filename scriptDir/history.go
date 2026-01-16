package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type HistoryEntry struct {
	ID      int
	Command string
	Time    time.Time
}

type HistoryConfig struct {
	Limit        int
	Offset       int
	ShowTimestamps bool
	ShowNumbers  bool
	Search       string
	Reverse      bool
	Clear        bool
	Delete       int
	Stats        bool
	Raw          bool
}

func main() {
	config := parseFlags()
	
	if config.Clear {
		clearHistory()
		return
	}
	
	if config.Delete > 0 {
		deleteHistoryEntry(config.Delete)
		return
	}
	
	entries := loadHistoryEntries()
	
	if config.Search != "" {
		entries = filterHistory(entries, config.Search)
	}
	
	if config.Stats {
		showHistoryStats(entries)
		return
	}
	
	displayHistory(entries, config)
}

func parseFlags() HistoryConfig {
	var config HistoryConfig
	
	limit := flag.Int("n", 0, "number of history entries to show")
	offset := flag.Int("o", 0, "offset from the end")
	showTimestamps := flag.Bool("t", false, "show timestamps")
	showNumbers := flag.Bool("num", true, "show line numbers")
	search := flag.String("s", "", "search for commands containing string")
	reverse := flag.Bool("r", false, "reverse order (newest first)")
	clear := flag.Bool("c", false, "clear the history file")
	deleteEntry := flag.Int("d", 0, "delete entry by number")
	stats := flag.Bool("stats", false, "show history statistics")
	raw := flag.Bool("raw", false, "raw output (no line numbers)")
	
	flag.Parse()
	
	config.Limit = *limit
	config.Offset = *offset
	config.ShowTimestamps = *showTimestamps
	config.ShowNumbers = *showNumbers
	config.Search = *search
	config.Reverse = *reverse
	config.Clear = *clear
	config.Delete = *deleteEntry
	config.Stats = *stats
	config.Raw = *raw
	
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
		filepath.Join(home, ".fish_history"),
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

func loadHistoryEntries() []HistoryEntry {
	var allEntries []HistoryEntry
	entryID := 1
	
	for _, historyFile := range getHistoryFiles() {
		entries := readHistoryFile(historyFile, entryID)
		allEntries = append(allEntries, entries...)
		entryID += len(entries)
	}
	
	return allEntries
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
	var lastTimestamp time.Time
	var inTimestampBlock bool
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			continue
		}
		
		// Обработка различных форматов истории
		
		// 1. Bash timestamp format: "#1612345678"
		if strings.HasPrefix(line, "#") && len(line) > 1 {
			// Проверяем, является ли это числом (таймстемпом)
			timestampStr := line[1:]
			if ts, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
				lastTimestamp = time.Unix(ts, 0)
				inTimestampBlock = true
				continue
			}
		}
		
		// 2. ZSH format: ": 1612345678:0;ls -la"
		if strings.HasPrefix(line, ": ") {
			entry := parseZshHistoryLine(line, currentID)
			if entry.Command != "" {
				entries = append(entries, entry)
				currentID++
			}
			continue
		}
		
		// 3. Обычная команда
		entry := HistoryEntry{
			ID:      currentID,
			Command: line,
		}
		
		// Если был таймстемп, добавляем его
		if inTimestampBlock && !lastTimestamp.IsZero() {
			entry.Time = lastTimestamp
			inTimestampBlock = false
		}
		
		// Проверяем, не является ли строка продолжением предыдущей команды
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ":") {
			// Это служебная строка, пропускаем
			continue
		}
		
		entries = append(entries, entry)
		currentID++
	}
	
	return entries
}

func parseZshHistoryLine(line string, id int) HistoryEntry {
	// ZSH format: ": 1612345678:0;ls -la"
	entry := HistoryEntry{ID: id}
	
	if !strings.HasPrefix(line, ": ") {
		entry.Command = line
		return entry
	}
	
	line = strings.TrimPrefix(line, ": ")
	
	// Ищем точку с запятой, разделяющую timestamp и команду
	semicolonIdx := strings.Index(line, ";")
	if semicolonIdx == -1 {
		entry.Command = line
		return entry
	}
	
	// Извлекаем timestamp
	timestampPart := line[:semicolonIdx]
	colonIdx := strings.Index(timestampPart, ":")
	if colonIdx != -1 {
		timestampStr := timestampPart[:colonIdx]
		if ts, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
			entry.Time = time.Unix(ts, 0)
		}
	}
	
	// Извлекаем команду
	entry.Command = line[semicolonIdx+1:]
	
	return entry
}

func filterHistory(entries []HistoryEntry, search string) []HistoryEntry {
	var filtered []HistoryEntry
	search = strings.ToLower(search)
	
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Command), search) {
			filtered = append(filtered, entry)
		}
	}
	
	return filtered
}

func applyHistoryFilters(entries []HistoryEntry, config HistoryConfig) []HistoryEntry {
	result := entries
	
	// Применяем offset
	if config.Offset > 0 {
		start := len(result) - config.Offset
		if start < 0 {
			start = 0
		}
		result = result[start:]
	}
	
	// Применяем limit
	if config.Limit > 0 && config.Limit < len(result) {
		if config.Reverse {
			result = result[:config.Limit]
		} else {
			result = result[len(result)-config.Limit:]
		}
	}
	
	// Применяем reverse
	if config.Reverse {
		reversed := make([]HistoryEntry, len(result))
		for i, j := 0, len(result)-1; i < len(result); i, j = i+1, j-1 {
			reversed[i] = result[j]
		}
		result = reversed
	}
	
	return result
}

func displayHistory(entries []HistoryEntry, config HistoryConfig) {
	if len(entries) == 0 {
		fmt.Println("No history entries found")
		fmt.Println("\nTo populate history:")
		fmt.Println("1. Run commands in your shell")
		fmt.Println("2. For bash: run 'history -w' to write to file")
		fmt.Println("3. Check if history files exist: ls ~/.bash_history")
		return
	}
	
	filtered := applyHistoryFilters(entries, config)
	
	fmt.Println("Command History:")
	fmt.Println("================")
	
	for _, entry := range filtered {
		var output strings.Builder
		
		if config.ShowNumbers && !config.Raw {
			output.WriteString(fmt.Sprintf("%5d  ", entry.ID))
		}
		
		if config.ShowTimestamps && !entry.Time.IsZero() {
			output.WriteString(fmt.Sprintf("%s  ", entry.Time.Format("2006-01-02 15:04:05")))
		}
		
		output.WriteString(entry.Command)
		
		fmt.Println(output.String())
	}
	
	fmt.Printf("\nTotal: %d commands", len(entries))
	if len(entries) != len(filtered) {
		fmt.Printf(" (showing %d)", len(filtered))
	}
	fmt.Println()
}

func showHistoryStats(entries []HistoryEntry) {
	if len(entries) == 0 {
		fmt.Println("No history entries to analyze")
		return
	}
	
	fmt.Println("History Statistics:")
	fmt.Println("===================")
	fmt.Printf("Total commands: %d\n", len(entries))
	
	// Команды с временными метками
	var timeEntries []HistoryEntry
	for _, entry := range entries {
		if !entry.Time.IsZero() {
			timeEntries = append(timeEntries, entry)
		}
	}
	
	if len(timeEntries) > 0 {
		// Временной диапазон
		firstTime := timeEntries[0].Time
		lastTime := timeEntries[len(timeEntries)-1].Time
		duration := lastTime.Sub(firstTime)
		
		fmt.Printf("\nTime range: %s to %s\n", 
			firstTime.Format("2006-01-02 15:04:05"),
			lastTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Duration: %v\n", duration)
		
		if duration.Hours() > 0 {
			commandsPerDay := float64(len(timeEntries)) / (duration.Hours() / 24)
			fmt.Printf("Average commands per day: %.1f\n", commandsPerDay)
		}
	}
	
	// Анализ по командам
	commandCount := make(map[string]int)
	for _, entry := range entries {
		// Берем первую часть команды (программу)
		parts := strings.Fields(entry.Command)
		if len(parts) > 0 {
			cmd := parts[0]
			commandCount[cmd]++
		}
	}
	
	// Сортировка по частоте использования
	type CommandFreq struct {
		Command string
		Count   int
	}
	
	var freqList []CommandFreq
	for cmd, count := range commandCount {
		freqList = append(freqList, CommandFreq{cmd, count})
	}
	
	// Сортировка по убыванию
	for i := 0; i < len(freqList)-1; i++ {
		for j := i + 1; j < len(freqList); j++ {
			if freqList[i].Count < freqList[j].Count {
				freqList[i], freqList[j] = freqList[j], freqList[i]
			}
		}
	}
	
	// Вывод топ-10 команд
	fmt.Println("\nMost frequent commands:")
	limit := 10
	if len(freqList) < limit {
		limit = len(freqList)
	}
	
	for i := 0; i < limit; i++ {
		percentage := float64(freqList[i].Count) * 100 / float64(len(entries))
		fmt.Printf("  %-20s: %d times (%.1f%%)\n", 
			freqList[i].Command, 
			freqList[i].Count, 
			percentage)
	}
	
	// Длина команд
	var totalChars int
	longestCmd := ""
	shortestCmd := strings.Repeat("x", 1000) // начальное большое значение
	
	for _, entry := range entries {
		length := len(entry.Command)
		totalChars += length
		
		if length > len(longestCmd) {
			longestCmd = entry.Command
		}
		if length < len(shortestCmd) && length > 0 {
			shortestCmd = entry.Command
		}
	}
	
	fmt.Printf("\nAverage command length: %.1f characters\n", 
		float64(totalChars)/float64(len(entries)))
	
	if len(longestCmd) > 0 {
		fmt.Printf("Longest command (%d chars): %s\n", 
			len(longestCmd), truncateString(longestCmd, 60))
	}
	if len(shortestCmd) < 1000 && len(shortestCmd) > 0 {
		fmt.Printf("Shortest command (%d chars): %s\n", 
			len(shortestCmd), shortestCmd)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func clearHistory() {
	files := getHistoryFiles()
	
	if len(files) == 0 {
		fmt.Println("No history files found to clear")
		fmt.Println("Typical history file locations:")
		fmt.Println("  ~/.bash_history")
		fmt.Println("  ~/.zsh_history")
		return
	}
	
	fmt.Println("Clearing history files:")
	for _, file := range files {
		// Создаем пустой файл
		if err := os.WriteFile(file, []byte{}, 0644); err != nil {
			fmt.Printf("  Error clearing %s: %v\n", file, err)
		} else {
			fmt.Printf("  Cleared: %s\n", file)
		}
	}
	
	fmt.Println("\nNote: This only clears history files.")
	fmt.Println("To clear current shell session history, run 'history -c' in your shell.")
}

func deleteHistoryEntry(entryNum int) {
	files := getHistoryFiles()
	
	if len(files) == 0 {
		fmt.Println("No history files found")
		return
	}
	
	// Используем первый найденный файл истории
	historyFile := files[0]
	
	// Читаем все записи
	entries := readHistoryFile(historyFile, 1)
	
	if entryNum <= 0 || entryNum > len(entries) {
		fmt.Printf("Entry %d not found (history has %d entries)\n", 
			entryNum, len(entries))
		return
	}
	
	// Показываем удаляемую команду
	entryToDelete := entries[entryNum-1]
	fmt.Printf("Deleting entry %d: %s\n", entryNum, entryToDelete.Command)
	
	// Удаляем запись
	var newLines []string
	file, err := os.Open(historyFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	currentEntry := 1
	var skipNext bool
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Пропускаем строки с таймстемпами связанные с удаляемой командой
		if skipNext {
			skipNext = false
			continue
		}
		
		// Проверяем таймстемп bash
		if strings.HasPrefix(line, "#") && len(line) > 1 {
			if _, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
				// Это таймстемп, проверяем связанную команду
				if currentEntry == entryNum {
					// Пропускаем таймстемп и следующую команду
					skipNext = true
					currentEntry++
					continue
				}
			}
		}
		
		// Проверяем zsh формат
		if strings.HasPrefix(line, ": ") && currentEntry == entryNum {
			// Пропускаем эту строку
			currentEntry++
			continue
		}
		
		// Обычная команда
		if !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ": ") && line != "" {
			if currentEntry == entryNum {
				// Пропускаем эту команду
				currentEntry++
				continue
			}
			currentEntry++
		}
		
		newLines = append(newLines, line)
	}
	
	// Записываем обновленный файл
	content := strings.Join(newLines, "\n")
	if err := os.WriteFile(historyFile, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}
	
	fmt.Printf("Entry %d deleted successfully\n", entryNum)
}

// Улучшенная версия с кэшированием
type HistoryCache struct {
	entries    []HistoryEntry
	loadedFrom string
	timestamp  time.Time
}

var historyCache *HistoryCache

func getCachedHistory() []HistoryEntry {
	if historyCache == nil {
		historyCache = &HistoryCache{}
	}
	
	// Получаем основной файл истории
	files := getHistoryFiles()
	if len(files) == 0 {
		return []HistoryEntry{}
	}
	
	mainFile := files[0]
	
	// Проверяем, нужно ли обновить кэш
	fileInfo, err := os.Stat(mainFile)
	if err != nil {
		return []HistoryEntry{}
	}
	
	if historyCache.loadedFrom == mainFile && 
	   fileInfo.ModTime().Before(historyCache.timestamp) {
		// Кэш актуален
		return historyCache.entries
	}
	
	// Обновляем кэш
	historyCache.entries = readHistoryFile(mainFile, 1)
	historyCache.loadedFrom = mainFile
	historyCache.timestamp = time.Now()
	
	return historyCache.entries
}

// Версия с поддержкой форматированного вывода
func printFormattedHistory(entries []HistoryEntry, config HistoryConfig) {
	if config.Raw {
		// Сырой вывод
		for _, entry := range entries {
			fmt.Println(entry.Command)
		}
		return
	}
	
	// Определяем максимальную ширину для ID
	maxIDWidth := len(strconv.Itoa(len(entries)))
	formatStr := fmt.Sprintf("%%%dd  ", maxIDWidth)
	
	for _, entry := range entries {
		// Номер команды
		if config.ShowNumbers {
			fmt.Printf(formatStr, entry.ID)
		}
		
		// Временная метка
		if config.ShowTimestamps && !entry.Time.IsZero() {
			fmt.Printf("[%s]  ", entry.Time.Format("2006-01-02 15:04:05"))
		}
		
		// Команда
		fmt.Println(entry.Command)
	}
}
