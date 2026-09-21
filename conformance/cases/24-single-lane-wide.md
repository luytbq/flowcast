# Một lane, nhiều nhánh, gần với flowchart

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Nhận yêu cầu | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Có trong bộ nhớ đệm? | |
| E2 | edge | | Không | from=A-2; to=A-3 |
| E3 | edge | | Có | from=A-2; to=A-7 |
| A-3 | task | A | Truy vấn nguồn | |
| E4 | edge | | | from=A-3; to=A-4 |
| A-4 | condition | A | Truy vấn lỗi? | |
| E5 | edge | | Có | from=A-4; to=A-5 |
| E6 | edge | | Không | from=A-4; to=A-6 |
| A-5 | end | A | Trả lỗi | |
| A-6 | task | A | Ghi vào bộ nhớ đệm | |
| E7 | edge | | | from=A-6; to=A-7 |
| A-7 | end | A | Trả kết quả | |
