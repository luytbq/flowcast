# Dashed, bold, arrowless and highlighted edges

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start | | Enter | |
| E1 | edge | | dash | from=A-1; to=A-2; style=dashed |
| A-2 | task | | Bold step | style=highlight |
| E2 | edge | | bold, marked | from=A-2; to=A-3; style=bold,highlight |
| A-3 | task | | Plain step | |
| E3 | edge | | no arrowhead | from=A-3; to=A-4; style=noarrow,dashed |
| A-4 | end | | Done | |
