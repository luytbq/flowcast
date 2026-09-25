# Condition with one outgoing edge

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Fork? | |
| E2 | edge | | Yes | from=A-2; to=A-3 |
| A-3 | end | A | Done | |
