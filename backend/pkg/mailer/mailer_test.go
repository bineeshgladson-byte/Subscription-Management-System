package mailer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSMTPDeliversToLocalServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		fmt.Fprint(conn, "220 localhost SMTP\r\n")
		scanner := bufio.NewScanner(conn)
		var data strings.Builder
		inData := false
		for scanner.Scan() {
			line := scanner.Text()
			if inData {
				if line == "." {
					inData = false
					fmt.Fprint(conn, "250 accepted\r\n")
					received <- data.String()
				} else {
					data.WriteString(line + "\n")
				}
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				fmt.Fprint(conn, "250 localhost\r\n")
			case strings.HasPrefix(line, "MAIL FROM:") || strings.HasPrefix(line, "RCPT TO:"):
				fmt.Fprint(conn, "250 OK\r\n")
			case line == "DATA":
				inData = true
				fmt.Fprint(conn, "354 End with dot\r\n")
			case line == "QUIT":
				fmt.Fprint(conn, "221 Bye\r\n")
				return
			}
		}
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	n, _ := strconv.Atoi(port)
	sender := NewSMTP(host, n, "sender@example.test", "Subly", "", "")
	if err := sender.Send(context.Background(), Message{To: "recipient@example.test", Subject: "Invoice ₹100", Body: "Total: ₹100\nThank you"}); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-received:
		for _, want := range []string{"To: recipient@example.test", "Total: ₹100", "Thank you"} {
			if !strings.Contains(data, want) {
				t.Fatalf("missing %q in message", want)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("message not received")
	}
}

func TestSMTPCancellationInterruptsStalledServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			buf := make([]byte, 1)
			_, _ = conn.Read(buf)
		}
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	n, _ := strconv.Atoi(port)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = NewSMTP(host, n, "sender@example.test", "Subly", "", "").Send(ctx, Message{To: "recipient@example.test", Subject: "Test", Body: "Test"})
	if err == nil || time.Since(started) > time.Second {
		t.Fatal("SMTP should stop promptly when its context is cancelled")
	}
}
