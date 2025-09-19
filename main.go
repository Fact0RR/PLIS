package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	url := "https://www.smd.ru/katalog/poluprovodnikovye_diody_SMD/smd_LED_svetodiody/LED_0603_1204_1206/"

	// Массив искомых столбцов
	targetColumns := []string{
		"Цвет свечения диода",
		"Прямое напряжение (В) при токе 20 мА",
	}

	// Получаем HTML страницу
	resp, err := http.Get(url)
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

	// ВЫВОДИМ ВСЕ ЗАГОЛОВКИ ВСЕХ ТАБЛИЦ В НАЧАЛЕ
	//printAllTableHeaders(tables, targetColumns)

	// Перебираем все найденные таблицы
	tables.Each(func(tableIndex int, table *goquery.Selection) {
		fmt.Printf("═════ Таблица №%d ═════\n", tableIndex+1)

		// Получаем заголовки таблицы
		headers := getTableHeaders(table)
		
		// Нормализуем targetColumns
		normalizedTargets := make([]string, len(targetColumns))
		for i, target := range targetColumns {
			normalizedTargets[i] = normalizeString(target)
		}

		// Находим индексы искомых столбцов
		foundColumns := findTargetColumns(headers, normalizedTargets, targetColumns)
		
		if len(foundColumns) == 0 {
			fmt.Println("❌ Искомые столбцы не найдены")
			fmt.Printf("Доступные заголовки: %v\n", headers)
			fmt.Println()
			return
		}

		// Выводим найденные столбцы
		fmt.Println("✅ Найдены столбцы:")
		for target, index := range foundColumns {
			fmt.Printf("   %s (столбец %d)\n", target, index+1)
		}

		// Получаем и выводим данные
		fmt.Println("\n📊 Данные:")
		rows := table.Find("tr").Slice(1, goquery.ToEnd)
		rows.Each(func(rowIndex int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() == 0 {
				return
			}

			fmt.Printf("\nСтрока %d:\n", rowIndex+1)
			fmt.Println("─" + strings.Repeat("─", 30))
			
			for target, columnIndex := range foundColumns {
				if columnIndex < cells.Length() {
					value := strings.TrimSpace(cells.Eq(columnIndex).Text())
					fmt.Printf("│ %-25s: %s\n", target, value)
				}
			}
		})

		fmt.Println("\n" + strings.Repeat("═", 50) + "\n")
	})
}

// Функция для вывода всех заголовков всех таблиц
func printAllTableHeaders(tables *goquery.Selection, targetColumns []string) {
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("📋 ВСЕ ЗАГОЛОВКИ ВСЕХ ТАБЛИЦ:")
	fmt.Println("═══════════════════════════════════════════════════")

	// Нормализуем targetColumns для сравнения
	normalizedTargets := make([]string, len(targetColumns))
	for i, target := range targetColumns {
		normalizedTargets[i] = normalizeString(target)
	}

	tables.Each(func(tableIndex int, table *goquery.Selection) {
		headers := getTableHeaders(table)
		
		fmt.Printf("\n📊 Таблица №%d - Заголовки (%d):\n", tableIndex+1, len(headers))
		fmt.Println("─" + strings.Repeat("─", 50))
		
		for i, header := range headers {
			normalizedHeader := normalizeString(header)
			matchInfo := ""
			
			// Проверяем совпадение с каждым targetColumn
			for j, normalizedTarget := range normalizedTargets {
				if strings.Contains(normalizedHeader, normalizedTarget) {
					matchInfo = fmt.Sprintf(" ✅ (совпадение с targetColumns[%d])", j)
					break
				}
			}
			
			fmt.Printf("│ %2d. Оригинал: \"%s\"\n", i+1, header)
			fmt.Printf("│     Нормал.:  \"%s\"%s\n", normalizedHeader, matchInfo)
			fmt.Println("│" + strings.Repeat("─", 50))
		}
	})

	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("🎯 Ищем столбцы:", targetColumns)
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
}

// Функция для нормализации строки
func normalizeString(s string) string {
	// Приводим к нижнему регистру
	s = strings.ToLower(s)
	
	// Удаляем все пробельные символы (пробелы, табуляции, переносы)
	re := regexp.MustCompile(`[\s\p{Zs}]+`)
	s = re.ReplaceAllString(s, "")
	
	return s
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

// Функция для поиска индексов целевых столбцов
func findTargetColumns(headers []string, normalizedTargets []string, originalTargets []string) map[string]int {
	result := make(map[string]int)
	
	for i, header := range headers {
		// Нормализуем заголовок таблицы
		normalizedHeader := normalizeString(header)
		
		// Ищем совпадение с нормализованными targetColumns
		for j, normalizedTarget := range normalizedTargets {
			if strings.Contains(normalizedHeader, normalizedTarget) {
				// Используем оригинальное название для вывода
				result[originalTargets[j]] = i
				break
			}
		}
	}
	
	return result
}