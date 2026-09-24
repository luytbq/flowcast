// Package conformance giữ bộ case đầu vào, golden đầu ra và các test chạy trên
// toàn bộ bộ case: so golden, và các bất biến mà mọi sơ đồ phải thỏa.
//
// Golden do chính flowcast sinh bằng go test ./conformance -update. Đổi golden
// thì phải xem ảnh và đọc diff: một golden đổi im lặng là một hành vi đã thay
// đổi mà không ai xem.
package conformance

// Dir trả về thư mục bộ case, tính từ vị trí gói này.
func Dir() string { return "." }
