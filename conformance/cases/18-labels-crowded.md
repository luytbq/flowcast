# Many labels compete for space

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | request to open session | from=A-1; to=B-1 |
| B-1 | condition | B | Session valid? | |
| E2 | edge | | session expired, renew | from=B-1; to=B-2 |
| E3 | edge | | session still valid | from=B-1; to=B-3 |
| B-2 | task | B | Issue session | |
| E4 | edge | | | from=B-2; to=B-3 |
| B-3 | task | B | Return data | |
| E5 | edge | | result with session id | from=B-3; to=A-2 |
| A-2 | end | A | Take | |
