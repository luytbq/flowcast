# Straight flow in one lane

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Start | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Handle | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Done | |
