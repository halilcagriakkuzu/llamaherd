//go:build !unix && !windows

package doctor

func diskFree(path string) (free, total uint64, ok bool) { return 0, 0, false }
