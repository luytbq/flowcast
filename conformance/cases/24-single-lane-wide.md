# One lane, many branches, close to a flowchart

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Get request | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Found in the cache? | |
| E2 | edge | | No | from=A-2; to=A-3 |
| E3 | edge | | Yes | from=A-2; to=A-7 |
| A-3 | task | A | Query source | |
| E4 | edge | | | from=A-3; to=A-4 |
| A-4 | condition | A | Query failed? | |
| E5 | edge | | Yes | from=A-4; to=A-5 |
| E6 | edge | | No | from=A-4; to=A-6 |
| A-5 | end | A | Send error | |
| A-6 | task | A | Write to the cache | |
| E7 | edge | | | from=A-6; to=A-7 |
| A-7 | end | A | Return result | |
