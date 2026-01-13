package main

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FileType представляет информацию о типе файла
type FileType struct {
	Description string
	MimeType    string
	Extensions  []string
}

var fileTypes = map[string]FileType{
	// Исполняемые файлы
	"\x7fELF":                 {"ELF executable", "application/x-executable", []string{}},
	"MZ":                     {"DOS executable", "application/x-dosexec", []string{".exe", ".com"}},
	"#!":                     {"script text executable", "text/x-script", []string{".sh", ".bash", ".py", ".pl"}},
	
	// Архивы
	"!<arch>":                {"current ar archive", "application/x-archive", []string{".a", ".lib"}},
	"PK\x03\x04":             {"Zip archive data", "application/zip", []string{".zip", ".jar", ".docx", ".xlsx", ".pptx"}},
	"\x1f\x8b":               {"gzip compressed data", "application/gzip", []string{".gz", ".tgz"}},
	"BZh":                    {"bzip2 compressed data", "application/x-bzip2", []string{".bz2", ".tbz2"}},
	"\xfd7zXZ\x00":           {"XZ compressed data", "application/x-xz", []string{".xz", ".txz"}},
	"7z\xbc\xaf\x27\x1c":     {"7-zip archive data", "application/x-7z-compressed", []string{".7z"}},
	"Rar!\x1a\x07\x00":       {"RAR archive data", "application/x-rar-compressed", []string{".rar"}},
	
	// Изображения
	"\xff\xd8\xff":           {"JPEG image data", "image/jpeg", []string{".jpg", ".jpeg", ".jpe", ".jfif"}},
	"\x89PNG\r\n\x1a\n":      {"PNG image data", "image/png", []string{".png"}},
	"GIF87a":                 {"GIF image data", "image/gif", []string{".gif"}},
	"GIF89a":                 {"GIF image data", "image/gif", []string{".gif"}},
	"BM":                     {"BMP image data", "image/bmp", []string{".bmp", ".dib"}},
	"RIFF....WEBP":           {"WebP image", "image/webp", []string{".webp"}},
	"\x00\x00\x01\x00":       {"ICO image data", "image/x-icon", []string{".ico"}},
	
	// Аудио
	"ID3":                    {"MP3 audio file", "audio/mpeg", []string{".mp3"}},
	"OggS":                   {"Ogg data", "audio/ogg", []string{".ogg", ".oga"}},
	"fLaC":                   {"FLAC audio bitstream data", "audio/flac", []string{".flac"}},
	"RIFF....WAVE":           {"WAVE audio", "audio/wav", []string{".wav"}},
	
	// Видео
	"\x00\x00\x00 ftyp":      {"MP4 video", "video/mp4", []string{".mp4", ".m4v", ".m4a"}},
	"RIFF....AVI ":           {"AVI video", "video/avi", []string{".avi"}},
	"\x1a\x45\xdf\xa3":       {"Matroska data", "video/x-matroska", []string{".mkv", ".mka"}},
	
	// Документы
	"%PDF-":                  {"PDF document", "application/pdf", []string{".pdf"}},
	"\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1": {"Microsoft Office document", "application/msword", []string{".doc", ".xls", ".ppt"}},
	"PK\x03\x04\x14\x00\x06\x00":       {"Microsoft Office 2007+ document", "application/vnd.openxmlformats-officedocument", []string{".docx", ".xlsx", ".pptx"}},
	
	// Текстовые форматы
	"\xef\xbb\xbf":           {"UTF-8 Unicode text", "text/plain; charset=utf-8", []string{}},
	"\xff\xfe":               {"Little-endian UTF-16 Unicode text", "text/plain; charset=utf-16le", []string{}},
	"\xfe\xff":               {"Big-endian UTF-16 Unicode text", "text/plain; charset=utf-16be", []string{}},
	
	// Базы данных
	"SQLite format 3\x00":    {"SQLite 3.x database", "application/x-sqlite3", []string{".db", ".sqlite", ".sqlite3"}},
	
	// Другие
	"<?xml ":                 {"XML document", "text/xml", []string{".xml"}},
	"<!DOCTYPE html":         {"HTML document", "text/html", []string{".html", ".htm"}},
	"{\"":                    {"JSON data", "application/json", []string{".json"}},
}

