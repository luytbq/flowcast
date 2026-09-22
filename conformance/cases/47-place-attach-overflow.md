# Năm bảng dữ liệu bám một node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Đối soát | |
| D1 | db | A | DB.MOT | attach=A-2 |
| D2 | db | A | DB.HAI | attach=A-2 |
| D3 | db | A | DB.BA | attach=A-2 |
| D4 | db | A | DB.BON | attach=A-2 |
| D5 | db | A | DB.NAM | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Xong | |
