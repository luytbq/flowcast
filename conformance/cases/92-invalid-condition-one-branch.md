# Condition chỉ có một cạnh ra

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Rẽ? | |
| E2 | edge | | Có | from=A-2; to=A-3 |
| A-3 | end | A | Xong | |
