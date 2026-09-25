# Branch depth ignores paths through a loop edge

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start | | Enter | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | | Fork? | |
| E2 | edge | | short | from=A-2; to=A-3 |
| E3 | edge | | long | from=A-2; to=A-5 |
| A-3 | task | | Retry | |
| E4 | edge | | | from=A-3; to=A-2; back=true |
| A-5 | task | | Step one | |
| E5 | edge | | | from=A-5; to=A-6 |
| A-6 | task | | Step two | |
| E6 | edge | | | from=A-6; to=A-7 |
| A-7 | end | | Done | |
