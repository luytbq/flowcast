// Package data nhúng các bảng dữ liệu dùng chung vào binary.
//
// Bảng số đo font nằm ở đây chứ không đọc từ đường dẫn lúc chạy, để flowcast
// là một file chạy được ở bất kỳ đâu.
package data

import _ "embed"

// Verdana là bảng advance width của Verdana, do tools/extractmetrics sinh.
//
//go:embed verdana.json
var Verdana []byte
