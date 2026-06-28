package model

import "strings"

func postgresTextIDRows(fields []string) string {
	rows := make([]string, 0, len(fields))
	for _, field := range fields {
		if isIDColumn(field) {
			rows = append(rows, field+"::text AS "+field)
			continue
		}
		rows = append(rows, field)
	}
	return strings.Join(rows, ",")
}

func isIDColumn(field string) bool {
	return field == "id" ||
		strings.HasSuffix(field, "_id") ||
		field == "created_by" ||
		field == "reviewed_by"
}
