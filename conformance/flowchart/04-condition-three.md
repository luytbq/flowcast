# Condition ba nhánh, không nhận mũi tên ngang

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Vào |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition |  | Loại nào? |  |
| E2 | edge | | Một | from=A-2; to=A-3 |
| E3 | edge | | Hai | from=A-2; to=A-4 |
| E4 | edge | | Ba | from=A-2; to=A-5 |
| A-3 | end |  | Loại một |  |
| A-4 | end |  | Loại hai |  |
| A-5 | end |  | Loại ba |  |
