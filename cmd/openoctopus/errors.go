/*
Package main errors 负责 CLI 错误码与错误包装。
Author: Anner
Created on 2026/3/8
*/
package main

import "errors"

const (
	exitCodeSuccess                  = 0
	exitCodeCommandFailed            = 1
	exitCodeConfigValidationFailed   = 2
	exitCodeSessionNotFound          = 3
	exitCodeRecoveryValidationFailed = 4
)

// CLIError 描述命令层稳定错误。
type CLIError struct {
	Code     string
	Message  string
	ExitCode int
	Cause    error
	Details  any
}

func newCLIError(code string, message string, exitCode int, cause error, details any) *CLIError {
	return &CLIError{Code: code, Message: message, ExitCode: exitCode, Cause: cause, Details: details}
}

func (e *CLIError) Error() string {
	return e.Message
}

func (e *CLIError) Unwrap() error {
	return e.Cause
}

func exitCodeForError(err error) int {
	if err == nil {
		return exitCodeSuccess
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.ExitCode
	}
	return exitCodeCommandFailed
}
