package template

import (
	"fmt"
	"strings"
	"unicode"
)

// ParseCommand accepts quotes and backslash escapes, without shell evaluation.
// Variables, globs, substitutions, redirections and pipelines are literal argv.
func ParseCommand(command string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	escaped, started := false, false
	for _, r := range command {
		if r == 0 {
			return nil, fmt.Errorf("command contains NUL")
		}
		if escaped {
			word.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			continue
		}
		switch {
		case r == '\'' || r == '"':
			quote = r
			started = true
		case unicode.IsSpace(r):
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
		default:
			word.WriteRune(r)
			started = true
		}
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("unterminated quote or escape")
	}
	if started {
		args = append(args, word.String())
	}
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("empty command")
	}
	return args, nil
}

// CommandArgs parses before expanding so values containing spaces or quotes
// remain one argument and cannot introduce extra executable syntax.
func CommandArgs(step TemplateStep, vars Vars) ([]string, error) {
	if step.Shell {
		return nil, fmt.Errorf("shell mode is not permitted")
	}
	args, err := ParseCommand(step.Run)
	if err != nil {
		return nil, err
	}
	for i := range args {
		args[i] = ExpandVars(args[i], vars)
	}
	return args, nil
}
