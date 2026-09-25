# Line breaks, escaped pipes, accented letters

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Start | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | task | A | Line one<br>Line two \| résumé \| café | |
| E2 | edge | | label \<with\> tags | from=A-2; to=A-3 |
| A-3 | end | A | End the recon session | |
