package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Парсинг флагов
	universal := flag.Bool("u", false, "показывать или устанавливать время в формате UTC")
	reference := flag.String("r", "", "отображать время последней модификации файла")
	format := flag.String("f", "", "использовать указанный формат для вывода")
	dateStr := flag.String("d", "", "отобразить время, описанное в строке")
	help := flag.Bool("h", false, "показать справку")
	
	flag.Usage = func() {
		printHelp()
	}
	
	flag.Parse()
	
	// Проверяем флаг помощи
	if *help {
		printHelp()
		return
	}
	
	// Определяем время для отображения
	var displayTime time.Time
	var err error
	
	if *reference != "" {
		// Используем время файла
		displayTime, err = getFileTime(*reference)
		if err != nil {
			fmt.Fprintf(os.Stderr, "date: %s: %v\n", *reference, err)
			os.Exit(1)
		}
	} else if *dateStr != "" {
		// Используем время из строки
		displayTime, err = parseDateString(*dateStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "date: неверный формат даты '%s': %v\n", *dateStr, err)
			os.Exit(1)
		}
	} else {
		// Показываем текущее время
		if *universal {
			displayTime = time.Now().UTC()
		} else {
			displayTime = time.Now()
		}
	}
	
	// Форматируем и выводим время
	output := formatTime(displayTime, *format, *universal)
	fmt.Println(output)
}

func printHelp() {
	fmt.Println(`Использование: date [опции] [+ФОРМАТ]
Показывает системную дату и время.

Опции:
  -d, --date=СТРОКА     отобразить время, описанное в СТРОКЕ
  -f, --format=ФОРМАТ   использовать указанный формат для вывода
  -r, --reference=ФАЙЛ  отобразить время последней модификации ФАЙЛА
  -u, --utc             использовать UTC время
  -h, --help            показать эту справку

Форматы:
  %%     символ процента
  %%a    сокращенное название дня недели (Пн, Вт...)
  %%A    полное название дня недели (Понедельник...)
  %%b    сокращенное название месяца (Янв, Фев...)
  %%B    полное название месяца (Январь...)
  %%c    полная дата и время (по умолчанию)
  %%C    век (год/100)
  %%d    день месяца (01..31)
  %%D    дата в формате (мм/дд/гг)
  %%e    день месяца ( 1..31)
  %%F    полная дата (гггг-мм-дд)
  %%H    час (00..23)
  %%I    час (01..12)
  %%j    день года (001..366)
  %%k    час ( 0..23)
  %%l    час ( 1..12)
  %%m    месяц (01..12)
  %%M    минуты (00..59)
  %%n    перевод строки
  %%p    AM/PM
  %%P    am/pm
  %%r    время в 12-часовом формате (чч:мм:сс AM/PM)
  %%R    время в 24-часовом формате (чч:мм)
  %%s    количество секунд с 1970-01-01 00:00:00 UTC
  %%S    секунды (00..60)
  %%t    табуляция
  %%T    время в 24-часовом формате (чч:мм:сс)
  %%u    день недели (1..7, 1 - понедельник)
  %%U    номер недели года (00..53, воскресенье первый день недели)
  %%V    номер недели года по ISO (01..53)
  %%w    день недели (0..6, 0 - воскресенье)
  %%W    номер недели года (00..53, понедельник первый день недели)
  %%x    дата (дд/мм/гг)
  %%X    время (чч:мм:сс)
  %%y    последние две цифры года (00..99)
  %%Y    год
  %%z    смещение часового пояса (+hhmm)
  %%Z    название часового пояса

Примеры:
  date                  # Текущая дата и время
  date -u              # Текущее время в UTC
  date -r file.txt     # Время модификации файла
  date -d "2023-12-25" # Показать указанную дату`)
}

