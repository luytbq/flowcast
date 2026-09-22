# Luồng gọi lại có giới hạn

| id | type | parent | content | metadata |
|---|---|---|---|---|
| CLI | lane | | Client | |
| GW | lane | | Gateway | |
| BANK | lane | | Bank | |
| CLI-1 | start | CLI | Gửi yêu cầu | |
| E1 | edge | | POST /pay | from=CLI-1; to=GW-1 |
| GW-1 | task | GW | Gọi bank | |
| GW-2 | db | GW | DB.ATTEMPT | attach=GW-1 |
| E2 | edge | | | from=GW-1; to=BANK-1 |
| BANK-1 | task | BANK | Xử lý | |
| E3 | edge | | | from=BANK-1; to=GW-3 |
| GW-3 | condition | GW | Kết quả bank | style=highlight |
| E4 | edge | | timeout | from=GW-3; to=GW-4 |
| E5 | edge | | lỗi \| từ chối | from=GW-3; to=GW-5 |
| E6 | edge | | thành công | from=GW-3; to=GW-6 |
| GW-4 | task | GW | Tăng số lần thử | |
| E7 | edge | | | from=GW-4; to=GW-1; back=true; style=dashed |
| GW-5 | task | GW | Trả lỗi | |
| E8 | edge | | | from=GW-5; to=CLI-2 |
| GW-6 | task | GW | Trả kết quả | |
| E9 | edge | | | from=GW-6; to=CLI-2 |
| CLI-2 | end | CLI | Nhận kết quả | |
| GW-9 | text | GW | Phần còn lại | |
| BANK-2 | external | BANK | Đối soát cuối ngày | |
