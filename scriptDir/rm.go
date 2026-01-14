package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	force := flag.Bool("f", false, "force removal")
	interactive := flag.Bool("i", false, "interactive removal")
	recursive := flag.Bool("r", false, "remove directories recursively")
	verbose := flag.Bool("v", false, "verbose output")
	dir := flag.Bool("d", false, "remove empty directories")
	help := flag.Bool("h", false, "show help")
	
	flag.Parse()

	if *help {
		fmt.Println(`rm - remove files or directories
Usage: rm [options] <file>...
Options:
  -f  force, ignore errors
  -i  interactive, ask before each removal
  -r  recursive, remove directories and contents
  -d  remove empty directories only
  -v  verbose, show what is being removed
  -h  show this help`)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "rm: missing operand")
		os.Exit(1)
	}

	exitCode := 0
	for _, path := range args {
		if err := rm(path, *force, *interactive, *recursive, *verbose, *dir); err != nil {
			fmt.Fprintf(os.Stderr, "rm: %s: %v\n", path, err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func rm(path string, force, interactive, recursive, verbose, dir bool) error {
	// Проверяем существование
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) && force {
			return nil // Игнорируем с -f
		}
		return fmt.Errorf("no such file or directory")
	}

	// Определяем тип
	isDir := info.IsDir()
	isEmpty := false
	if isDir {
		isEmpty = isDirEmpty(path)
	}

	// Определяем способ удаления
	if isDir {
		if recursive {
			// -r имеет приоритет над -d
			return rmRecursive(path, force, interactive, verbose)
		} else if dir {
			// Только -d (без -r)
			if !isEmpty {
				return fmt.Errorf("directory not empty")
			}
			return rmSingle(path, force, interactive, verbose, "empty directory")
		} else {
			// Ни -r, ни -d
			return fmt.Errorf("is a directory")
		}
	} else {
		// Обычный файл
		return rmSingle(path, force, interactive, verbose, "file")
	}
}

func rmSingle(path string, force, interactive, verbose bool, itemType string) error {
	// Интерактивное подтверждение
	if interactive && !force {
		if !confirm(fmt.Sprintf("remove %s '%s'?", itemType, path)) {
			if verbose {
				fmt.Printf("skipped '%s'\n", path)
			}
			return nil
		}
	}

	// Удаление
	if verbose {
		fmt.Printf("removing '%s'\n", path)
	}
	return os.Remove(path)
}

func rmRecursive(path string, force, interactive, verbose bool) error {
	// Сначала собираем все пути
	var allPaths []string
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			if force {
				return nil
			}
			return err
		}
		allPaths = append(allPaths, currentPath)
		return nil
	})
	
	if err != nil {
		return err
	}

	// Удаляем в обратном порядке
	for i := len(allPaths) - 1; i >= 0; i-- {
		currentPath := allPaths[i]
		
		// Проверяем, существует ли еще
		if _, err := os.Stat(currentPath); err != nil {
			if os.IsNotExist(err) && force {
				continue
			}
			if !force {
				return err
			}
		}

		// Определяем тип
		info, err := os.Lstat(currentPath)
		if err != nil {
			if force {
				continue
			}
			return err
		}

		itemType := "file"
		if info.IsDir() {
			itemType = "directory"
		}

		// Интерактивное подтверждение
		if interactive && !force {
			relPath, _ := filepath.Rel(path, currentPath)
			if relPath == "." {
				relPath = filepath.Base(path)
			}
			if !confirm(fmt.Sprintf("remove %s '%s'?", itemType, relPath)) {
				continue
			}
		}

		// Удаление
		if verbose {
			fmt.Printf("removing %s '%s'\n", itemType, currentPath)
		}
		
		if err := os.Remove(currentPath); err != nil && !force {
			return err
		}
	}

	return nil
}

func isDirEmpty(path string) bool {
	dir, err := os.Open(path)
	if err != nil {
		return false
	}
	defer dir.Close()

	_, err = dir.Readdir(1)
	return err != nil
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes" || response == "д" || response == "да"
}