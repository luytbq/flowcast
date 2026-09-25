# A db without attach stands on its own in the flow

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-3 |
| A-3 | end | A | Done | |
| A-2 | db | A | DB.ORDER | |
