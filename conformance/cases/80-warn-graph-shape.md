# Cảnh báo về hình dạng đồ thị

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Rẽ? | |
| E2 | edge | | | from=A-2; to=A-3 |
| E3 | edge | | Có | from=A-2; to=A-4 |
| A-3 | end | A | Kết thúc sớm | |
| E4 | edge | | quay ra | from=A-3; to=A-4 |
| A-4 | end | A | Kết thúc | |
| E5 | edge | | vòng về start | from=A-4; to=A-1 |
