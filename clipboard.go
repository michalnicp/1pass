package main

import (
	"io"
	"os/exec"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

func writeClipboard(text string) error {
	cmd := exec.Command("xsel", "--input", "--clipboard")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "open stdin pipe")
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, text)
	}()

	_, err = cmd.Output()
	if err != nil {
		return errors.Wrap(err, "xsel error")
	}

	return nil
}

func writeClipboardTimeout(text string, timeout time.Duration) error {
	timeoutStr := strconv.Itoa(int(timeout / time.Millisecond))
	cmd := exec.Command("xsel", "--input", "--clipboard", "--selectionTimeout", timeoutStr)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "open stdin pipe")
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, text)
	}()

	_, err = cmd.Output()
	if err != nil {
		return errors.Wrap(err, "xsel error")
	}

	return nil
}
