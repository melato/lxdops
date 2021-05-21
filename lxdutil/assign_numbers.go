package lxdutil

import (
	"fmt"
	"os"
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
	Clean   bool   `name:"clean" usage:"remove numbers for containers that are not selected"`
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
	selectedNames := make(map[string]bool)
	if len(names) > 0 {
		for _, name := range names {
			selectedNames[name] = true
		}
	}
	for _, container := range containers {
		if !t.All && !selectedNames[container.Name] {
			continue
		}
		if t.Running && container.State.Status != Running {
			continue
		}
		err = f(container.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func selectNumbers(numbers []*NamedNumber, names []string) []*NamedNumber {
	namesMap := make(map[string]bool)
	for _, name := range names {
		namesMap[name] = true
	}
	var result []*NamedNumber
	for _, num := range numbers {
		if namesMap[num.Name] {
			result = append(result, num)
		} else {
			fmt.Printf("removing number for: %s\n", num.Name)
		}
	}
	return result
}

func (t *AssignNumbers) AddNumbers(numbers []*NamedNumber, names []string) ([]*NamedNumber, error) {
	var containers []string
	err := t.selectContainers(names, func(name string) error {
		containers = append(containers, name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if t.Clean {
		numbers = selectNumbers(numbers, containers)
	}
	usedNumbers := make(map[int]bool)
	numberedContainers := make(map[string]bool)
	nextNumber := t.First
	for _, num := range numbers {
		usedNumbers[num.Value] = true
		numberedContainers[num.Name] = true
	}
	for _, name := range containers {
		if !numberedContainers[name] {
			for ; ; nextNumber++ {
				if t.Last != 0 && nextNumber > t.Last {
					return nil, fmt.Errorf("no numbers available between %d, %d", t.First, t.Last)
				}
				if !usedNumbers[nextNumber] {
					numbers = append(numbers, &NamedNumber{name, nextNumber})
					nextNumber++
					break
				}
			}
		}
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
	var numbers []*NamedNumber
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
