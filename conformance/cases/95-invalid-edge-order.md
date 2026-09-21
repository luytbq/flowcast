# Cạnh ra không nằm liền sau node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| A-2 | task | A | Xen vào giữa | |
| E1 | edge | | | from=A-1; to=A-2 |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Xong | |
