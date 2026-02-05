package branch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Kazuto/Weave/internal/config"
)

type BranchInfo struct {
	Type     string
	TicketID string
	Title    string
}

type Generator struct {
	sanitizer *Sanitizer
	config    config.BranchConfig
}

func NewGenerator(cfg config.BranchConfig) *Generator {
	return &Generator{
		sanitizer: NewSanitizer(),
		config:    cfg,
	}
}

func (g *Generator) GenerateName(info BranchInfo) string {
	if info.Type == "" || info.TicketID == "" {
		return ""
	}

	title := info.Title
	if title == "" {
		title = info.TicketID
	}

	separator := g.config.Sanitization.Separator
	if separator == "" {
		separator = "-"
	}

	// Calculate available length for title part
	// Format: type/ticket-title
	prefixLength := len(info.Type) + 1 + len(info.TicketID) + len(separator)
	availableTitleLength := g.config.MaxLength - prefixLength

	if availableTitleLength < 1 {
		availableTitleLength = 10
	}

	sanitizedTitle := g.sanitizer.Sanitize(title, SanitizationOptions{
		Separator:     separator,
		Lowercase:     g.config.Sanitization.Lowercase,
		RemoveUmlauts: g.config.Sanitization.RemoveUmlauts,
		MaxLength:     availableTitleLength,
	})

	branchName := fmt.Sprintf("%s/%s%s%s", info.Type, info.TicketID, separator, sanitizedTitle)

	// Final length check
	if len(branchName) > g.config.MaxLength {
		maxTitleLength := g.config.MaxLength - prefixLength
		if maxTitleLength > 0 {
			truncatedTitle := sanitizedTitle
			if len(sanitizedTitle) > maxTitleLength {
				truncatedTitle = sanitizedTitle[:maxTitleLength]
				truncatedTitle = strings.TrimSuffix(truncatedTitle, separator)
			}
			branchName = fmt.Sprintf("%s/%s%s%s", info.Type, info.TicketID, separator, truncatedTitle)
		}
	}

	return branchName
}

func (g *Generator) ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	invalidPatterns := []string{
		`^\.`,              // Cannot start with dot
		`\.$`,              // Cannot end with dot
		`\.\.`,             // Cannot contain double dots
		`^/`,               // Cannot start with slash
		`/$`,               // Cannot end with slash
		`//`,               // Cannot contain double slashes
		`\s`,               // Cannot contain spaces
		`[\x00-\x1f\x7f]`,  // Cannot contain control characters
		`[~^:?*\[]`,        // Cannot contain special Git characters
	}

	for _, pattern := range invalidPatterns {
		matched, err := regexp.MatchString(pattern, name)
		if err != nil {
			return fmt.Errorf("error validating branch name: %v", err)
		}
		if matched {
			return fmt.Errorf("invalid branch name: contains invalid pattern")
		}
	}

	return nil
}

func (g *Generator) GetBranchType(typeKey string) string {
	if prefix, ok := g.config.Types[typeKey]; ok {
		return prefix
	}
	return g.config.DefaultType
}
