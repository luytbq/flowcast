# Condition hai nhánh phụ không nhận mũi tên ngang

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=B-1 |
| B-1 | condition | B | Ba hướng? | |
| E2 | edge | | x | from=B-1; to=B-2 |
| E3 | edge | | y | from=B-1; to=B-3 |
| E4 | edge | | z | from=B-1; to=B-4 |
| B-2 | end | B | X | |
| B-3 | end | B | Y | |
| B-4 | end | B | Z | |
