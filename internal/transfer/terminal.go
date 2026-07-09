package transfer

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// readLine prompts on the terminal and reads back one line of text, with
// normal echo and line editing (backspace, etc.).
//
// It explicitly puts the terminal into raw mode and drives a
// golang.org/x/term.Terminal itself, the same way term.ReadPassword does,
// rather than assuming os.Stdin is already in a normal cooked/echoing mode.
// A plain bufio.Reader over os.Stdin breaks right after a picker.Pick* call:
// go-fuzzyfinder leaves the console in raw mode on at least some platforms,
// which disables echo and stops Enter from producing the '\n' a bufio
// reader waits for, making it look like keystrokes are simply not
// registering.
func readLine(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("enter raw terminal mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	rw := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stderr}

	line, err := term.NewTerminal(rw, prompt).ReadLine()
	if err != nil {
		return "", fmt.Errorf("read line: %w", err)
	}
	return line, nil
}
