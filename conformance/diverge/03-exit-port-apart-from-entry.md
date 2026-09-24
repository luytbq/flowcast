# Dây ra tách khỏi dây vào cùng mặt

| id | type | parent | content | metadata |
|---|---|---|---|---|
| P | lane | | Lane P | |
| Q | lane | | Lane Q | |
| P-1 | start | P | Vào | |
| E1 | edge | | gửi | from=P-1; to=Q-1 |
| Q-1 | external | Q | Hệ thống ngoài | |
| Q-1.t | text | Q | ghi chú | attach=Q-1 |
| E2 | edge | | tiếp | from=Q-1; to=Q-2 |
| E3 | edge | | trả lỗi | from=Q-1; to=P-2 |
| Q-2 | end | Q | Xong | |
| P-2 | end | P | Lỗi | |
