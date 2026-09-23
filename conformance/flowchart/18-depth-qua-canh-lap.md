# Độ sâu nhánh không tính đường đi qua cạnh vòng lặp

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start | | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | | Rẽ? | |
| E2 | edge | | ngắn | from=A-2; to=A-3 |
| E3 | edge | | dài | from=A-2; to=A-5 |
| A-3 | task | | Thử lại | |
| E4 | edge | | | from=A-3; to=A-2; back=true |
| A-5 | task | | Bước một | |
| E5 | edge | | | from=A-5; to=A-6 |
| A-6 | task | | Bước hai | |
| E6 | edge | | | from=A-6; to=A-7 |
| A-7 | end | | Xong | |
