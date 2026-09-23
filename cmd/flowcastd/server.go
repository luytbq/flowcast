package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/luytbq/flowcast"
	"github.com/luytbq/flowcast/layout"
	"github.com/luytbq/flowcast/model"
	"github.com/luytbq/flowcast/source"
)

//go:embed static
var static embed.FS

// multipartSlack là phần dư cho header của multipart ngoài chính nội dung file.
const multipartSlack = 64 << 10

type server struct {
	h    http.Handler
	lim  flowcast.Limits
	slot chan struct{} // số lần dựng chạy cùng lúc
	wait time.Duration // chờ bao lâu để có chỗ trước khi báo bận
	log  *slog.Logger
}

func newServer(lim flowcast.Limits, concurrent int, log *slog.Logger) *server {
	s := &server{lim: lim, slot: make(chan struct{}, concurrent), wait: 5 * time.Second, log: log}
	mux := http.NewServeMux()
	// Đăng ký từng file thay vì "GET /": mẫu đó khớp mọi đường dẫn, và GET vào
	// /api/build sẽ ra 404 của file server thay vì 405.
	files, _ := fs.Sub(static, "static")
	fileServer := http.FileServerFS(files)
	for _, p := range []string{"GET /{$}", "GET /app.js", "GET /style.css"} {
		mux.Handle(p, fileServer)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "ok\n") })
	mux.HandleFunc("GET /api/fields", s.fields)
	mux.HandleFunc("POST /api/build", func(w http.ResponseWriter, r *http.Request) { s.handle(w, r, true) })
	mux.HandleFunc("POST /api/check", func(w http.ResponseWriter, r *http.Request) { s.handle(w, r, false) })
	s.h = s.logged(secure(mux))
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.h.ServeHTTP(w, r) }

// secure đặt các header chặn trang bị nhúng hay bị đoán sai kiểu nội dung. Trang
// không có script hay style nội tuyến nên CSP khóa được về 'self'.
func secure(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		h.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *server) logged(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{w, http.StatusOK}
		h.ServeHTTP(sw, r)
		s.log.Info("request", "method", r.Method, "path", r.URL.Path, "status", sw.status,
			"ms", time.Since(start).Milliseconds())
	})
}

// apiField là một trường cấu hình ở dạng JSON. Trang web dựng form từ danh
// sách này, nên thêm một trường trong layout là đủ để nó hiện ra ở web.
type apiField struct {
	Name    string `json:"name"`
	Help    string `json:"help"`
	Default int    `json:"default"`
	Lo      int    `json:"lo"`
	Hi      int    `json:"hi"`
}

func (s *server) fields(w http.ResponseWriter, _ *http.Request) {
	var out []apiField
	for _, f := range layout.Fields() {
		out = append(out, apiField{f.Name, f.Help, f.Default, f.Lo, f.Hi})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"directions": []string{layout.DirTD, layout.DirBT, layout.DirLR, layout.DirRL},
		"fields":     out,
	})
}

// apiIssue là một Issue ở dạng JSON. location là chuỗi hiển thị sẵn, loc là
// vị trí có cấu trúc để giao diện trỏ đúng dòng hay ô.
type apiIssue struct {
	Code     string            `json:"code"`
	Level    string            `json:"level"`
	Location string            `json:"location,omitempty"`
	Loc      model.Location    `json:"loc"`
	ID       string            `json:"id,omitempty"`
	Message  string            `json:"message"`
	Params   map[string]string `json:"params,omitempty"`
}

