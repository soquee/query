// Package query is used to parse the simple query language used by Soquee.
package query // import "code.soquee.net/query"

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/secure/precis"
)

// IssueStatus represents the status of an issue (closed, open or any).
type IssueStatus int

// A collection of issue statuses.
// Issues may be open or closed, and, in this special case "Any" which means
// "either of those".
const (
	StatusAny IssueStatus = iota
	StatusClosed
	StatusOpen
)

// Query contains the parsed query string split into fields.
// This struct may grow over time and the field order is not part the package
// stability guarantee.
//
// TSVector is a PostgreSQL compatible full text search string.
// It is not guaranteed to be safe from SQL injection and should always be
// parameterized.
type Query struct {
	Status   IssueStatus
	TSVector string
	Assignee string
	Limit    int
	Labels   []string
}

// String parses a query from a string.
func String(q string) *Query {
	/* #nosec */
	parsed, _ := Parse(strings.NewReader(q))
	return parsed
}

// Bytes parses a query from a byte slice.
func Bytes(q []byte) *Query {
	/* #nosec */
	parsed, _ := Parse(bytes.NewReader(q))
	return parsed
}

func isSkipable(r rune) bool {
	return unicode.IsSpace(r) ||
		r == '!' ||
		r == '|' ||
		r == '&' ||
		r == '(' ||
		r == ')'
}

// scanTokens is a copy of bufio.ScanWords except with the isSpace function
// replaced by one that also skips various tsquery operators.
func scanTokens(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := 0

	// Skip leading spaces.
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !isSkipable(r) {
			break
		}
	}
	// Scan until space, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if isSkipable(r) {
			return i + width, data[start:i], nil
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}

const (
	prefixStatus   = "status:"
	prefixLabel    = "label:"
	prefixLimit    = "limit:"
	prefixAssignee = "assignee:"
)

// Parse parses the query string from r and returns a parsed representation.
func Parse(r io.Reader) (*Query, error) {
	parsed := &Query{}
	w := new(strings.Builder)
	s := bufio.NewScanner(r)
	s.Split(scanTokens)

	sep := ""
	no := ""
	for s.Scan() {
		tok := s.Text()
		if idx := strings.IndexByte(tok, ':'); idx > -1 {
			switch tok[:idx+1] {
			case prefixAssignee:
				/* #nosec */
				parsed.Assignee, _ = precis.UsernameCaseMapped.String(tok[len(prefixAssignee):])
				continue
			case prefixLimit:
				/* #nosec */
				parsed.Limit, _ = strconv.Atoi(tok[len(prefixLimit):])
				continue
			case prefixLabel:
				l := tok[len(prefixLabel):]
				if l == "" {
					continue
				}
				parsed.Labels = append(parsed.Labels, l)
				continue
			case prefixStatus:
				switch tok[len(prefixStatus):] {
				case "open":
					parsed.Status = StatusOpen
				case "closed":
					parsed.Status = StatusClosed
				case "any":
					parsed.Status = StatusAny
				default:
				}
				continue
			}
			continue
		}
		t := strings.TrimPrefix(tok, "-")
		if t == "" {
			continue
		}
		if tok != t {
			no = "!"
		}
		fmt.Fprintf(w, "%s%s%s", sep, no, t)
		no = ""
		sep = "&"
	}
	parsed.TSVector = w.String()
	switch {
	case parsed.Limit == 0:
		parsed.Limit = 15
	case parsed.Limit < 10:
		parsed.Limit = 10
	case parsed.Limit > 100:
		parsed.Limit = 100
	}
	return parsed, s.Err()
}
