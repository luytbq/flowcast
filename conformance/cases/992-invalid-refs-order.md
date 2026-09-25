# from and to both broken, duplicate id out of order

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Target | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Done | |
| E9 | edge | | | from=KHONG1; to=KHONG2 |
| A-1 | task | A | Duplicate id A-1, after A-2 | |
