//go:build e2e

package e2e

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// smtpSink is a minimal ESMTP server that accepts everything and records it.
// Deliberately does NOT advertise STARTTLS or AUTH — net/smtp's SendMail then
// proceeds in the clear, exactly like the no-auth local Postfix relay the
// services target in production.
type mailMsg struct {
	From string
	To   []string
	Data string
}

type smtpSink struct {
	ln net.Listener

	mu   sync.Mutex
	msgs []mailMsg
}

func newSMTPSink() (*smtpSink, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &smtpSink{ln: ln}
	go s.accept()
	return s, nil
}

func (s *smtpSink) host() string { return "127.0.0.1" }
func (s *smtpSink) port() string {
	return fmt.Sprintf("%d", s.ln.Addr().(*net.TCPAddr).Port)
}
func (s *smtpSink) close() { _ = s.ln.Close() }

func (s *smtpSink) accept() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.serve(conn)
	}
}

func (s *smtpSink) serve(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	r := bufio.NewReader(conn)
	reply := func(line string) { fmt.Fprintf(conn, "%s\r\n", line) }

	reply("220 e2e-sink ESMTP")
	var msg mailMsg
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			reply("250 e2e-sink")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			msg = mailMsg{From: angleAddr(line)}
			reply("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			msg.To = append(msg.To, angleAddr(line))
			reply("250 OK")
		case cmd == "DATA":
			reply("354 end with .")
			var b strings.Builder
			for {
				dl, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				b.WriteString(dl)
			}
			msg.Data = b.String()
			s.mu.Lock()
			s.msgs = append(s.msgs, msg)
			s.mu.Unlock()
			reply("250 OK")
		case cmd == "RSET":
			msg = mailMsg{}
			reply("250 OK")
		case cmd == "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 OK")
		}
	}
}

func angleAddr(line string) string {
	if i := strings.IndexByte(line, '<'); i >= 0 {
		if j := strings.IndexByte(line[i:], '>'); j > 0 {
			return line[i+1 : i+j]
		}
	}
	return strings.TrimSpace(line)
}

// waitFor blocks until a recorded message matches (recipient substring AND
// body/data substring), or fails the test.
func (s *smtpSink) waitFor(t *testing.T, to, substr string, timeout time.Duration) mailMsg {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		for _, m := range s.msgs {
			if strings.Contains(strings.Join(m.To, ","), to) && strings.Contains(m.Data, substr) {
				s.mu.Unlock()
				return m
			}
		}
		s.mu.Unlock()
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("no email to %q containing %q within %s", to, substr, timeout)
	return mailMsg{}
}
