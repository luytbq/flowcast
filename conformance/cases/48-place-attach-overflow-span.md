# db tràn ra ô chỉ bị mũi tên ngang chặn

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | Vào từ lane phải | |
| E1 | edge | | | from=B-1; to=A-2 |
| A-2 | task | A | Nhận | |
| D1 | db | A | DB.MOT | attach=A-2 |
| D2 | db | A | DB.HAI | attach=A-2 |
| D3 | db | A | DB.BA | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Xong | |
