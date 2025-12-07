package utils

import (
	"os"
)

func EnsureDir(path string) {
	os.MkdirAll(path, os.ModePerm)
}
