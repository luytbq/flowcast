# Order flow

| id | type | parent | content | metadata |
|---|---|---|---|---|
| USR | lane | | User | |
| API | lane | | API | |
| USR-1 | start | USR | Submit order request | |
| E1 | edge | | POST /orders | from=USR-1; to=API-1 |
| API-1 | task | API | Validate data | |
| API-1.1 | text | API | Checks format only, not stock levels yet | attach=API-1 |
| E2 | edge | | | from=API-1; to=API-2 |
| API-2 | condition | API | Data valid? | style=highlight |
| E3 | edge | | No | from=API-2; to=API-3 |
| E4 | edge | | Yes | from=API-2; to=USR-2 |
| API-3 | task | API | Return 400 | |
| E5 | edge | | | from=API-3; to=USR-2 |
| USR-2 | end | USR | Get result | |
