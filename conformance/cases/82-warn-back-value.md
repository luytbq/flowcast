# back có giá trị lạ trên cạnh quay ngược

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Thử | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | condition | A | Lại? | |
| E3 | edge | | Có | from=A-3; to=A-2; back=maybe |
| E4 | edge | | Không | from=A-3; to=A-4 |
| A-4 | end | A | Xong | |
