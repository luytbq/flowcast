// Package data nhúng các bảng dữ liệu dùng chung vào binary.
//
// Bảng số đo font nằm ở đây chứ không đọc từ đường dẫn lúc chạy, để flowcast
// là một file chạy được ở bất kỳ đâu. Bản tham chiếu Python đọc đúng file này.
package data

import _ "embed"

// Verdana là bảng advance width của Verdana, do tools/extract_metrics.py sinh.
//
//go:embed verdana.json
var Verdana []byte
