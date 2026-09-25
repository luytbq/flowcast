# A segment 12 pixels long already takes a label

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | System A<br>payments<br>internal | |
| B | lane | | Long business processing dept B | |
| C | lane | | Lane C<br>line two | |
| A-1 | start | A | A-1 x | |
| E1 | edge | | Yes | from=A-1; to=C-2 |
| C-2 | task | C | C-2 x | |
| E2 | edge | | send request | from=C-2; to=B-4 |
| E3 | edge | | Yes | from=C-2; to=B-3 |
| B-3 | end | B | B-3 x | |
| B-4 | task | B | B-4 x | |