func getFileTime(filename string) (time.Time, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func parseDateString(dateStr string) (time.Time, error) {
	// Пробуем разные форматы
	
	// Unix timestamp
	if strings.HasPrefix(dateStr, "@") {
		timestamp, err := strconv.ParseInt(dateStr[1:], 10, 64)
		if err == nil {
			return time.Unix(timestamp, 0), nil
		}
	}
	
	// Относительные времена (упрощенная реализация)
	lowerStr := strings.ToLower(dateStr)
	
	switch lowerStr {
	case "tomorrow":
		return time.Now().AddDate(0, 0, 1), nil
	case "yesterday":
		return time.Now().AddDate(0, 0, -1), nil
	case "next week":
		return time.Now().AddDate(0, 0, 7), nil
	case "last week":
		return time.Now().AddDate(0, 0, -7), nil
	case "next month":
		return time.Now().AddDate(0, 1, 0), nil
	case "last month":
		return time.Now().AddDate(0, -1, 0), nil
	case "next year":
		return time.Now().AddDate(1, 0, 0), nil
	case "last year":
		return time.Now().AddDate(-1, 0, 0), nil
	}
	
	// Проверяем "ago"
	if strings.Contains(lowerStr, "ago") {
		// Простая реализация для "2 hours ago" и т.д.
		parts := strings.Fields(lowerStr)
		if len(parts) >= 3 {
			amount, err := strconv.Atoi(parts[0])
			if err == nil {
				unit := parts[1]
				switch unit {
				case "second", "seconds":
					return time.Now().Add(-time.Duration(amount) * time.Second), nil
				case "minute", "minutes":
					return time.Now().Add(-time.Duration(amount) * time.Minute), nil
				case "hour", "hours":
					return time.Now().Add(-time.Duration(amount) * time.Hour), nil
				case "day", "days":
					return time.Now().AddDate(0, 0, -amount), nil
				case "week", "weeks":
					return time.Now().AddDate(0, 0, -amount*7), nil
				case "month", "months":
					return time.Now().AddDate(0, -amount, 0), nil
				case "year", "years":
					return time.Now().AddDate(-amount, 0, 0), nil
				}
			}
		}
	}
	
	// Стандартные форматы
	formats := []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02 Jan 2006 15:04:05 MST",
		"02 Jan 2006 15:04:05",
		"02 Jan 2006 15:04",
		"02 Jan 2006",
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123,
		time.RFC1123Z,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
	}
	
	for _, fmt := range formats {
		t, err := time.ParseInLocation(fmt, dateStr, time.Local)
		if err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("неверный формат даты")
}

func formatTime(t time.Time, format string, utc bool) string {
	if utc {
		t = t.UTC()
	}
	
	// Если формат пустой или начинается с +, используем его как формат
	if format != "" {
		// Убираем начальный + если есть
		if strings.HasPrefix(format, "+") {
			format = format[1:]
		}
		return formatCustom(t, format)
	}
	
	// Формат по умолчанию
	return t.Format("Mon Jan 2 15:04:05 MST 2006")
}

func formatCustom(t time.Time, format string) string {
	// Заменяем спецификаторы формата
	result := ""
	i := 0
	for i < len(format) {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case '%':
				result += "%"
			case 'a':
				result += t.Format("Mon")
			case 'A':
				days := map[time.Weekday]string{
					time.Sunday:    "Воскресенье",
					time.Monday:    "Понедельник",
					time.Tuesday:   "Вторник",
					time.Wednesday: "Среда",
					time.Thursday:  "Четверг",
					time.Friday:    "Пятница",
					time.Saturday:  "Суббота",
				}
				result += days[t.Weekday()]
			case 'b':
				result += t.Format("Jan")
			case 'B':
				months := map[time.Month]string{
					time.January:   "Январь",
					time.February:  "Февраль",
					time.March:     "Март",
					time.April:     "Апрель",
					time.May:       "Май",
					time.June:      "Июнь",
					time.July:      "Июль",
					time.August:    "Август",
					time.September: "Сентябрь",
					time.October:   "Октябрь",
					time.November:  "Ноябрь",
					time.December:  "Декабрь",
				}
				result += months[t.Month()]
			case 'c':
				result += t.Format("Mon Jan 2 15:04:05 2006")
			case 'C':
				result += fmt.Sprintf("%02d", t.Year()/100)
			case 'd':
				result += fmt.Sprintf("%02d", t.Day())
			case 'D':
				result += t.Format("01/02/06")
			case 'e':
				result += fmt.Sprintf("%2d", t.Day())
			case 'F':
				result += t.Format("2006-01-02")
			case 'H':
				result += fmt.Sprintf("%02d", t.Hour())
			case 'I':
				hour := t.Hour() % 12
				if hour == 0 {
					hour = 12
				}
				result += fmt.Sprintf("%02d", hour)
			case 'j':
				result += fmt.Sprintf("%03d", t.YearDay())
			case 'k':
				result += fmt.Sprintf("%2d", t.Hour())
			case 'l':
				hour := t.Hour() % 12
				if hour == 0 {
					hour = 12
				}
				result += fmt.Sprintf("%2d", hour)
			case 'm':
				result += fmt.Sprintf("%02d", int(t.Month()))
			case 'M':
				result += fmt.Sprintf("%02d", t.Minute())
			case 'n':
				result += "\n"
			case 'p':
				if t.Hour() < 12 {
					result += "AM"
				} else {
					result += "PM"
				}
			case 'P':
				if t.Hour() < 12 {
					result += "am"
				} else {
					result += "pm"
				}
			case 'r':
				result += t.Format("03:04:05 PM")
			case 'R':
				result += t.Format("15:04")
			case 's':
				result += fmt.Sprintf("%d", t.Unix())
			case 'S':
				result += fmt.Sprintf("%02d", t.Second())
			case 't':
				result += "\t"
			case 'T':
				result += t.Format("15:04:05")
			case 'u':
				weekday := int(t.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				result += fmt.Sprintf("%d", weekday)
			case 'w':
				result += fmt.Sprintf("%d", t.Weekday())
			case 'U':
				result += fmt.Sprintf("%02d", getWeekNumber(t, time.Sunday))
			case 'V':
				_, week := t.ISOWeek()
				result += fmt.Sprintf("%02d", week)
			case 'W':
				_, week := t.ISOWeek()
				result += fmt.Sprintf("%02d", week)
			case 'x':
				result += t.Format("01/02/06")
			case 'X':
				result += t.Format("15:04:05")
			case 'y':
				result += fmt.Sprintf("%02d", t.Year()%100)
			case 'Y':
				result += fmt.Sprintf("%04d", t.Year())
			case 'z':
				_, offset := t.Zone()
				hours := offset / 3600
				minutes := (offset % 3600) / 60
				sign := "+"
				if hours < 0 {
					sign = "-"
					hours = -hours
					minutes = -minutes
				}
				result += fmt.Sprintf("%s%02d%02d", sign, hours, minutes)
			case 'Z':
				name, _ := t.Zone()
				result += name
			default:
				result += string(format[i:i+2])
			}
			i += 2
		} else {
			result += string(format[i])
			i++
		}
	}
	
	return result
}

func getWeekNumber(t time.Time, firstDay time.Weekday) int {
	yearStart := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	
	for yearStart.Weekday() != firstDay {
		yearStart = yearStart.AddDate(0, 0, 1)
	}
	
	if t.Before(yearStart) {
		return 0
	}
	
	days := int(t.Sub(yearStart).Hours() / 24)
	return days/7 + 1
}
