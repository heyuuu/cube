package console

import (
	"fmt"
	"strings"
)

func PrintTable(headers []string, rows [][]string) {
	// 计算列数
	colCount := len(headers)
	for _, line := range rows {
		colCount = max(colCount, len(line))
	}

	// 计算每列宽度
	colWidths := make([]int, colCount)
	for i, field := range headers {
		colWidths[i] = unicodeWidth(field)
	}
	for _, line := range rows {
		for i, field := range line {
			colWidths[i] = max(colWidths[i], unicodeWidth(field))
		}
	}

	// 计算分隔线
	splitLineBuilder := strings.Builder{}
	splitLineBuilder.WriteString("+")
	for _, fieldLen := range colWidths {
		splitLineBuilder.WriteString(strings.Repeat("-", fieldLen+2))
		splitLineBuilder.WriteString("+")
	}
	splitLine := splitLineBuilder.String()

	// 绘制表格
	fmt.Println(splitLine)
	printTableLine(colCount, headers, colWidths)
	fmt.Println(splitLine)
	for _, line := range rows {
		printTableLine(colCount, line, colWidths)
	}
	fmt.Println(splitLine)
}

func printTableLine(colCount int, fields []string, maxLen []int) {
	builder := strings.Builder{}
	builder.WriteString("|")
	for i := 0; i < colCount; i++ {
		var field string
		if i < len(fields) {
			field = fields[i]
		}
		fieldWidth := unicodeWidth(field)
		if fieldWidth < maxLen[i] {
			field = field + strings.Repeat(" ", maxLen[i]-fieldWidth)
		}
		builder.WriteString(" " + field + " |")
	}
	fmt.Println(builder.String())
}

// 计算字符串宽度，支持 unicode
func unicodeWidth(str string) int {
	var width int
	for _, r := range []rune(str) {
		rint := int64(r)
		if rint <= 0x0019 {
			width += 0
		} else if rint <= 0x1fff {
			width += 1
		} else if rint <= 0xff60 {
			width += 2
		} else if rint <= 0xff9f {
			width += 1
		} else {
			width += 2
		}
	}
	return width
}