// printHelp выводит справку по использованию программы
func printHelp() {
	programName := os.Args[0]
	if len(os.Args[0]) > 0 {
		programName = filepath.Base(os.Args[0])
	}
	
	fmt.Printf(`Использование: %s [КЛЮЧ]... [ФАЙЛ]...
Определяет тип файлов.

Ключи:
  -b    краткий режим - не выводить имена файлов
  -i    выводить MIME-тип вместо описания
  -z    попытаться определить содержимое сжатых файлов

Поведение по умолчанию:
  - %s выводит тип каждого указанного файла.
  - Если файл является символической ссылкой, указывается куда она ведет.
  - Если файл является каталогом, указывается его тип.

Примеры:
  %s file.txt              # Определить тип file.txt
  %s -i image.jpg          # Вывести MIME-тип изображения
  %s -b *.go               # Краткий вывод для всех Go файлов
  %s -z archive.tar.gz     # Определить содержимое архива

Примечания:
  - Программа читает первые несколько байт файла для определения типа.
  - Для текстовых файлов определяется кодировка (UTF-8, UTF-16, ASCII).
  - Ключ -z работает для gzip, bzip2 и некоторых других архивов.
`, programName, programName, programName, programName, programName)
}

func main() {
	// Парсинг флагов
	brief := flag.Bool("b", false, "краткий режим - не выводить имена файлов")
	mime := flag.Bool("i", false, "выводить MIME-тип вместо описания")
	uncompress := flag.Bool("z", false, "попытаться определить содержимое сжатых файлов")
	help := flag.Bool("h", false, "показать справку и выйти")
	
	// Устанавливаем кастомное использование
	flag.Usage = func() {
		printHelp()
	}
	
	flag.Parse()
	
	// Проверяем флаг помощи
	if *help {
		printHelp()
		return
	}
	
	// Получаем список файлов
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"-"} // stdin по умолчанию
	}
	
	// Обрабатываем каждый файл
	exitCode := 0
	for _, filename := range args {
		result, err := analyzeFile(filename, *brief, *mime, *uncompress)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s: %v\n", os.Args[0], filename, err)
			exitCode = 1
		} else {
			fmt.Println(result)
		}
	}
	
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func analyzeFile(filename string, brief, mime, uncompress bool) (string, error) {
	// Специальный случай: stdin
	if filename == "-" {
		return analyzeStdin(brief, mime, uncompress)
	}
	
	// Получаем информацию о файле
	info, err := os.Stat(filename)
	if err != nil {
		return "", err
	}
	
	// Проверяем тип файла
	if info.IsDir() {
		return formatOutput(filename, "directory", "inode/directory", brief, mime), nil
	}
	
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(filename)
		if err != nil {
			return formatOutput(filename, "broken symbolic link", "inode/symlink", brief, mime), nil
		}
		desc := fmt.Sprintf("symbolic link to %s", target)
		return formatOutput(filename, desc, "inode/symlink", brief, mime), nil
	}
	
	// Открываем файл для чтения
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// Читаем данные для анализа
	data, err := readFileForAnalysis(file, uncompress)
	if err != nil {
		return "", err
	}
	
	if len(data) == 0 {
		return formatOutput(filename, "empty", "application/x-empty", brief, mime), nil
	}
	
	// Определяем тип файла
	desc, mimeType := detectFileType(data, filename, uncompress)
	
	return formatOutput(filename, desc, mimeType, brief, mime), nil
}

func readFileForAnalysis(file *os.File, uncompress bool) ([]byte, error) {
	// Читаем первые 1024 байта для анализа
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	data := buffer[:n]
	
	// Если запрошен анализ сжатых файлов и это архив, пытаемся распаковать
	if uncompress && isCompressedFile(data) {
		return tryUncompress(file, data)
	}
	
	return data, nil
}

func isCompressedFile(data []byte) bool {
	// Проверяем различные форматы сжатых файлов
	magicSignatures := []string{
		"\x1f\x8b",               // gzip
		"BZh",                    // bzip2
		"PK\x03\x04",             // ZIP
	}
	
	for _, magic := range magicSignatures {
		if bytes.HasPrefix(data, []byte(magic)) {
			return true
		}
	}
	
	return false
}

func tryUncompress(file *os.File, header []byte) ([]byte, error) {
	// Сбрасываем позицию чтения
	_, err := file.Seek(0, 0)
	if err != nil {
		return header, nil
	}
	
	// Пробуем определить тип архива и распаковать
	if bytes.HasPrefix(header, []byte("\x1f\x8b")) {
		// GZIP файл
		return uncompressGzip(file)
	} else if bytes.HasPrefix(header, []byte("BZh")) {
		// BZIP2 файл
		return uncompressBzip2(file)
	} else if bytes.HasPrefix(header, []byte("PK\x03\x04")) {
		// ZIP файл - просто возвращаем заголовок, так как ZIP сложнее распаковать
		// но можем попытаться прочитать первые файлы внутри
		return peekZipContent(file)
	}
	
	return header, nil
}

func uncompressGzip(file *os.File) ([]byte, error) {
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()
	
	// Читаем распакованные данные
	buffer := make([]byte, 1024)
	n, err := gzReader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	return buffer[:n], nil
}

