package email

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// StdoutSender logs emails to stdout — ideal for local development.
type StdoutSender struct{}

func NewStdoutSender() *StdoutSender { return &StdoutSender{} }

func (s *StdoutSender) Name() string { return "stdout" }

func (s *StdoutSender) Send(_ context.Context, msg plugin.EmailMessage) error {
	out := os.Stdout
	_, _ = fmt.Fprintf(out, "\n--- EMAIL (%s) ---\nTo: %s\nSubject: %s\nTemplate: %s\n%s\n--- END EMAIL ---\n\n",
		s.Name(), msg.To, msg.Subject, msg.TemplateID, msg.TextBody)
	return nil
}

// FileSender writes emails to a directory for inspection in CI/tests.
type FileSender struct {
	Dir string
}

func NewFileSender(dir string) *FileSender {
	return &FileSender{Dir: dir}
}

func (s *FileSender) Name() string { return "file" }

func (s *FileSender) Send(_ context.Context, msg plugin.EmailMessage) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	path := fmt.Sprintf("%s/%s-%s.eml", s.Dir, sanitize(msg.To), sanitize(msg.TemplateID))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeEML(f, msg)
}

func writeEML(w io.Writer, msg plugin.EmailMessage) error {
	_, err := fmt.Fprintf(w, "To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		msg.To, msg.Subject, msg.HTMLBody)
	return err
}

func sanitize(v string) string {
	out := make([]rune, 0, len(v))
	for _, ch := range v {
		switch ch {
		case '@', '.', '-', '_':
			out = append(out, ch)
		default:
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
				out = append(out, ch)
			} else {
				out = append(out, '_')
			}
		}
	}
	return string(out)
}
