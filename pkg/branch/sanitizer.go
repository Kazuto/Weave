package branch

import (
	"regexp"
	"strings"
)

type SanitizationOptions struct {
	Separator     string
	Lowercase     bool
	RemoveUmlauts bool
	MaxLength     int
}

type Sanitizer struct{}

func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

func (s *Sanitizer) Sanitize(input string, options SanitizationOptions) string {
	if input == "" {
		return ""
	}

	result := strings.TrimSpace(input)

	if options.RemoveUmlauts {
		result = s.removeUmlauts(result)
	}

	// Remove quotes, parentheses, colons, brackets, and other problematic characters
	result = strings.ReplaceAll(result, "/", " ")
	result = strings.ReplaceAll(result, "\\", " ")
	result = regexp.MustCompile(`["\(\)\[\]{}:;,<>?|*&^%$#@!~` + "`" + `]`).ReplaceAllString(result, "")

	separator := options.Separator
	if separator == "" {
		separator = "-"
	}

	// Handle various hyphen-space combinations
	result = regexp.MustCompile(`\s*-\s*`).ReplaceAllString(result, separator)
	result = regexp.MustCompile(`\s*_\s*`).ReplaceAllString(result, separator)

	// Replace remaining spaces and tabs with the specified separator
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, separator)

	// Remove consecutive separators
	doublePattern := regexp.MustCompile(regexp.QuoteMeta(separator) + `{2,}`)
	result = doublePattern.ReplaceAllString(result, separator)

	// Remove special characters that might cause Git issues
	allowedChars := `a-zA-Z0-9` + regexp.QuoteMeta(separator) + `\.`
	result = regexp.MustCompile(`[^`+allowedChars+`]`).ReplaceAllString(result, "")

	if options.Lowercase {
		result = strings.ToLower(result)
	}

	// Trim to specified length
	if options.MaxLength > 0 && len(result) > options.MaxLength {
		result = result[:options.MaxLength]
		if lastSep := strings.LastIndex(result, separator); lastSep > options.MaxLength/2 {
			result = result[:lastSep]
		}
	}

	// Clean up leading and trailing separators
	result = strings.Trim(result, separator)

	// Ensure no leading dots
	result = strings.TrimLeft(result, ".")

	return result
}

func (s *Sanitizer) removeUmlauts(input string) string {
	replacements := map[string]string{
		"ä": "ae", "Ä": "Ae",
		"ö": "oe", "Ö": "Oe",
		"ü": "ue", "Ü": "Ue",
		"ß": "ss",
		"à": "a", "á": "a", "â": "a", "ã": "a", "å": "a", "æ": "ae",
		"À": "A", "Á": "A", "Â": "A", "Ã": "A", "Å": "A", "Æ": "Ae",
		"è": "e", "é": "e", "ê": "e", "ë": "e",
		"È": "E", "É": "E", "Ê": "E", "Ë": "E",
		"ì": "i", "í": "i", "î": "i", "ï": "i",
		"Ì": "I", "Í": "I", "Î": "I", "Ï": "I",
		"ò": "o", "ó": "o", "ô": "o", "õ": "o", "ø": "o",
		"Ò": "O", "Ó": "O", "Ô": "O", "Õ": "O", "Ø": "O",
		"ù": "u", "ú": "u", "û": "u",
		"Ù": "U", "Ú": "U", "Û": "U",
		"ç": "c", "Ç": "C",
		"ñ": "n", "Ñ": "N",
	}

	result := input
	for char, replacement := range replacements {
		result = strings.ReplaceAll(result, char, replacement)
	}

	return result
}
