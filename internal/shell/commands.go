package shell

import "strings"

type Command struct {
	Name string
	Arg  string
}

func Parse(input string) (Command, bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return Command{}, false
	}

	trimmed = strings.TrimPrefix(trimmed, "/")
	parts := strings.SplitN(trimmed, " ", 2)
	cmd := Command{Name: strings.TrimSpace(parts[0])}
	if len(parts) > 1 {
		cmd.Arg = strings.TrimSpace(parts[1])
	}
	return cmd, cmd.Name != ""
}
