package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/tealeg/xlsx/v3"
)

type Component struct {
	targetColumns map[string]string
	sheetName     string
	url           string
}

func main() {

	filename := "Library.xlsx"

	component := Component{
		targetColumns: map[string]string{
		"Color":   "Цвет свечения диода",
		"Voltage": "Прямое напряжение (В) при токе 20 мА",
		},
		sheetName: "LedsParsed",
		url: "https://www.smd.ru/katalog/poluprovodnikovye_diody_SMD/smd_LED_svetodiody/LED_0603_1204_1206/",
	}

	// Получаем HTML страницу
	resp, err := http.Get(component.url)
	if err != nil {
		log.Fatal("Ошибка при получении страницы:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatal("Статус код не 200:", resp.StatusCode)
	}

	// Парсим HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal("Ошибка при парсинге HTML:", err)
	}

	// Находим все таблицы с классом "goodsByArticul"
	tables := doc.Find("table.goodsByArticul")
	fmt.Printf("Найдено таблиц с классом 'goodsByArticul': %d\n\n", tables.Length())

	// Выводим все заголовки таблиц
	printAllTableHeaders(tables, component.targetColumns)

	// Собираем данные из всех таблиц
	allData := collectDataFromTables(tables, component.targetColumns)

	// Записываем данные в XLSX файл
	err = writeToXLSX(allData, filename, component.sheetName)
	if err != nil {
		log.Fatal("Ошибка при записи в XLSX:", err)
	}

	fmt.Printf("✅ Данные успешно записаны в %s на страницу %s", filename, component.sheetName)
}

// Функция для сбора данных из всех таблиц
func collectDataFromTables(tables *goquery.Selection, targetColumns map[string]string) []map[string]string {
	var allData []map[string]string

	tables.Each(func(tableIndex int, table *goquery.Selection) {
		fmt.Printf("═════ Обработка таблицы №%d ═════\n", tableIndex+1)

		// Получаем заголовки таблицы
		headers := getTableHeaders(table)

		// Создаем map для соответствия XLSX заголовков и индексов столбцов
		columnIndexes := make(map[string]int)

		// Находим индексы нужных столбцов
		for xlsxHeader, searchPattern := range targetColumns {
			normalizedSearch := normalizeString(searchPattern)

			for i, header := range headers {
				normalizedHeader := normalizeString(header)
				if strings.Contains(normalizedHeader, normalizedSearch) {
					columnIndexes[xlsxHeader] = i
					fmt.Printf("✅ Найден столбец: '%s' -> '%s' (индекс: %d)\n",
						searchPattern, xlsxHeader, i)
					break
				}
			}
		}

		// Получаем данные из таблицы
		rows := table.Find("tr").Slice(1, goquery.ToEnd)
		rows.Each(func(rowIndex int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() == 0 {
				return
			}

			// Создаем map для данных строки
			rowData := make(map[string]string)

			// Заполняем данные для каждого XLSX столбца
			for xlsxHeader, columnIndex := range columnIndexes {
				if columnIndex < cells.Length() {
					value := strings.TrimSpace(cells.Eq(columnIndex).Text())
					rowData[xlsxHeader] = value
				}
			}

			// Добавляем данные строки в общий массив
			if len(rowData) > 0 {
				allData = append(allData, rowData)
				fmt.Printf("📝 Собрана строка %d: %v\n", len(allData), rowData)
			}
		})
	})

	return allData
}

