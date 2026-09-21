# Ra mặt bên rồi xuống đỉnh node đích

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Bước một | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | task | A | Bước hai | |
| E3 | edge | | chuyển | from=A-3; to=B-1 |
| B-1 | task | B | Bước ba | |
| E4 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Xong? | |
| E5 | edge | | Chưa | from=B-2; to=B-1; back=true |
| E6 | edge | | Rồi | from=B-2; to=B-3 |
| B-3 | end | B | Kết thúc | |
