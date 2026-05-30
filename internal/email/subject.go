package email

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

// ValidateTemplates ensures every referenced template file exists.
func ValidateTemplates(dir string, names ...string) error {
	for _, name := range names {
		path := filepath.Join(dir, name+".html")
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("missing template %q: %w", name, err)
		}
	}
	return nil
}

// RenderSubject applies Go template syntax to email subjects.
func RenderSubject(subject string, data TemplateData) (string, error) {
	tmpl, err := template.New("subject").Option("missingkey=zero").Parse(subject)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
