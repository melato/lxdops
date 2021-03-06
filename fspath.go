package lxdops

import (
	"sort"
	"strings"
)

type FSPath struct {
	Id   string
	Path string
}

func (t *FSPath) IsDir() bool {
	return strings.HasPrefix(string(t.Path), "/")
}

func (t *FSPath) IsZfs() bool {
	return !t.IsDir()
}

func (t *FSPath) Dir() string {
	if t.IsDir() {
		return string(t.Path)
	} else {
		return "/" + string(t.Path)
	}
}

type FSPathList []FSPath

func (t FSPathList) Len() int           { return len(t) }
func (t FSPathList) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t FSPathList) Less(i, j int) bool { return t[i].Path < t[j].Path }

func (t FSPathList) Sort() { sort.Sort(t) }

// Roots returns the FSPaths that are not children of other FSPaths, within this set.
func (t FSPathList) Roots() []FSPath {
	paths := make([]string, t.Len())
	fsMap := make(map[string]int)
	for i, fs := range t {
		paths[i] = fs.Path
		fsMap[fs.Path] = i
	}
	rootPaths := RootPaths(paths)
	roots := make([]FSPath, len(rootPaths))
	for i, path := range rootPaths {
		roots[i] = t[fsMap[path]]
	}
	return roots
}
