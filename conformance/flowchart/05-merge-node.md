# Hợp nhánh về cột của node rẽ

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start |  | Vào |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition |  | Rẽ? |  |
| E2 | edge | | Không | from=A-2; to=A-3 |
| E3 | edge | | Có | from=A-2; to=A-4 |
| A-3 | task |  | Nhánh một |  |
| E4 | edge | | | from=A-3; to=A-5 |
| A-4 | task |  | Nhánh hai |  |
| E5 | edge | | | from=A-4; to=A-5 |
| A-5 | end |  | Hợp nhánh |  |
