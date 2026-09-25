# Retry flow with a cap

| id | type | parent | content | metadata |
|---|---|---|---|---|
| CLI | lane | | Client | |
| GW | lane | | Gateway | |
| BANK | lane | | Bank | |
| CLI-1 | start | CLI | Send request | |
| E1 | edge | | POST /pay | from=CLI-1; to=GW-1 |
| GW-1 | task | GW | Call bank | |
| GW-2 | db | GW | DB.ATTEMPT | attach=GW-1 |
| E2 | edge | | | from=GW-1; to=BANK-1 |
| BANK-1 | task | BANK | Process | |
| E3 | edge | | | from=BANK-1; to=GW-3 |
| GW-3 | condition | GW | Bank result | style=highlight |
| E4 | edge | | timeout | from=GW-3; to=GW-4 |
| E5 | edge | | error \| reject | from=GW-3; to=GW-5 |
| E6 | edge | | success | from=GW-3; to=GW-6 |
| GW-4 | task | GW | Increment attempts | |
| E7 | edge | | | from=GW-4; to=GW-1; back=true; style=dashed |
| GW-5 | task | GW | Return error | |
| E8 | edge | | | from=GW-5; to=CLI-2 |
| GW-6 | task | GW | Return result | |
| E9 | edge | | | from=GW-6; to=CLI-2 |
| CLI-2 | end | CLI | Get result | |
| GW-9 | text | GW | Rest of the table | |
| BANK-2 | external | BANK | End-of-day reconcile | |
