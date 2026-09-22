# Đối soát giao dịch của cổng thanh toán nội bộ với ngân hàng đối tác qua kênh "thẻ & ví" điện tử

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Kế toán & đối soát | |
| A-1 | start | A | Nhận file "đối soát" | |
| E1 | edge | | gửi & chờ | from=A-1; to=A-2 |
| A-2 | task | A | Cột	tách bằng tab | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Xong | |
