package outil

import (
	"os"
	"runtime"
)

func CheckForFlag(slice []string, str string) ([]string, bool) {
	found := false
	for i, s := range slice {
		if s == str {
			found = true
			slice = append(slice[:i], slice[i+1:]...)
		}
	}
	return slice, found
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func CreateFileIfNotExists(path string) error {
	_, err := os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
	}
	return nil
}

func FindUserHomeDir() string {
	var (
		homeDir string
		err     error
	)
	switch runtime.GOOS {
	case "darwin":
		homeDir, err = os.UserHomeDir()
		if err != nil {
			homeDir = "./"
		}
		homeDir = homeDir + "/"
	case "linux":
		homeDir, err = os.UserHomeDir()
		if err != nil {
			homeDir = "./"
		}
		homeDir = homeDir + "/"
	case "windows":
		homeDir, err = os.UserHomeDir()
		if err != nil {
			homeDir = "./"
		}
		homeDir = homeDir + "\\"
	default:
		homeDir = "./"
	}
	return homeDir
}
