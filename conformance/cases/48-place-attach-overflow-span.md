# db overflows into a cell blocked only by a horizontal arrow

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | In from right lane | |
| E1 | edge | | | from=B-1; to=A-2 |
| A-2 | task | A | Take | |
| D1 | db | A | DB.ONE | attach=A-2 |
| D2 | db | A | DB.TWO | attach=A-2 |
| D3 | db | A | DB.THREE | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Done | |
