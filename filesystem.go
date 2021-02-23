package lxdops

import (
	"sort"
)

type FilesystemList []*Filesystem

func (t FilesystemList) Len() int           { return len(t) }
func (t FilesystemList) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t FilesystemList) Less(i, j int) bool { return t[i].Pattern < t[j].Pattern }

func (t FilesystemList) Sort() { sort.Sort(t) }

func (filesystems FilesystemList) Sorted() FilesystemList {
	sorted := make([]*Filesystem, len(filesystems))
	copy(sorted, filesystems)
	FilesystemList(sorted).Sort()
	return sorted

}

func RootFilesystems(filesystems []*Filesystem) []*Filesystem {
	paths := make([]string, len(filesystems))
	fsMap := make(map[string]*Filesystem)
	for i, fs := range filesystems {
		paths[i] = fs.Pattern
		fsMap[fs.Pattern] = fs
	}
	rootPaths := RootPaths(paths)
	roots := make([]*Filesystem, len(rootPaths))
	for i, path := range rootPaths {
		roots[i] = fsMap[path]
	}
	return roots
}
