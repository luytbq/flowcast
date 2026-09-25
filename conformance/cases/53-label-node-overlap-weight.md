# A label on a node costs more than on a wire

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | System A<br>payments<br>internal | |
| A-3 | task | A | A-3 x | |
| A-3.0 | text | A | d0 | attach=A-3 |
| E3 | edge | | return result with error detail | from=A-3; to=A-4 |
| E4 | edge | |  | from=A-3; to=A-5 |
| E5 | edge | | return result with error detail | from=A-3; to=A-4 |
| A-4 | task | A | A-4 x | |
| A-4.0 | db | A | d0 | attach=A-4 |
| E6 | edge | |  | from=A-4; to=A-5 |
| A-5 | task | A | A-5 x | |
