# Order flow

| id | type | parent | content | metadata |
|---|---|---|---|---|
| USR | lane | | User | |
| API | lane | | API | |
| SVC | lane | | Order Service | |
| USR-1 | start | USR | Submit order request | |
| E1 | edge | | POST /orders | from=USR-1; to=API-1 |
| API-1 | task | API | Validate data | |
| API-1.1 | text | API | Checks format only, not stock levels yet | attach=API-1 |
| E2 | edge | | | from=API-1; to=API-2 |
| API-2 | condition | API | Data valid? | style=highlight |
| E3 | edge | | No | from=API-2; to=API-3 |
| E4 | edge | | Yes | from=API-2; to=SVC-1 |
| API-3 | task | API | Return 400 | |
| E5 | edge | | | from=API-3; to=USR-2 |
| SVC-1 | task | SVC | Create order | |
| SVC-2 | db | SVC | DB.ORDER | attach=SVC-1 |
| E6 | edge | | | from=SVC-1; to=SVC-3 |
| SVC-3 | task | SVC | Return result | |
| E7 | edge | | | from=SVC-3; to=USR-2 |
| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |
| USR-2 | end | USR | Get result | |
| SVC-4 | end | SVC | Send confirmation | |
