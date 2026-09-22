# Cạnh back không ra mặt đáy

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | A-1 x | |
| A-3 | task | A | A-3 x | |
| A-3.0 | db | A | d0 | attach=A-3 |
| E2 | edge | |  | from=A-3; to=A-1; back=true |
