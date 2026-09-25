# Side branch leads to the left lane

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | Enter | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Error? | |
| E2 | edge | | Yes | from=B-2; to=B-3 |
| E3 | edge | | No | from=B-2; to=B-4 |
| B-3 | task | B | Build error message | |
| E4 | edge | | return err | from=B-3; to=A-1 |
| A-1 | end | A | Get error | |
| B-4 | end | B | Continue | |
