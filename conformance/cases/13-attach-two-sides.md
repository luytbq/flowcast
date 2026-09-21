# Node có cạnh hai bên, db phải dạt ra

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| B-1 | start | B | Vào | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Rẽ? | |
| B-3 | db | B | DB.LOG | attach=B-2 |
| E2 | edge | | trái | from=B-2; to=A-1 |
| E3 | edge | | phải | from=B-2; to=C-1 |
| A-1 | end | A | Nhánh trái | |
| C-1 | end | C | Nhánh phải | |
