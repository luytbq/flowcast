# Nhãn không bị phạt vì dây của chính cạnh

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A<br>dòng hai | |
| B | lane | | Lane B | |
| A-1 | start | A | A-1 x | |
| E1 | edge | |  | from=A-1; to=B-2 |
| E2 | edge | | Không | from=A-1; to=B-3 |
| B-2 | task | B | B-2 x | |
| B-2.0 | db | B | d0 | attach=B-2 |
| E3 | edge | | ok | from=B-2; to=B-3 |
| E4 | edge | | gửi yêu cầu | from=B-2; to=B-3 |
| B-3 | task | B | B-3 x | |
