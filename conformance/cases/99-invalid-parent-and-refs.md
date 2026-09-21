# Parent và tham chiếu sai

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | A | Lane không được có parent | |
| B | lane | | Lane B | |
| A-1 | task | | Thiếu parent | |
| E1 | edge | A | Edge không được có parent | from=A-1 |
| E2 | edge | | | from=A; to=A-1 |
| A-2 | db | A | DB.X | attach=KHONGCO |
| A-3 | text | A | Ghi chú | attach=B |
