// Package mailer sends plain-text email over SMTP.
// In development every message lands in Mailpit (http://localhost:8025).
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer is an interface so tests can capture emails instead of sending them.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

type SMTP struct {
	host, from, fromName, username, password string
	port                                     int
}

// NewSMTP: fromName is the sender name shown in inboxes, e.g. "Subly".
func NewSMTP(host string, port int, from, fromName, username, password string) *SMTP {
	return &SMTP{host: host, port: port, from: from, fromName: fromName, username: username, password: password}
}

func (m *SMTP) Send(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	to := clean(msg.To)
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s <%s>\r\n", mime.QEncoding.Encode("utf-8", clean(m.fromName)), m.from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", clean(msg.Subject)))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(strings.ReplaceAll(msg.Body, "\n", "\r\n"))

	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	conn, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(b.String())); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	// DATA was accepted: failure to close the connection is not a failed send.
	_ = client.Quit()
	return nil
}

// clean strips CR/LF so user input can't inject extra email headers.
func clean(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
