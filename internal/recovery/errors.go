/*
Package recovery errors 定义 recovery 首版的稳定错误。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"errors"
	"strings"
)

var ErrRecoveryLayoutInvalid = errors.New("recovery layout invalid")

type layoutInvalidError struct {
	Missing []string
}

func (e layoutInvalidError) Error() string {
	return "recovery layout invalid: missing " + strings.Join(e.Missing, ", ")
}

func (e layoutInvalidError) Is(target error) bool {
	return target == ErrRecoveryLayoutInvalid
}
