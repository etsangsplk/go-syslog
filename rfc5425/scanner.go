package rfc5425

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
)

// eof represents a marker byte for the end of the reader
var eof = byte(0)

// ws represents the whitespace
var ws = byte(32)

// isDigit returns true if the byte represents a number in [0,9]
func isDigit(ch byte) bool {
	return (ch >= 47 && ch <= 57)
}

// isNonZeroDigit returns true if the byte represents a number in ]0,9]
func isNonZeroDigit(ch byte) bool {
	return (ch >= 48 && ch <= 57)
}

// isWhitespace returns true if the byte represents a space
func isWhitespace(ch byte) bool {
	return ch == 32
}

// Scanner represents a lexical scanner
type Scanner struct {
	r      *bufio.Reader
	msglen uint64
	ready  bool
}

// NewScanner returns a new instance of Scanner
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		r: bufio.NewReader(r),
	}
}

// read reads the next byte from the buffered reader
// it returns the byte(0) if an error occurs (or io.EOF is returned)
func (s *Scanner) read() byte {
	b, err := s.r.ReadByte()
	if err != nil {
		return eof
	}
	return b
}

// unread places the previously read byte back on the reader
func (s *Scanner) unread() {
	_ = s.r.UnreadByte()
}

// Scan returns the next token
func (s *Scanner) Scan() (tok Token) {
	// Read the next byte.
	b := s.read()

	if isNonZeroDigit(b) {
		s.unread()
		s.ready = false
		return s.scanMsgLen()
	}

	// Otherwise read the individual character
	switch b {
	case eof:
		s.ready = false
		return Token{
			typ: EOF,
		}
	case ws:
		s.ready = true
		return Token{
			typ: WS,
			lit: []byte{ws},
		}
	default:
		if s.msglen > 0 && s.ready {
			s.unread()
			return s.scanSyslogMsg()
		}
		//s.ready = false // (todo) > verify ...
	}

	return Token{
		typ: ILLEGAL,
		lit: []byte{b},
	}
}

func (s *Scanner) scanMsgLen() Token {
	// Create a buffer and read the current character into it
	var buf bytes.Buffer
	buf.WriteByte(s.read())

	// Read every subsequent digit character into the buffer
	// Non-digit characters and EOF will cause the loop to exit
	for {
		if b := s.read(); b == eof {
			break
		} else if !isDigit(b) {
			s.unread()
			break
		} else {
			buf.WriteByte(b)
		}
	}

	msglen := buf.String()
	s.msglen, _ = strconv.ParseUint(msglen, 10, 64)

	return Token{
		typ: MSGLEN,
		lit: buf.Bytes(),
	}
}

func (s *Scanner) scanSyslogMsg() Token {
	// Create a buffer and read the current character into it
	buf := make([]byte, 0, s.msglen)

	for i := uint64(0); i < s.msglen; i++ {
		b := s.read()

		if b == eof {
			return Token{
				typ: EOF,
				lit: buf,
			}
		}

		buf = append(buf, b)
	}

	s.ready = false
	s.msglen = 0
	return Token{
		typ: SYSLOGMSG,
		lit: buf,
	}
}