// Package secretsauce resolves secret values from either a literal plain-text
// value or the standard output of an external command.
//
// A [Source] carries both a literal Value and a Command. Resolution prefers the
// literal value and otherwise executes the command, capturing its standard
// output as the secret. Commands are executed directly via argv without a shell,
// must reference an absolute-path executable, run under a timeout, and have
// their output size capped.
//
// The package is self-contained and free of any application-specific naming so
// it can be reused independently.
package secretsauce

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const (
	// commandTimeout bounds how long a secret command may run.
	commandTimeout = 30 * time.Second
	// maxCommandOutputSize caps how much standard output a secret command may
	// produce before resolution fails.
	maxCommandOutputSize = 4 * 1024
)

var errCommandOutputTooLarge = errors.New("secret command output too large")

// Source describes how to obtain a single secret value. Value takes precedence
// over Command when both are set.
type Source struct {
	// Value is a literal plain-text secret. When non-empty (after trimming
	// surrounding whitespace) it is returned as-is.
	Value string
	// Command is a command line whose standard output provides the secret. It
	// is parsed into argv and executed directly, without a shell. The
	// executable must be an absolute path.
	Command string
}

// FromEnv builds a [Source] from environment variables using a conventional
// naming scheme. The literal value is read from name, and the command from
// name+"_COMMAND" (falling back to name+"_CMD").
func FromEnv(name string) Source {
	return Source{
		Value:   os.Getenv(name),
		Command: firstNonEmpty(os.Getenv(name+"_COMMAND"), os.Getenv(name+"_CMD")),
	}
}

// Resolve returns the secret value for the source. If Value is set it is
// returned directly; otherwise Command is executed and its standard output is
// returned. The name is used only to produce descriptive error messages and may
// be empty. When neither Value nor Command is set, Resolve returns an empty
// string and a nil error.
func (s Source) Resolve(ctx context.Context, name string) (string, error) {
	if value := strings.TrimSpace(s.Value); value != "" {
		return value, nil
	}
	commandText := strings.TrimSpace(s.Command)
	if commandText == "" {
		return "", nil
	}
	commandArgs, err := splitCommandLine(commandText)
	if err != nil {
		return "", fmt.Errorf("%s command is invalid: %w", name, err)
	}
	if len(commandArgs) == 0 {
		return "", nil
	}
	if !filepath.IsAbs(commandArgs[0]) {
		return "", fmt.Errorf("%s command executable must be an absolute path", name)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	command := exec.CommandContext(cmdCtx, commandArgs[0], commandArgs[1:]...)
	stdout := &limitedBuffer{max: maxCommandOutputSize}
	command.Stdout = stdout
	if err := command.Run(); err != nil {
		if errors.Is(err, errCommandOutputTooLarge) || stdout.overflow {
			return "", fmt.Errorf("%s command output exceeds %d bytes", name, maxCommandOutputSize)
		}
		return "", fmt.Errorf("%s command failed: %w", name, err)
	}
	if stdout.overflow {
		return "", fmt.Errorf("%s command output exceeds %d bytes", name, maxCommandOutputSize)
	}
	value := strings.TrimSpace(stdout.String())
	if value == "" {
		return "", fmt.Errorf("%s command returned empty output", name)
	}
	return value, nil
}

// splitCommandLine tokenizes a command line into argv using shell-like quoting
// and escaping rules, without invoking a shell or interpreting metacharacters.
func splitCommandLine(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	var escaped bool
	var active bool

	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			active = true
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			active = true
			continue
		}

		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			active = true
			continue
		}

		switch {
		case r == '\'' || r == '"':
			quote = r
			active = true
		case unicode.IsSpace(r):
			if active {
				args = append(args, current.String())
				current.Reset()
				active = false
			}
		default:
			current.WriteRune(r)
			active = true
		}
	}

	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if active {
		args = append(args, current.String())
	}
	return args, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
