package processor

import (
	"fmt"
	"strings"
)

func TemplateFuncs() map[string]any {
	return map[string]any{
		"HumanToSnakeCase": func(s string) string {
			s = strings.ReplaceAll(s, " ", "_")
			s = strings.ToLower(s)
			return s
		},
		"HumanToKebabCase": func(s string) string {
			s = strings.ReplaceAll(s, " ", "-")
			s = strings.ToLower(s)
			return s
		},
		"Sprintf": fmt.Sprintf,
	}
}
