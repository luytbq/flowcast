# Five data tables attached to one node

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Reconcile | |
| D1 | db | A | DB.ONE | attach=A-2 |
| D2 | db | A | DB.TWO | attach=A-2 |
| D3 | db | A | DB.THREE | attach=A-2 |
| D4 | db | A | DB.FOUR | attach=A-2 |
| D5 | db | A | DB.FIVE | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Done | |
