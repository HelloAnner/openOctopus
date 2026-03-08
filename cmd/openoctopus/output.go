/*
Package main output 负责 CLI 成功/失败输出渲染。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type outputEnvelope struct {
	OK      bool          `json:"ok"`
	Command string        `json:"command"`
	Data    any           `json:"data,omitempty"`
	Error   *errorPayload `json:"error,omitempty"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeCommandSuccess(stdout io.Writer, stderr io.Writer, format string, command string, text string, data any) error {
	if normalizeFormat(format) == "json" {
		return writeJSON(stdout, outputEnvelope{OK: true, Command: command, Data: data})
	}
	_, err := fmt.Fprintln(stdout, text)
	return err
}

func renderCommandError(stdout io.Writer, stderr io.Writer, format string, command string, err error) error {
	cliErr := ensureCLIError(err)
	if normalizeFormat(format) == "json" {
		encodeErr := writeJSON(stderr, outputEnvelope{
			OK:      false,
			Command: command,
			Error:   &errorPayload{Code: cliErr.Code, Message: cliErr.Message, Details: cliErr.Details},
		})
		if encodeErr != nil {
			return encodeErr
		}
		return cliErr
	}
	if _, writeErr := fmt.Fprintln(stderr, cliErr.Message); writeErr != nil {
		return writeErr
	}
	return cliErr
}

func normalizeFormat(format string) string {
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		return "json"
	}
	return "text"
}

func ensureCLIError(err error) *CLIError {
	var cliErr *CLIError
	if err == nil {
		return newCLIError("command_failed", "command failed", exitCodeCommandFailed, nil, nil)
	}
	if errors.As(err, &cliErr) {
		return cliErr
	}
	return newCLIError("command_failed", err.Error(), exitCodeCommandFailed, err, nil)
}

func writeJSON(writer io.Writer, payload outputEnvelope) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(payload)
}
