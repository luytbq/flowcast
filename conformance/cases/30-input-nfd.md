# Input in NFD form, accents split from letters

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Café order desk | |
| A-1 | start | A | Take the naïve user request | |
| E1 | edge | | authenticated | from=A-1; to=A-2 |
| A-2 | task | A | Check stock and reserve items | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Return result | |