func uncompressBzip2(file *os.File) ([]byte, error) {
	// bzip2.Reader читает из io.Reader
	bzReader := bzip2.NewReader(file)
	
	// Читаем распакованные данные
	buffer := make([]byte, 1024)
	n, err := bzReader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	return buffer[:n], nil
}

func peekZipContent(file *os.File) ([]byte, error) {
	// Для простоты просто читаем больше данных ZIP файла
	// В реальной реализации здесь была бы парсинг структуры ZIP
	buffer := make([]byte, 4096)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	return buffer[:n], nil
}

func analyzeStdin(brief, mime, uncompress bool) (string, error) {
	// Читаем данные из stdin
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения stdin: %v", err)
	}
	
	if len(data) == 0 {
		return formatOutput("(standard input)", "empty", "application/x-empty", brief, mime), nil
	}
	
	// Если запрошена распаковка и данные сжаты
	if uncompress && isCompressedFile(data) {
		// Создаем reader из данных
		reader := bytes.NewReader(data)
		
		if bytes.HasPrefix(data, []byte("\x1f\x8b")) {
			// GZIP
			gzReader, err := gzip.NewReader(reader)
			if err == nil {
				defer gzReader.Close()
				uncompressed, err := ioutil.ReadAll(gzReader)
				if err == nil && len(uncompressed) > 0 {
					data = uncompressed
				}
			}
		} else if bytes.HasPrefix(data, []byte("BZh")) {
			// BZIP2
			bzReader := bzip2.NewReader(reader)
			uncompressed, err := ioutil.ReadAll(bzReader)
			if err == nil && len(uncompressed) > 0 {
				data = uncompressed
			}
		}
	}
	
	desc, mimeType := detectFileType(data, "", uncompress)
	return formatOutput("(standard input)", desc, mimeType, brief, mime), nil
}

func detectFileType(data []byte, filename string, uncompress bool) (string, string) {
	if len(data) == 0 {
		return "empty", "application/x-empty"
	}
	
	// Если данные были распакованы с помощью -z, определяем тип распакованных данных
	if uncompress {
		// Сначала проверяем, не является ли это tar архивом после распаковки
		if bytes.HasPrefix(data, []byte("ustar")) || len(data) >= 257 && data[257] == 'u' && data[258] == 's' && data[259] == 't' && data[260] == 'a' && data[261] == 'r' {
			return "tar archive", "application/x-tar"
		}
	}
	
	// Проверяем magic numbers
	for magic, fileType := range fileTypes {
		if bytes.HasPrefix(data, []byte(magic)) {
			// Для сжатых файлов с ключом -z добавляем информацию о содержимом
			if uncompress && isCompressedFormat(magic) {
				contentType := guessContentType(data)
				if contentType != "" {
					return fmt.Sprintf("%s (%s)", fileType.Description, contentType), fileType.MimeType
				}
			}
			return fileType.Description, fileType.MimeType
		}
	}
	
	// Проверяем совпадения по шаблонам
	for pattern, fileType := range fileTypes {
		if len(pattern) > 4 && pattern[len(pattern)-4:] == "...." {
			// Паттерн типа "RIFF....WEBP"
			parts := strings.Split(pattern, "....")
			if len(parts) == 2 {
				if len(data) >= 12 && bytes.HasPrefix(data, []byte(parts[0])) {
					// Проверяем подстроку на позиции 8
					if len(data) >= len(parts[1])+8 {
						if string(data[8:8+len(parts[1])]) == parts[1] {
							return fileType.Description, fileType.MimeType
						}
					}
				}
			}
		}
	}
	
	// Проверяем шелл-скрипты
	if bytes.HasPrefix(data, []byte("#!")) {
		firstLine := string(data)
		if idx := bytes.IndexByte(data, '\n'); idx > 0 {
			firstLine = string(data[:idx])
		}
		interpreter := strings.TrimSpace(firstLine[2:])
		return fmt.Sprintf("%s script text executable", interpreter), "text/x-script"
	}
	
	// Проверяем XML
	if bytes.HasPrefix(data, []byte("<?xml ")) {
		return "XML document", "text/xml"
	}
	
	// Проверяем HTML
	if bytes.Contains(data, []byte("<!DOCTYPE html")) || 
	   (bytes.Contains(data, []byte("<html")) && bytes.Contains(data, []byte("</html>"))) {
		return "HTML document", "text/html"
	}
	
	// Проверяем JSON
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) ||
	   bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		if isLikelyJSON(data) {
			return "JSON data", "application/json"
		}
	}
	
	// Проверяем текстовый ли файл
	if isTextFile(data) {
		encoding := detectTextEncoding(data)
		ext := strings.ToLower(filepath.Ext(filename))
		
		// Определяем по расширению
		switch ext {
		case ".go":
			return fmt.Sprintf("%s Go source, text", encoding), "text/x-go"
		case ".c", ".h":
			return fmt.Sprintf("%s C source, text", encoding), "text/x-c"
		case ".py":
			return fmt.Sprintf("%s Python script, text", encoding), "text/x-python"
		case ".java":
			return fmt.Sprintf("%s Java source, text", encoding), "text/x-java"
		case ".js":
			return fmt.Sprintf("%s JavaScript source, text", encoding), "application/javascript"
		case ".html", ".htm":
			return fmt.Sprintf("%s HTML document, text", encoding), "text/html"
		case ".css":
			return fmt.Sprintf("%s CSS stylesheet, text", encoding), "text/css"
		case ".sh", ".bash":
			return fmt.Sprintf("%s shell script, text", encoding), "application/x-sh"
		case ".txt":
			return fmt.Sprintf("%s plain text", encoding), "text/plain"
		case ".csv":
			return fmt.Sprintf("%s CSV text", encoding), "text/csv"
		default:
			return fmt.Sprintf("%s text", encoding), "text/plain"
		}
	}
	
	// Бинарный файл без конкретного типа
	return "data", "application/octet-stream"
}

