# back has an unknown value on a backward edge

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Try | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | condition | A | Again? | |
| E3 | edge | | Yes | from=A-3; to=A-2; back=maybe |
| E4 | edge | | No | from=A-3; to=A-4 |
| A-4 | end | A | Done | |
