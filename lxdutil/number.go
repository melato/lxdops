package lxdutil

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

type Number struct {
	Name  string
	Value int
}

func ReadNumbers(file string) ([]*Number, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(bytes.NewReader(content))
	var result []*Number
	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(fields) != 2 {
			return nil, fmt.Errorf("expected 2 fields: %v", fields)
		}
		var num Number
		num.Name = fields[0]
		num.Value, err = strconv.Atoi(fields[1])
		if err != nil {
			return nil, err
		}
		result = append(result, &num)
	}
	return result, nil
}

func WriteNumbers(numbers []*Number, file string) error {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	for _, num := range numbers {
		writer.Write([]string{num.Name, strconv.Itoa(num.Value)})
	}
	writer.Flush()
	return os.WriteFile(file, buf.Bytes(), os.FileMode(0664))
}
