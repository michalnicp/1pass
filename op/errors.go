package op

import (
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	errors2 "github.com/pkg/errors"
)

var (

	// ErrInvalidOPConfig represents an invalid op config file.
	ErrInvalidOPConfig = errors.New("invalid op config")
)

var opLogRe = regexp.MustCompile(`^\[LOG\] [\d\/]+ [\d:]+ \(\w+\) (.*)`)

// Error represents an error returned from op cli tool.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("op: %d %s", e.Code, e.Message)
}

// StatusCode returns an http status code for the error.
func (e *Error) StatusCode() int {
	switch e.Code {
	case 1:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// fromExitError converts an exec.ExitError into an Error.
//
//  $ op list items || echo $?
//  [LOG] 2018/05/25 19:02:52 (ERROR) 401: Authentication required.
//  145
func fromExitError(err error) error {
	if err == nil {
		return nil
	}

	exiterr, ok := err.(*exec.ExitError)
	if !ok {
		return err
	}

	operr := Error{
		Message: "unknown error",
	}

	match := opLogRe.FindStringSubmatch(string(exiterr.Stderr))
	if match != nil {
		operr.Message = match[1]
	}

	if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
		operr.Code = status.ExitStatus()
	}

	return &operr
}

func IsUnauthorizedError(err error) bool {
	if operr, ok := errors2.Cause(err).(*Error); ok {
		message := strings.ToLower(operr.Message)
		return strings.Contains(message, "not currently signed in") || // no session
			strings.Contains(message, "authenticated required") // bad session
	}
	return false
}

func IsInvalidCredentialsError(err error) bool {
	if operr, ok := errors2.Cause(err).(*Error); ok {
		message := strings.ToLower(operr.Message)
		return strings.Contains(message, "invalid account key") || // invalid account key format
			strings.Contains(message, "invalid request parameters") || // invalid email format
			strings.Contains(message, "authentication required") // wrong password
	}
	return false
}
