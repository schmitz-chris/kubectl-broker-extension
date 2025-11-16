package main

import (
	"fmt"
	"strings"
)

type tableColumn struct {
	Title string
	Width int
}

func renderTableHeader(columns []tableColumn, padding int) {
	if padding < 1 {
		padding = 1
	}

	separator := strings.Repeat(" ", padding)
	headerParts := make([]string, len(columns))
	dividerParts := make([]string, len(columns))

	for i, col := range columns {
		width := col.Width
		if width < len(col.Title) {
			width = len(col.Title)
		}

		headerParts[i] = fmt.Sprintf("%-*s", width, col.Title)
		dividerParts[i] = strings.Repeat("-", width)
	}

	fmt.Println(strings.Join(headerParts, separator))
	fmt.Println(strings.Join(dividerParts, separator))
}
