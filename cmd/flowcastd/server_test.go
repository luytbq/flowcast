package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/layout"
)

func testServer(lim flowcast.Limits, concurrent int) *server {
	return newServer(lim, concurrent, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func upload(t *testing.T, h http.Handler, url, name string, data []byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if name != "" {
		fw, err := mw.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(data)
	}
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func readCase(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../conformance/cases/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var r apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("broken JSON: %v\n%s", err, rec.Body.String())
	}
	return r
}

func TestBuildReturnsDrawioMatchingGolden(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", "05-merge-node.md", readCase(t, "05-merge-node.md"), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	r := decode(t, rec)
	golden, err := os.ReadFile("../../conformance/golden/cases/05-merge-node.drawio")
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK || r.Drawio != string(golden) || r.Filename != "05-merge-node.drawio" || r.Stats.Items != 5 {
		t.Errorf("ok=%v filename=%q stats=%+v, drawio matches golden: %v", r.OK, r.Filename, r.Stats, r.Drawio == string(golden))
	}
}

func TestDownloadReturnsFileDirectly(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build?download=1", "Sơ đồ đặt hàng.md", readCase(t, "05-merge-node.md"), nil)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "<mxfile") {
		t.Fatalf("code %d: %.200s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/vnd.jgraph.mxfile") {
		t.Errorf("Content-Type %q", ct)
	}
	want := `attachment; filename*=utf-8''S%C6%A1%20%C4%91%E1%BB%93%20%C4%91%E1%BA%B7t%20h%C3%A0ng.drawio`
	if cd := rec.Header().Get("Content-Disposition"); cd != want {
		t.Errorf("Content-Disposition %q", cd)
	}
}

func TestFilenameKeepsOnlyBaseName(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", `..\..\etc/bang.md`, readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); r.Filename != "bang.drawio" {
		t.Errorf("filename %q", r.Filename)
	}
}

func TestInvalidTableReturns422WithIssues(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	for _, url := range []string{"/api/build", "/api/check"} {
		rec := upload(t, h, url, "loi.md", readCase(t, "90-invalid-dangling-ref.md"), nil)
		r := decode(t, rec)
		if rec.Code != http.StatusUnprocessableEntity || r.OK || r.Drawio != "" || len(r.Issues) == 0 {
			t.Fatalf("%s: code %d, %+v", url, rec.Code, r)
		}
		found := false
		for _, i := range r.Issues {
			if i.Level == "error" && i.Code != "" && i.Location != "" && i.Loc.Row > 0 {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no error issue carrying a code and location: %+v", url, r.Issues)
		}
	}
}

func TestCheckReturnsNoDrawio(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/check", "a.md", readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); rec.Code != http.StatusOK || !r.OK || r.Drawio != "" || r.Source != "markdown" {
		t.Errorf("code %d, %+v", rec.Code, r)
	}
}

func TestXlsxAndReadOptions(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", "hai.xlsx", readCase(t, "xlsx-08-two-flows.xlsx"), map[string]string{"sheet": "Hai"})
	r := decode(t, rec)
	if rec.Code != http.StatusOK || !r.OK || r.Source == "" {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	rec = upload(t, h, "/api/build", "hai.xlsx", readCase(t, "xlsx-08-two-flows.xlsx"), map[string]string{"sheet": "Không có"})
	if r := decode(t, rec); rec.Code != http.StatusUnprocessableEntity || r.Error == nil {
		t.Errorf("missing sheet: code %d, %s", rec.Code, rec.Body.String())
	}
}

func TestExceedingLimitReturns413(t *testing.T) {
	lim := flowcast.WebLimits
	lim.MaxRows = 5
	h := testServer(lim, 2)
	rec := upload(t, h, "/api/build", "a.md", readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); rec.Code != http.StatusRequestEntityTooLarge || r.Error == nil || r.Error.Code != "limit.rows" {
		t.Errorf("code %d, %s", rec.Code, rec.Body.String())
	}
	lim = flowcast.WebLimits
	lim.MaxBytes = 1000
	h = testServer(lim, 2)
	big := bytes.Repeat([]byte("x"), 200<<10)
	rec = upload(t, h, "/api/build", "a.md", big, nil)
	if r := decode(t, rec); rec.Code != http.StatusRequestEntityTooLarge || r.Error == nil || r.Error.Code != "limit.bytes" {
		t.Errorf("body too large: code %d, %s", rec.Code, rec.Body.String())
	}
	rec = upload(t, h, "/api/build", "a.md", bytes.Repeat([]byte("x"), 2000), nil)
	if r := decode(t, rec); rec.Code != http.StatusRequestEntityTooLarge || r.Error == nil || r.Error.Code != "limit.bytes" {
		t.Errorf("file too large within the multipart allowance: code %d, %s", rec.Code, rec.Body.String())
	}
}

// countReader counts the bytes the server actually reads from the request body.
type countReader struct {
	r io.Reader
	n int64
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// The request body must be cut off while it is being read, not read in full and
// then compared with the limit: an 8 MB file sent to a server with a 1 KB limit
// must not end up entirely in memory.
func TestRequestBodyIsCutOffWhileReading(t *testing.T) {
	lim := flowcast.WebLimits
	lim.MaxBytes = 1000
	h := testServer(lim, 2)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "to.md")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(bytes.Repeat([]byte("x"), 8<<20))
	mw.Close()
	counted := &countReader{r: bytes.NewReader(body.Bytes())}
	req := httptest.NewRequest(http.MethodPost, "/api/build", counted)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if r := decode(t, rec); rec.Code != http.StatusRequestEntityTooLarge || r.Error == nil {
		t.Fatalf("code %d, %s", rec.Code, rec.Body.String())
	}
	if max := int64(lim.MaxBytes) + multipartSlack + 64<<10; counted.n > max {
		t.Errorf("server read %d bytes, more than needed to know the limit was exceeded (%d)", counted.n, max)
	}
}

func TestMissingFileReturns400(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", "", nil, map[string]string{"title": "x"})
	if r := decode(t, rec); rec.Code != http.StatusBadRequest || r.Error == nil {
		t.Errorf("code %d, %s", rec.Code, rec.Body.String())
	}
}

func TestWrongMethodReturns405(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/build", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("code %d", rec.Code)
	}
}

func TestBusyReturns503(t *testing.T) {
	s := testServer(flowcast.WebLimits, 1)
	// The only slot is already taken: the request must be told busy instead of queueing forever.
	s.slot <- struct{}{}
	s.wait = 0
	rec := upload(t, s, "/api/build", "a.md", readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); rec.Code != http.StatusServiceUnavailable || r.Error == nil || r.Error.Code != "server.busy" {
		t.Errorf("code %d, %s", rec.Code, rec.Body.String())
	}
}

func TestFieldsDeclaresEveryParameter(t *testing.T) {
	h := testServer(flowcast.WebLimits, 1)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/fields", nil))
	var got struct {
		Directions []string
		Fields     []apiField
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Fields) != len(layout.Fields()) || len(got.Directions) != 4 {
		t.Fatalf("%d fields, %d directions", len(got.Fields), len(got.Directions))
	}
	for _, f := range got.Fields {
		if f.Name == "" || f.Help == "" || f.Lo > f.Default || f.Default > f.Hi {
			t.Errorf("broken field: %+v", f)
		}
	}
}

func TestLayoutParametersAndDirectionOverWeb(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	data := readCase(t, "05-merge-node.md")
	base := decode(t, upload(t, h, "/api/build", "a.md", data, nil))
	wide := decode(t, upload(t, h, "/api/build", "a.md", data, map[string]string{"task-min-w": "400"}))
	if !wide.OK || wide.Stats.W <= base.Stats.W {
		t.Errorf("task-min-w did not change the layout: %v then %v", base.Stats, wide.Stats)
	}
	lr := decode(t, upload(t, h, "/api/build", "a.md", data, map[string]string{"direction": "LR"}))
	if !lr.OK || lr.Stats.W <= lr.Stats.H || !strings.Contains(lr.Drawio, "horizontal=0") {
		t.Errorf("direction LR must give a horizontal diagram with horizontal lanes: %v", lr.Stats)
	}
	// track-gap accepts 0, so "x" is only caught if the server really checks the
	// text converts to a number rather than silently taking 0.
	for _, bad := range []map[string]string{{"task-min-w": "x"}, {"track-gap": "x"}, {"task-min-w": "-5"},
		{"task-min-w": "999999"}, {"direction": "XY"}} {
		rec := upload(t, h, "/api/build", "a.md", data, bad)
		if r := decode(t, rec); rec.Code != http.StatusBadRequest || r.Error == nil {
			t.Errorf("%v: code %d, %s", bad, rec.Code, rec.Body.String())
		}
	}
}

func TestHomePageAndSecurityHeaders(t *testing.T) {
	h := testServer(flowcast.WebLimits, 1)
	for _, p := range []string{"/", "/app.js", "/style.css", "/healthz"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: code %d", p, rec.Code)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("%s: missing security headers", p)
		}
	}
}
