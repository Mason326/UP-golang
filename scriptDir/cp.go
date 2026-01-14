package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Базовые флаги
	recursive := flag.Bool("r", false, "copy directories recursively")
	force := flag.Bool("f", false, "force copy, overwrite existing files")
	interactive := flag.Bool("i", false, "ask before overwrite")
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("help", false, "show help")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... SOURCE DEST\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... SOURCE... DIRECTORY\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Copy SOURCE to DEST, or multiple SOURCE(s) to DIRECTORY.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s file1.txt file2.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -r dir1 dir2\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -v file.txt backup/\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	sources := args[:len(args)-1]
	dest := args[len(args)-1]

	// Проверяем dest
	destInfo, err := os.Stat(dest)
	destIsDir := err == nil && destInfo.IsDir()

	// Если несколько источников, dest должен быть директорией
	if len(sources) > 1 && !destIsDir {
		fmt.Fprintf(os.Stderr, "%s: target '%s' is not a directory\n", os.Args[0], dest)
		os.Exit(1)
	}

	// Выполняем копирование
	success := true
	for _, src := range sources {
		srcInfo, err := os.Stat(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot stat '%s': %v\n", os.Args[0], src, err)
			success = false
			continue
		}

		var target string
		if destIsDir {
			target = filepath.Join(dest, filepath.Base(src))
		} else {
			target = dest
		}

		if srcInfo.IsDir() {
			if !*recursive {
				fmt.Fprintf(os.Stderr, "%s: omitting directory '%s' (use -r to copy)\n", os.Args[0], src)
				success = false
				continue
			}
			if !copyDir(src, target, *force, *interactive, *verbose) {
				success = false
			}
		} else {
			if !copyFile(src, target, *force, *interactive, *verbose) {
				success = false
			}
		}
	}

	if !success {
		os.Exit(1)
	}
}

func copyFile(src, dst string, force, interactive, verbose bool) bool {
	// Проверяем, существует ли файл назначения
	if _, err := os.Stat(dst); err == nil {
		// Файл существует
		if interactive {
			fmt.Printf("%s: overwrite '%s'? (y/n) ", os.Args[0], dst)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				if verbose {
					fmt.Printf("skipped '%s'\n", dst)
				}
				return true
			}
		} else if !force {
			fmt.Fprintf(os.Stderr, "%s: cannot create regular file '%s': File exists\n", os.Args[0], dst)
			return false
		}
	}

	// Открываем исходный файл
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot open '%s': %v\n", os.Args[0], src, err)
		return false
	}
	defer srcFile.Close()

	// Получаем информацию о файле
	srcInfo, err := srcFile.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot stat '%s': %v\n", os.Args[0], src, err)
		return false
	}

	// Создаем целевой файл
	dstFile, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot create '%s': %v\n", os.Args[0], dst, err)
		return false
	}
	defer dstFile.Close()

	// Копируем содержимое
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: error copying '%s' to '%s': %v\n", os.Args[0], src, dst, err)
		// Удаляем частично скопированный файл
		os.Remove(dst)
		return false
	}

	// Устанавливаем правильные права доступа
	err = os.Chmod(dst, srcInfo.Mode())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: warning: failed to preserve permissions for '%s': %v\n", os.Args[0], dst, err)
	}

	if verbose {
		fmt.Printf("'%s' -> '%s'\n", src, dst)
	}

	return true
}

func copyDir(src, dst string, force, interactive, verbose bool) bool {
	// Создаем целевую директорию
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot create directory '%s': %v\n", os.Args[0], dst, err)
		return false
	}

	// Открываем исходную директорию
	dir, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot open directory '%s': %v\n", os.Args[0], src, err)
		return false
	}
	defer dir.Close()

	// Получаем информацию о директории
	srcInfo, err := dir.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot stat directory '%s': %v\n", os.Args[0], src, err)
		return false
	}

	// Устанавливаем права доступа для директории
	err = os.Chmod(dst, srcInfo.Mode())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: warning: failed to preserve permissions for directory '%s': %v\n", os.Args[0], dst, err)
	}

	// Читаем содержимое директории
	entries, err := dir.Readdir(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: error reading directory '%s': %v\n", os.Args[0], src, err)
		return false
	}

	success := true
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if !copyDir(srcPath, dstPath, force, interactive, verbose) {
				success = false
			}
		} else {
			if !copyFile(srcPath, dstPath, force, interactive, verbose) {
				success = false
			}
		}
	}

	return success
}