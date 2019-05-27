package query_test

import (
	"strconv"
	"strings"
	"testing"

	"code.soquee.net/query"
)

var queryTests = [...]struct {
	in       string
	out      string
	status   query.IssueStatus
	limit    int
	assignee string
	labels   string
}{
	0:  {},
	1:  {in: " &| !(\t)  ", out: ""},
	2:  {in: "a  test\tstatus:a", out: "a&test", status: query.StatusAny},
	3:  {in: "-a t-est- of -time status:open", out: "!a&t-est-&of&!time", status: query.StatusOpen},
	4:  {in: "- test", out: "test"},
	5:  {in: "new - status:closed test", out: "new&test", status: query.StatusClosed},
	6:  {in: "status:any test -", out: "test"},
	7:  {in: "status:open another:bad", status: query.StatusOpen},
	8:  {in: "status:open another:bad assignee:me label:a label:b label: ", assignee: "me", status: query.StatusOpen, labels: "a,b"},
	9:  {in: "limit:80", limit: 80},
	10: {in: "limit:110", limit: 100},
	11: {in: "limit:-1", limit: 10},
	12: {in: "status:open status:bad", status: query.StatusOpen},
	13: {in: "status:open status:any", status: query.StatusAny},
}

func doTests(t *testing.T, f func(string) *query.Query) {
	for i, tc := range queryTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			parsed := f(tc.in)
			if parsed.TSVector != tc.out {
				t.Errorf("Unexpected output: want=%q, got=%q", tc.out, parsed.TSVector)
			}
			if parsed.Status != tc.status {
				t.Errorf("Unexpected status: want=%q, got=%q", tc.status, parsed.Status)
			}
			if labels := strings.Join(parsed.Labels, ","); labels != tc.labels {
				t.Errorf("Unexpected labels: want=%q, got=%q", tc.labels, labels)
			}
			if tc.limit == 0 {
				tc.limit = 15
			}
			if parsed.Limit != tc.limit {
				t.Errorf("Unexpected limit: want=%d, got=%d", tc.limit, parsed.Limit)
			}
			if parsed.Assignee != tc.assignee {
				t.Errorf("Unexpected assignee: want=%q, got=%q", tc.assignee, parsed.Assignee)
			}
		})
	}
}

func TestQueryString(t *testing.T) {
	doTests(t, query.String)
}

func TestQueryBytes(t *testing.T) {
	doTests(t, func(q string) *query.Query {
		return query.Bytes([]byte(q))
	})
}

const errText = "Bad err"

type errReader struct{}

func (errReader) Error() string {
	return errText
}

func (e errReader) Read([]byte) (int, error) {
	return 0, e
}

func TestBadRead(t *testing.T) {
	b := new(strings.Builder)
	_, err := query.Parse(errReader{})
	if err.Error() != errText {
		t.Errorf("Unexpected error: want=%q, got=%q", errText, err)
	}
	if s := b.String(); s != "" {
		t.Errorf("Unexpected write to buffer: %q", s)
	}
}
