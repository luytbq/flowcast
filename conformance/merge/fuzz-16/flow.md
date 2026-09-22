# db tràn ra ô chỉ bị mũi tên ngang chặn

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| N1 | lane | | Lane mới 1 | |
| B | lane | | Lane B | |
| B-1 | start | B | Vào từ lane phải | |
| E1 | edge | | | from=B-1; to=A-2 |
| A-2 | task | N1 | Nhận |  |
| D1 | db | A | DB.MOT | attach=A-2 |
| D2 | db | A | DB.HAI | attach=A-2 |
| D3 | db | A | DB.BA | attach=A-2 |

| B-92 | task | B | Mới 2 | |
| E92 | edge | | | from=A-2; to=B-92 |
