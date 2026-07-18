package cli

import (
	"fmt"
	"io"
	"os"
)

type cliLocale string

const (
	localeEN   cliLocale = "en"
	localePtBR cliLocale = "pt-BR"
)

// cliLocaleFor resolves the process-local allowlist. Unknown values never
// become paths or commands and fall back to English.
func cliLocaleFor(stderr io.Writer) cliLocale {
	value := os.Getenv("POSE_LOCALE")
	switch value {
	case "", "en":
		return localeEN
	case "pt-BR":
		return localePtBR
	default:
		if stderr != nil {
			fmt.Fprintf(stderr, "[WARN] unsupported POSE_LOCALE=%q; falling back to en\n", value)
		}
		return localeEN
	}
}

func cliLocaleValue() cliLocale {
	if os.Getenv("POSE_LOCALE") == "pt-BR" {
		return localePtBR
	}
	return localeEN
}

func cliText(locale cliLocale, english, portuguese string) string {
	if locale == localePtBR {
		return portuguese
	}
	return english
}
