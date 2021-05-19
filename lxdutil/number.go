package lxdutil

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Assign persistent number to containers.
// Used to assign a unique ssh port to each container
type AssignNumbers struct {
	Client  *LxdClient
	File    string `name:"f" usage:"numbers CSV file (container,number)"`
	First   int    `name:"first" usage:"first number"`
	Last    int    `name:"last" usage:"last number (optional)"`
	All     bool   `name:"a" usage:"assign numbers to all containers"`
	Running bool   `name:"r" usage:"use only running containers"`
	Project string `name:"project" usage:"LXD project to use"`
}

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

func (t *AssignNumbers) selectContainers(names []string, f func(name string) error) error {
	server, err := t.Client.ProjectServer(t.Project)
	if err != nil {
		return err
	}
	containers, err := server.GetContainersFull()
	if err != nil {
		return err
	}
	for _, container := range containers {
		err = f(container.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *AssignNumbers) AddNumbers(numbers []*Number, names []string) ([]*Number, error) {
	usedNumbers := make(map[int]bool)
	numberedContainers := make(map[string]bool)
	nextNumber := t.First
	for _, num := range numbers {
		usedNumbers[num.Value] = true
		numberedContainers[num.Name] = true
	}
	err := t.selectContainers(names, func(name string) error {
		if !numberedContainers[name] {
			for ; ; nextNumber++ {
				if t.Last != 0 && nextNumber > t.Last {
					return fmt.Errorf("no numbers available between %d, %d", t.First, t.Last)
				}
				if !usedNumbers[nextNumber] {
					numbers = append(numbers, &Number{name, nextNumber})
					nextNumber++
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return numbers, nil
}

func (t *AssignNumbers) Run(containers []string) error {
	if t.File == "" {
		return fmt.Errorf("missing file")
	}
	if t.First == 0 {
		return fmt.Errorf("missing start")
	}
	var numbers []*Number
	_, err := os.Stat(t.File)
	if err == nil {
		numbers, err = ReadNumbers(t.File)
	} else if os.IsNotExist(err) {
		err = nil
		// the file does not exist, start with empty list
	}

	if err != nil {
		return err
	}
	numbers, err = t.AddNumbers(numbers, containers)
	if err != nil {
		return err
	}
	return WriteNumbers(numbers, t.File)
}
