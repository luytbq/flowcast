# Đoạn cùng đích ưu tiên track đã có đoạn cùng đích

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | A-1 x | |
| E1 | edge | |  | from=A-1; to=A-3 |
| E2 | edge | |  | from=A-1; to=A-4 |
| A-2 | task | A | A-2 x | |
| E3 | edge | |  | from=A-2; to=A-3 |
| A-3 | task | A | A-3 x | |
| E4 | edge | |  | from=A-3; to=A-4 |
| E5 | edge | |  | from=A-3; to=A-4 |
| A-4 | task | A | A-4 x | |
