# Two label candidates of equal cost: the first one wins

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Long business processing dept A | |
| B | lane | | Lane B | |
| C | lane | | System C<br>payments<br>internal | |
| B-1 | start | B | B-1 x | |
| E1 | edge | | send request | from=B-1; to=B-2 |
| B-2 | external | B | B-2 x | |
