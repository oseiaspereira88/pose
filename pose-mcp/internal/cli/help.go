package cli

import (
	"fmt"
	"io"
	"strings"
)

// hasHelpFlag reports whether "-h" or "--help" is present anywhere in the argument slice.
func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// cmdHelp dispatches help requests. When args is empty, prints top-level help;
// when args contains a command name, dispatches to command-specific help.
func cmdHelp(stdout io.Writer, args []string) int {
	loc := cliLocaleValue()
	if len(args) == 0 {
		fmt.Fprint(stdout, cliText(loc, helpTextEN, helpTextPtBR))
		return 0
	}

	cmd := args[0]
	subargs := args[1:]
	if dispatchCommandHelp(cmd, subargs, stdout, loc) {
		return 0
	}

	// Unknown command: print error and top-level help
	text := func(en, pt string) string { return cliText(loc, en, pt) }
	fmt.Fprintf(stdout, text("No specific help found for %q. Available commands:\n\n", "Nenhuma ajuda específica encontrada para %q. Comandos disponíveis:\n\n"), cmd)
	fmt.Fprint(stdout, cliText(loc, helpTextEN, helpTextPtBR))
	return 0
}

// dispatchCommandHelp renders structured help for a specific command and optional subcommand.
// Returns true if a help entry was found and rendered.
func dispatchCommandHelp(cmd string, args []string, stdout io.Writer, loc cliLocale) bool {
	help, ok := commandHelpCatalog[cmd]
	if !ok {
		return false
	}

	text := func(en, pt string) string { return cliText(loc, en, pt) }

	// Check if a subcommand was requested (e.g. `pose review bundle -h` or `pose help review bundle`)
	var targetSub *SubcommandHelp
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			for i := range help.Subcommands {
				if help.Subcommands[i].Name == a {
					targetSub = &help.Subcommands[i]
					break
				}
			}
		}
	}

	fmt.Fprintf(stdout, "POSE - %s\n\n", help.Name)
	summary := help.SummaryEN
	description := help.DescriptionEN
	if loc == "pt-BR" {
		summary = help.SummaryPtBR
		description = help.DescriptionPtBR
	}

	fmt.Fprintf(stdout, "%s\n\n", summary)

	if targetSub != nil {
		fmt.Fprintln(stdout, text("Usage:", "Uso:"))
		fmt.Fprintf(stdout, "  %s\n\n", targetSub.Usage)
		subSummary := targetSub.SummaryEN
		if loc == "pt-BR" {
			subSummary = targetSub.SummaryPtBR
		}
		fmt.Fprintf(stdout, "%s\n\n", subSummary)
	} else {
		fmt.Fprintln(stdout, text("Usage:", "Uso:"))
		fmt.Fprintf(stdout, "  %s\n\n", help.Usage)

		if description != "" {
			fmt.Fprintln(stdout, text("Description:", "Descrição:"))
			fmt.Fprintf(stdout, "  %s\n\n", description)
		}
	}

	if len(help.Subcommands) > 0 && targetSub == nil {
		fmt.Fprintln(stdout, text("Subcommands:", "Subcomandos:"))
		for _, sub := range help.Subcommands {
			subSummary := sub.SummaryEN
			if loc == "pt-BR" {
				subSummary = sub.SummaryPtBR
			}
			fmt.Fprintf(stdout, "  %-16s %s\n", sub.Name, subSummary)
		}
		fmt.Fprintln(stdout)
	}

	if len(help.Flags) > 0 {
		fmt.Fprintln(stdout, text("Options / Flags:", "Opções / Flags:"))
		for _, f := range help.Flags {
			desc := f.DescriptionEN
			if loc == "pt-BR" {
				desc = f.DescriptionPtBR
			}
			fmt.Fprintf(stdout, "  %-24s %s\n", f.Flag, desc)
		}
		fmt.Fprintln(stdout)
	}

	if len(help.Examples) > 0 {
		fmt.Fprintln(stdout, text("Examples:", "Exemplos:"))
		for _, ex := range help.Examples {
			fmt.Fprintf(stdout, "  $ %s\n", ex)
		}
		fmt.Fprintln(stdout)
	}

	return true
}
