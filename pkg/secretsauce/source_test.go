package secretsauce

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSourceReturnsLiteralValue(t *testing.T) {
	value, err := Source{Value: "  literal-secret  "}.Resolve(context.Background(), "SECRET")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if value != "literal-secret" {
		t.Fatalf("value = %q, want trimmed literal", value)
	}
}

func TestSourceEmptyReturnsEmpty(t *testing.T) {
	value, err := Source{}.Resolve(context.Background(), "SECRET")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if value != "" {
		t.Fatalf("value = %q, want empty", value)
	}
}

func TestSourceCommandUsesArgvWithoutShell(t *testing.T) {
	value, err := (Source{Command: secretHelperCommand(t, "--secret", "literal;$(echo injected)")}).Resolve(context.Background(), "SECRET")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if value != "literal;$(echo injected)" {
		t.Fatalf("value = %q, want shell metacharacters preserved literally", value)
	}
}

func TestSourceRejectsRelativeCommand(t *testing.T) {
	_, err := (Source{Command: "op read op://vault/item/password"}).Resolve(context.Background(), "SECRET")
	if err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("error = %v, want absolute path error", err)
	}
}

func TestSourceRejectsLargeCommandOutput(t *testing.T) {
	_, err := (Source{Command: secretHelperCommand(t, "--large-secret")}).Resolve(context.Background(), "SECRET")
	if err == nil || !strings.Contains(err.Error(), "output exceeds") {
		t.Fatalf("error = %v, want output size error", err)
	}
}

func TestSecretCommandHelperProcess(t *testing.T) {
	for i, arg := range os.Args {
		switch arg {
		case "--secret":
			if i+1 >= len(os.Args) {
				os.Exit(2)
			}
			fmt.Print(os.Args[i+1])
			os.Exit(0)
		case "--large-secret":
			fmt.Print(strings.Repeat("x", maxCommandOutputSize+1))
			os.Exit(0)
		}
	}
}

func secretHelperCommand(t *testing.T, args ...string) string {
	t.Helper()

	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatalf("absolute test executable path: %v", err)
	}
	parts := []string{executable, "-test.run=TestSecretCommandHelperProcess", "--"}
	parts = append(parts, args...)
	quoted := make([]string, len(parts))
	for i, part := range parts {
		quoted[i] = strconv.Quote(part)
	}
	return strings.Join(quoted, " ")
}
