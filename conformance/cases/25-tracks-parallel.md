# Nhiều cạnh khác đích chạy song song trong một máng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| B-1 | start | B | Vào | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Chặn một? | |
| E2 | edge | | Có | from=B-2; to=A-1 |
| E3 | edge | | Không | from=B-2; to=B-3 |
| A-1 | end | A | Lỗi một | |
| B-3 | condition | B | Chặn hai? | |
| E4 | edge | | Có | from=B-3; to=A-2 |
| E5 | edge | | Không | from=B-3; to=B-4 |
| A-2 | end | A | Lỗi hai | |
| B-4 | condition | B | Chặn ba? | |
| E6 | edge | | Có | from=B-4; to=A-3 |
| E7 | edge | | Không | from=B-4; to=B-5 |
| A-3 | end | A | Lỗi ba | |
| B-5 | end | B | Qua hết | |