func isCompressedFormat(magic string) bool {
	compressedMagics := []string{
		"\x1f\x8b",     // gzip
		"BZh",          // bzip2
		"\xfd7zXZ\x00", // XZ
		"PK\x03\x04",   // ZIP
		"Rar!\x1a\x07\x00", // RAR
	}
	
	for _, m := range compressedMagics {
		if magic == m {
			return true
		}
	}
	return false
}

func guessContentType(data []byte) string {
	// Пытаемся угадать тип содержимого по первым байтам
	if len(data) == 0 {
		return ""
	}
	
	// Проверяем текстовое ли содержимое
	if isTextFile(data) {
		return "text"
	}
	
	// Проверяем различные форматы
	if bytes.HasPrefix(data, []byte("\x1f\x8b")) {
		return "gzip compressed data"
	} else if bytes.HasPrefix(data, []byte("BZh")) {
		return "bzip2 compressed data"
	} else if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		return "zip archive data"
	} else if bytes.HasPrefix(data, []byte("ustar")) {
		return "tar archive"
	}
	
	return "binary"
}

func isTextFile(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	
	// Проверяем первые 1024 байта
	checkLen := len(data)
	if checkLen > 1024 {
		checkLen = 1024
	}
	
	nullCount := 0
	printableCount := 0
	
	for i := 0; i < checkLen; i++ {
		b := data[i]
		if b == 0 {
			nullCount++
			// Если много нулевых байтов - это бинарный файл
			if nullCount > checkLen/100 {
				return false
			}
		}
		if (b >= 32 && b <= 126) || b == 9 || b == 10 || b == 13 {
			printableCount++
		}
	}
	
	// Если больше 85% символов печатные - это текст
	return float64(printableCount)/float64(checkLen) > 0.85
}

func detectTextEncoding(data []byte) string {
	// Проверяем BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return "UTF-8 Unicode"
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return "Big-endian UTF-16 Unicode"
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return "Little-endian UTF-16 Unicode"
	}
	
	// Проверяем UTF-8
	if utf8.Valid(data) {
		hasNonASCII := false
		for i := 0; i < len(data) && i < 100; i++ {
			if data[i] >= 0x80 {
				hasNonASCII = true
				break
			}
		}
		if hasNonASCII {
			return "UTF-8 Unicode"
		}
		return "ASCII"
	}
	
	return "unknown-8bit"
}

func isLikelyJSON(data []byte) bool {
	data = bytes.TrimSpace(data)
	
	if len(data) == 0 {
		return false
	}
	
	firstChar := data[0]
	lastChar := data[len(data)-1]
	
	// Должен начинаться с { или [ и заканчиваться } или ]
	if (firstChar == '{' && lastChar == '}') || 
	   (firstChar == '[' && lastChar == ']') {
		// Простая проверка баланса скобок
		braceCount := 0
		bracketCount := 0
		
		for _, b := range data {
			switch b {
			case '{':
				braceCount++
			case '}':
				braceCount--
			case '[':
				bracketCount++
			case ']':
				bracketCount--
			}
		}
		
		return braceCount == 0 && bracketCount == 0
	}
	
	return false
}

func formatOutput(filename, desc, mime string, brief, mimeFlag bool) string {
	output := desc
	if mimeFlag {
		output = mime
	}
	
	if brief {
		return output
	}
	
	return fmt.Sprintf("%s: %s", filename, output)
}