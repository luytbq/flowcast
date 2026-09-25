# Label avoids the pool header

| id | type | parent | content | metadata |
|---|---|---|---|---|
| C | lane | | Lane C | |
| D | lane | | Business processing unit D, long | |
| C-1 | start | C | C-1 x | |
| E1 | edge | | ok | from=C-1; to=C-3 |
| C-2 | task | C | C-2 x |  |
| E3 | edge | | send request | from=C-2; to=C-3 |
| E4 | edge | |  | from=C-2; to=C-3 |
| E5 | edge | | return result with detailed error code | from=C-2; to=C-1; back=true |
| C-3 | task | C | C-3 x | |
| E6 | edge | | No | from=C-3; to=C-2; back=true |
| C-90 | task | C | New 0 | |
| E90 | edge | | | from=C-90; to=C-2 |