// Функция для записи данных в XLSX файл
func writeToXLSX(data []map[string]string, filename, sheetName string) error {
	var file *xlsx.File
	var sheet *xlsx.Sheet

	// Проверяем существует ли файл
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Создаем новый файл
		file = xlsx.NewFile()
		sheet, err = file.AddSheet(sheetName)
		if err != nil {
			return err
		}
	} else {
		// Открываем существующий файл
		file, err = xlsx.OpenFile(filename)
		if err != nil {
			return err
		}

		// Удаляем существующий лист, если есть
		if _, exists := file.Sheet[sheetName]; exists {
			delete(file.Sheet, sheetName)
		}

		// Создаем новый лист
		sheet, err = file.AddSheet(sheetName)
		if err != nil {
			return err
		}
	}

	// Если нет данных для записи
	if len(data) == 0 {
		return fmt.Errorf("нет данных для записи")
	}

	// Создаем заголовки (первая строка)
	headerRow := sheet.AddRow()
	headers := getXLSXHeaders(data[0])
	for _, header := range headers {
		cell := headerRow.AddCell()
		cell.Value = header
		cell.SetStyle(getHeaderStyle())
	}

	// Записываем данные
	for _, rowData := range data {
		row := sheet.AddRow()
		for _, header := range headers {
			cell := row.AddCell()
			cell.Value = rowData[header]
		}
	}

	// Сохраняем файл
	return file.Save(filename)
}

// Функция для получения заголовков XLSX из данных
func getXLSXHeaders(data map[string]string) []string {
	headers := make([]string, 0, len(data))
	for header := range data {
		headers = append(headers, header)
	}
	return headers
}

// Функция для стиля заголовков
func getHeaderStyle() *xlsx.Style {
	style := xlsx.NewStyle()
	style.Font.Bold = true
	style.Fill.FgColor = "00FFFF00" // Желтый фон
	style.Fill.PatternType = "solid"
	return style
}

// Функция для вывода всех заголовков всех таблиц
func printAllTableHeaders(tables *goquery.Selection, targetColumns map[string]string) {
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("📋 ВСЕ ЗАГОЛОВКИ ВСЕХ ТАБЛИЦ:")
	fmt.Println("═══════════════════════════════════════════════════")

	// Создаем массив поисковых паттернов
	var searchPatterns []string
	for _, pattern := range targetColumns {
		searchPatterns = append(searchPatterns, pattern)
	}

	tables.Each(func(tableIndex int, table *goquery.Selection) {
		headers := getTableHeaders(table)

		fmt.Printf("\n📊 Таблица №%d - Заголовки (%d):\n", tableIndex+1, len(headers))
		fmt.Println("─" + strings.Repeat("─", 50))

		for i, header := range headers {
			normalizedHeader := normalizeString(header)
			matchInfo := ""

			// Проверяем совпадение с каждым searchPattern
			for j, pattern := range searchPatterns {
				normalizedPattern := normalizeString(pattern)
				if strings.Contains(normalizedHeader, normalizedPattern) {
					matchInfo = fmt.Sprintf(" ✅ (совпадение с паттерном[%d])", j)
					break
				}
			}

			fmt.Printf("│ %2d. Оригинал: \"%s\"\n", i+1, header)
			fmt.Printf("│     Нормал.:  \"%s\"%s\n", normalizedHeader, matchInfo)
			fmt.Println("│" + strings.Repeat("─", 50))
		}
	})

	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("🎯 Ищем столбцы (XLSX Header -> Search Pattern):")
	for xlsxHeader, pattern := range targetColumns {
		fmt.Printf("   %s -> \"%s\"\n", xlsxHeader, pattern)
	}
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
}

// Функция для нормализации строки
func normalizeString(s string) string {
	// Приводим к нижнему регистру
	s = strings.ToLower(s)

	// Удаляем все пробелы и пробельные символы
	var result strings.Builder
	for _, r := range s {
		// Пропускаем пробелы, табуляции, переносы и другие пробельные символы
		if !unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Функция для получения заголовков таблицы
func getTableHeaders(table *goquery.Selection) []string {
	var headers []string

	// Ищем th элементы
	thElements := table.Find("th")
	if thElements.Length() > 0 {
		thElements.Each(func(i int, s *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(s.Text()))
		})
		return headers
	}

	// Если th нет, ищем в первой строке td
	firstRow := table.Find("tr").First()
	firstRow.Find("td").Each(func(i int, s *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(s.Text()))
	})

	return headers
}
