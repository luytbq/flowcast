# Reconcile internal payment gateway transactions with the partner bank over the "card & wallet" e-channel

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Accounting & recon | |
| A-1 | start | A | Receive "recon" file | |
| E1 | edge | | send & wait | from=A-1; to=A-2 |
| A-2 | task | A | Column	split by tab | |
| E2 | edge | | | from=A-2; to=A-3 |
| A-3 | end | A | Done | |
