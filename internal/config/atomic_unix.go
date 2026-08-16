//go:build !windows

package config

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}

func syncParentDirectory(directory string) error {
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
