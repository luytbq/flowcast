// Lệnh flowcastd là dịch vụ web: người dùng tải bảng lên và nhận về file
// .drawio.
//
//	flowcastd [-addr :8080] [-concurrent 4]
//
// POST /api/build nhận multipart/form-data với trường file, cùng các trường tùy
// chọn title, sheet, delimiter và encoding. Trả JSON gồm issue, cảnh báo và nội
// dung .drawio; thêm ?download=1 thì trả thẳng file khi dựng được. POST
// /api/check chỉ kiểm tra bảng. GET / là trang upload.
//
// Dịch vụ dùng hồ sơ giới hạn WebLimits, không merge và không gọi drawio: nó
// không đọc ghi file và không chạy tiến trình ngoài.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/luytbq/flowcast"
)

func main() {
	addr := flag.String("addr", ":8080", "địa chỉ lắng nghe")
	concurrent := flag.Int("concurrent", runtime.NumCPU(), "số lần dựng chạy cùng lúc")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           newServer(flowcast.WebLimits, *concurrent, log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(shut)
	}()
	log.Info("flowcastd lắng nghe", "addr", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("dừng", "err", err)
		os.Exit(1)
	}
}
