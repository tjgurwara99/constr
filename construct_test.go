package main

import (
	"fmt"
	"os"
	"testing"
)

func TestRealMain(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "tmpfile.go")
	if err != nil {
		t.Fatal("failed to create test file:", err)
	}
	fileName := tmpFile.Name()
	defer os.Remove(fileName)
	program := `package test

import "io"

type Something struct {
	some   string
	writer io.Writer
}
`
	fmt.Fprint(tmpFile, program)
	realMain(fileName, "Something")

	expectedData := `package test

import "io"

type Something struct {
	some   string
	writer io.Writer
}

func NewSomething(some string,
	writer io.Writer) *Something {
	return &Something{some: some,
		writer: writer}
}
`
	generated, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal("failed to read fileWritten", err)
	}
	if string(generated) != expectedData {
		t.Fatalf("generated file and expected file don't match: expected %s, generated %s", expectedData, generated)
	}
}
