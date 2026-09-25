# Long text needs word wrap and hard breaks

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Start | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Reconcile transactions with the payment gateway then record results in the ledger | |
| A-3 | db | A | DBPAYMENTRECONCILEFOREIGNTRANSFER | attach=A-2 |
| E2 | edge | | | from=A-2; to=A-4 |
| A-4 | task | A | https://api.example.com/v1/transactions/reconcile?from=2026-01-01 | |
| E3 | edge | | | from=A-4; to=A-5 |
| A-5 | end | A | Done | |
