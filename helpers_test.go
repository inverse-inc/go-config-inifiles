package inifiles

import (
	"path/filepath"
	"testing"
)

func testdata(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}