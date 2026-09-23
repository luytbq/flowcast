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
		t.Fatalf("JSON hỏng: %v\n%s", err, rec.Body.String())
	}
	return r
}

func TestBuildTraVeDrawioGiongGolden(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", "05-merge-node.md", readCase(t, "05-merge-node.md"), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("mã %d: %s", rec.Code, rec.Body.String())
	}
	r := decode(t, rec)
	golden, err := os.ReadFile("../../conformance/golden/05-merge-node.drawio")
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK || r.Drawio != string(golden) || r.Filename != "05-merge-node.drawio" || r.Stats.Items != 5 {
		t.Errorf("ok=%v filename=%q stats=%+v, drawio khớp golden: %v", r.OK, r.Filename, r.Stats, r.Drawio == string(golden))
	}
}

func TestDownloadTraThangFile(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build?download=1", "Sơ đồ đặt hàng.md", readCase(t, "05-merge-node.md"), nil)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "<mxfile") {
		t.Fatalf("mã %d: %.200s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/vnd.jgraph.mxfile") {
		t.Errorf("Content-Type %q", ct)
	}
	want := `attachment; filename*=utf-8''S%C6%A1%20%C4%91%E1%BB%93%20%C4%91%E1%BA%B7t%20h%C3%A0ng.drawio`
	if cd := rec.Header().Get("Content-Disposition"); cd != want {
		t.Errorf("Content-Disposition %q", cd)
	}
}

func TestTenFileChiLayPhanTen(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", `..\..\etc/bang.md`, readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); r.Filename != "bang.drawio" {
		t.Errorf("filename %q", r.Filename)
	}
}

func TestBangLoiTra422KemIssue(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	for _, url := range []string{"/api/build", "/api/check"} {
		rec := upload(t, h, url, "loi.md", readCase(t, "90-invalid-dangling-ref.md"), nil)
		r := decode(t, rec)
		if rec.Code != http.StatusUnprocessableEntity || r.OK || r.Drawio != "" || len(r.Issues) == 0 {
			t.Fatalf("%s: mã %d, %+v", url, rec.Code, r)
		}
		found := false
		for _, i := range r.Issues {
			if i.Level == "error" && i.Code != "" && i.Location != "" && i.Loc.Row > 0 {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: không có issue lỗi mang mã và vị trí: %+v", url, r.Issues)
		}
	}
}

func TestCheckKhongTraDrawio(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/check", "a.md", readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); rec.Code != http.StatusOK || !r.OK || r.Drawio != "" || r.Source != "markdown" {
		t.Errorf("mã %d, %+v", rec.Code, r)
	}
}

func TestXlsxVaTuyChonDoc(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", "hai.xlsx", readCase(t, "xlsx-08-two-flows.xlsx"), map[string]string{"sheet": "Hai"})
	r := decode(t, rec)
	if rec.Code != http.StatusOK || !r.OK || r.Source == "" {
		t.Fatalf("mã %d: %s", rec.Code, rec.Body.String())
	}
	rec = upload(t, h, "/api/build", "hai.xlsx", readCase(t, "xlsx-08-two-flows.xlsx"), map[string]string{"sheet": "Không có"})
	if r := decode(t, rec); rec.Code != http.StatusUnprocessableEntity || r.Error == nil {
		t.Errorf("sheet không có: mã %d, %s", rec.Code, rec.Body.String())
	}
}

func TestVuotGioiHanTra413(t *testing.T) {
	lim := flowcast.WebLimits
	lim.MaxRows = 5
	h := testServer(lim, 2)
	rec := upload(t, h, "/api/build", "a.md", readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); rec.Code != http.StatusRequestEntityTooLarge || r.Error == nil || r.Error.Code != "limit.rows" {
		t.Errorf("mã %d, %s", rec.Code, rec.Body.String())
	}
	lim = flowcast.WebLimits
	lim.MaxBytes = 1000
	h = testServer(lim, 2)
	big := bytes.Repeat([]byte("x"), 200<<10)
	rec = upload(t, h, "/api/build", "a.md", big, nil)
	if r := decode(t, rec); rec.Code != http.StatusRequestEntityTooLarge || r.Error == nil || r.Error.Code != "limit.bytes" {
		t.Errorf("thân quá lớn: mã %d, %s", rec.Code, rec.Body.String())
	}
	rec = upload(t, h, "/api/build", "a.md", bytes.Repeat([]byte("x"), 2000), nil)
	if r := decode(t, rec); rec.Code != http.StatusRequestEntityTooLarge || r.Error == nil || r.Error.Code != "limit.bytes" {
		t.Errorf("file quá lớn trong phần dư multipart: mã %d, %s", rec.Code, rec.Body.String())
	}
}

// countReader đếm số byte máy chủ thật sự đọc từ thân yêu cầu.
type countReader struct {
	r io.Reader
	n int64
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// Thân yêu cầu phải bị chặn ngay lúc đọc, không phải đọc hết rồi mới so với
// giới hạn: file 8 MB gửi tới máy chủ giới hạn 1 KB không được nằm trọn trong
// bộ nhớ.
func TestThanYeuCauBiChanNgayLucDoc(t *testing.T) {
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
		t.Fatalf("mã %d, %s", rec.Code, rec.Body.String())
	}
	if max := int64(lim.MaxBytes) + multipartSlack + 64<<10; counted.n > max {
		t.Errorf("máy chủ đọc %d byte, quá mức cần để biết là vượt giới hạn (%d)", counted.n, max)
	}
}

func TestThieuFileTra400(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := upload(t, h, "/api/build", "", nil, map[string]string{"title": "x"})
	if r := decode(t, rec); rec.Code != http.StatusBadRequest || r.Error == nil {
		t.Errorf("mã %d, %s", rec.Code, rec.Body.String())
	}
}

func TestSaiPhuongThucTra405(t *testing.T) {
	h := testServer(flowcast.WebLimits, 2)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/build", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("mã %d", rec.Code)
	}
}

func TestBanThiTra503(t *testing.T) {
	s := testServer(flowcast.WebLimits, 1)
	// Chỗ duy nhất đã có người giữ: yêu cầu phải được báo bận thay vì xếp hàng mãi.
	s.slot <- struct{}{}
	s.wait = 0
	rec := upload(t, s, "/api/build", "a.md", readCase(t, "05-merge-node.md"), nil)
	if r := decode(t, rec); rec.Code != http.StatusServiceUnavailable || r.Error == nil || r.Error.Code != "server.busy" {
		t.Errorf("mã %d, %s", rec.Code, rec.Body.String())
	}
}

func TestTrangChuVaHeaderBaoVe(t *testing.T) {
	h := testServer(flowcast.WebLimits, 1)
	for _, p := range []string{"/", "/app.js", "/style.css", "/healthz"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: mã %d", p, rec.Code)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("%s: thiếu header bảo vệ", p)
		}
	}
}