type apiFinding struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type apiResponse struct {
	OK       bool            `json:"ok"`
	Error    *apiIssue       `json:"error,omitempty"`
	Filename string          `json:"filename,omitempty"`
	Title    string          `json:"title,omitempty"`
	Source   string          `json:"source,omitempty"`
	Issues   []apiIssue      `json:"issues"`
	Warnings []string        `json:"warnings,omitempty"`
	Findings []apiFinding    `json:"findings,omitempty"`
	Stats    *flowcast.Stats `json:"stats,omitempty"`
	Drawio   string          `json:"drawio,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func fail(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apiResponse{Error: &apiIssue{Code: code, Level: model.LevelError, Message: msg},
		Issues: []apiIssue{}})
}

// statusOf chọn mã HTTP cho lỗi dừng việc dựng: vượt giới hạn là 413, quá thời
// gian là 503, còn lại là file người dùng gửi không đọc được.
func statusOf(code string) int {
	switch {
	case code == "limit.timeout":
		return http.StatusServiceUnavailable
	case strings.HasPrefix(code, "limit."):
		return http.StatusRequestEntityTooLarge
	case code == "schema.out_of_range" || code == "config.direction" || code == "merge.direction":
		return http.StatusBadRequest
	}
	return http.StatusUnprocessableEntity
}

func (s *server) acquire(ctx context.Context) bool {
	t := time.NewTimer(s.wait)
	defer t.Stop()
	select {
	case s.slot <- struct{}{}:
		return true
	case <-t.C:
	case <-ctx.Done():
	}
	return false
}

func (s *server) handle(w http.ResponseWriter, r *http.Request, build bool) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(s.lim.MaxBytes)+multipartSlack)
	file, hdr, err := r.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			fail(w, http.StatusRequestEntityTooLarge, "limit.bytes", "file quá giới hạn "+itoa(s.lim.MaxBytes)+" byte")
			return
		}
		fail(w, http.StatusBadRequest, "request.file", "thiếu file: gửi multipart/form-data với trường file")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		fail(w, http.StatusRequestEntityTooLarge, "limit.bytes", "file quá giới hạn "+itoa(s.lim.MaxBytes)+" byte")
		return
	}
	// Chỉ lấy phần tên: tên file đến từ trình duyệt của người lạ.
	name := path.Base(strings.ReplaceAll(hdr.Filename, "\\", "/"))
	src := flowcast.Source{Name: name, Data: data, Options: map[string]string{}}
	for _, k := range []string{"sheet", "delimiter", "encoding"} {
		if v := r.FormValue(k); v != "" {
			src.Options[k] = v
		}
	}
	if !s.acquire(r.Context()) {
		fail(w, http.StatusServiceUnavailable, "server.busy", "máy chủ đang bận, thử lại sau ít giây")
		return
	}
	defer func() { <-s.slot }()

	cfg := layout.DefaultConfig()
	for _, f := range layout.Fields() {
		v := strings.TrimSpace(r.FormValue(f.Name))
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			fail(w, http.StatusBadRequest, "schema.bad_value", f.Name+" cần một số nguyên, nhận "+v)
			return
		}
		*f.Get(&cfg) = n
	}
	opt := flowcast.Options{Title: strings.TrimSpace(r.FormValue("title")), Config: &cfg, Limits: &s.lim,
		Direction: strings.TrimSpace(r.FormValue("direction"))}
	var res flowcast.Result
	if build {
		res, err = flowcast.Build(src, opt)
	} else {
		res, err = flowcast.Check(src, opt)
	}
	if err != nil {
		var me *model.Error
		if errors.As(err, &me) {
			fail(w, statusOf(me.Code), me.Code, me.Msg)
			return
		}
		s.log.Error("build", "err", err)
		fail(w, http.StatusInternalServerError, "server.internal", "lỗi máy chủ")
		return
	}

	out := strings.TrimSuffix(name, source.Ext(name)) + ".drawio"
	resp := apiResponse{OK: !res.HasErrors(), Title: res.Title, Source: res.Source, Issues: []apiIssue{}}
	for _, i := range res.Issues {
		resp.Issues = append(resp.Issues, apiIssue{i.Code, i.Level, i.Loc.String(), i.Loc, i.ID, i.Msg, i.Params})
	}
	status := http.StatusOK
	if !resp.OK {
		status = http.StatusUnprocessableEntity
	}
	if build && resp.OK {
		resp.Filename = out
		resp.Warnings = res.Warnings
		for _, f := range res.Findings {
			resp.Findings = append(resp.Findings, apiFinding{f.Level, f.Msg})
		}
		resp.Stats = &res.Stats
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Type", "application/vnd.jgraph.mxfile; charset=utf-8")
			w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": out}))
			io.WriteString(w, res.Text)
			return
		}
		resp.Drawio = res.Text
	}
	writeJSON(w, status, resp)
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
