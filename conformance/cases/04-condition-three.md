# Condition with three branches, takes no horizontal arrow

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | What type? | |
| E2 | edge | | One | from=A-2; to=A-3 |
| E3 | edge | | Two | from=A-2; to=A-4 |
| E4 | edge | | Three | from=A-2; to=A-5 |
| A-3 | end | A | Type one | |
| A-4 | end | A | Type two | |
| A-5 | end | A | Type three | |
