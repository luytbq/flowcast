# db spills into a cell blocked only by an arrow

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| N1 | lane | | New lane 1 | |
| B | lane | | Lane B | |
| B-1 | start | B | In from right lane | |
| E1 | edge | | | from=B-1; to=A-2 |
| A-2 | task | N1 | Receive |  |
| D1 | db | A | DB.ONE | attach=A-2 |
| D2 | db | A | DB.TWO | attach=A-2 |
| D3 | db | A | DB.THREE | attach=A-2 |

| B-92 | task | B | New 2 | |
| E92 | edge | | | from=A-2; to=B-92 |
