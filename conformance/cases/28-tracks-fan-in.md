# Five long edges to different targets cross one gutter

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | One? | |
| E2 | edge | | Yes | from=A-2; to=B-5 |
| E3 | edge | | No | from=A-2; to=A-3 |
| A-3 | condition | A | Two? | |
| E4 | edge | | Yes | from=A-3; to=B-4 |
| E5 | edge | | No | from=A-3; to=A-4 |
| A-4 | condition | A | Three? | |
| E6 | edge | | Yes | from=A-4; to=B-3 |
| E7 | edge | | No | from=A-4; to=A-5 |
| A-5 | condition | A | Four? | |
| E8 | edge | | Yes | from=A-5; to=B-2 |
| E9 | edge | | No | from=A-5; to=A-6 |
| A-6 | task | A | All pass | |
| E10 | edge | | | from=A-6; to=B-1 |
| B-1 | task | B | Take | |
| E11 | edge | | | from=B-1; to=B-2 |
| B-2 | task | B | Join four | |
| E12 | edge | | | from=B-2; to=B-3 |
| B-3 | task | B | Join three | |
| E13 | edge | | | from=B-3; to=B-4 |
| B-4 | task | B | Join two | |
| E14 | edge | | | from=B-4; to=B-5 |
| B-5 | end | B | Join one | |
