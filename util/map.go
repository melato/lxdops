package util

import (
	"os"
	"sort"

	"melato.org/export/table3"
)

func MapKeys(m map[string]string) []string {
	var keys []string
	for key, _ := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func PrintMap(m map[string]string) {
	var key, value string
	writer := &table.FixedWriter{Writer: os.Stdout}
	writer.Columns(
		table.NewColumn("KEY", func() interface{} { return key }),
		table.NewColumn("VALUE", func() interface{} { return value }),
	)
	keys := MapKeys(m)
	for _, key = range keys {
		value = m[key]
		writer.WriteRow()
	}
	writer.End()
}
