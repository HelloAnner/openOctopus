package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}

func execute(args []string, stdout io.Writer, stderr io.Writer) int {
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(normalizeRootArgs(args))
	return exitCodeForError(command.Execute())
}

func normalizeRootArgs(args []string) []string {
	if !shouldTreatFirstArgAsConfig(args) {
		return args
	}
	rewritten := make([]string, 0, len(args)+2)
	rewritten = append(rewritten, "run", "--config", args[0])
	rewritten = append(rewritten, args[1:]...)
	return rewritten
}

func shouldTreatFirstArgAsConfig(args []string) bool {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(args[0]))
	return ext == ".yaml" || ext == ".yml"
}
