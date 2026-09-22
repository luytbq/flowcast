# Ràng buộc thứ tự chân nối theo chiều xuôi

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| C | lane | | Lane C | |
| C-1 | start | C | C-1 x | |
| E1 | edge | |  | from=C-1; to=B-3 |
| E2 | edge | |  | from=C-1; to=A-4 |
| E3 | edge | |  | from=C-1; to=B-2 |
| B-2 | task | B | B-2 x | |
| B-2.0 | db | B | d0 | attach=B-2 |
| E4 | edge | |  | from=B-2; to=A-4 |
| B-3 | external | B | B-3 x | |
| A-4 | end | A | A-4 x | |
