# Condition hai nhánh

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Hợp lệ? | |
| E2 | edge | | Không | from=A-2; to=A-3 |
| E3 | edge | | Có | from=A-2; to=A-4 |
| A-3 | end | A | Từ chối | |
| A-4 | end | A | Chấp nhận | |
