# Bad parent and references

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | A | A lane must not have parent | |
| B | lane | | Lane B | |
| A-1 | task | | Missing parent | |
| E1 | edge | A | Edges must not have parent | from=A-1 |
| E2 | edge | | | from=A; to=A-1 |
| A-2 | db | A | DB.X | attach=KHONGCO |
| A-3 | text | A | A note | attach=B |
