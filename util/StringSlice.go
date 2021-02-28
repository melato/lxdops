package util

import (
	"sort"
)

type StringSlice []string

func (t StringSlice) Equals(b []string) bool {
	if len(t) != len(b) {
		return false
	}
	for i, s := range t {
		if s != b[i] {
			return false
		}
	}
	return true
}

/** Return the elements of this slice that are not in the b slice */
func (t StringSlice) Diff(b []string) []string {
	var result []string
	m := StringSlice(b).ToMap()
	for _, s := range t {
		_, inB := m[s]
		if !inB {
			result = append(result, s)
		}
	}
	return result
}

func (t StringSlice) Sorted() []string {
	result := make([]string, len(t))
	for i, s := range t {
		result[i] = s
	}
	sort.Strings(result)
	return result
}

func (t StringSlice) RemoveDuplicates() []string {
	var result []string
	set := make(map[string]bool)
	for _, s := range t {
		if _, exists := set[s]; !exists {
			set[s] = true
			result = append(result, s)
		}
	}
	return result
}

func (t StringSlice) Remove(remove string) []string {
	var removed bool
	var result []string
	for i, s := range t {
		if s == remove {
			if !removed {
				removed = true
				result = append(result, t[0:i]...)
			}
		} else if removed {
			result = append(result, s)

		}
	}
	if removed {
		return result
	} else {
		return t
	}
}

func (t StringSlice) ToMap() map[string]bool {
	result := make(map[string]bool)
	for _, s := range t {
		result[s] = true
	}
	return result
}
