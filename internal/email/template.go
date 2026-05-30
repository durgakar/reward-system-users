package email

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// Renderer loads HTML templates and fills them with client + rule context.
type Renderer struct {
	dir string
}

func NewRenderer(dir string) *Renderer {
	return &Renderer{dir: dir}
}

type TemplateData struct {
	Client    ClientView
	Points    int
	RuleName  string
	Campaign  string
	Profile   ProfileView
}

type ClientView struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
}

type ProfileView struct {
	LifetimeSpendUSD  float64
	LastOrderTotalUSD float64
	PreferredCategory string
}

func (r *Renderer) Render(templateID string, data TemplateData) (html string, text string, err error) {
	path := filepath.Join(r.dir, templateID+".html")
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read template %q: %w", templateID, err)
	}

	tmpl, err := template.New(templateID).Parse(string(content))
	if err != nil {
		return "", "", fmt.Errorf("parse template %q: %w", templateID, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("execute template %q: %w", templateID, err)
	}
	html = buf.String()
	text = stripHTML(html)
	return html, text, nil
}

func stripHTML(s string) string {
	// Minimal fallback for plaintext part; production can use a dedicated .txt template.
	replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n")
	out := replacer.Replace(s)
	var b strings.Builder
	inTag := false
	for _, ch := range out {
		switch ch {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(ch)
			}
		}
	}
	return strings.TrimSpace(b.String())
}
