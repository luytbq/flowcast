# Năm cạnh dài khác đích cùng vượt một máng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Một? | |
| E2 | edge | | Có | from=A-2; to=B-5 |
| E3 | edge | | Không | from=A-2; to=A-3 |
| A-3 | condition | A | Hai? | |
| E4 | edge | | Có | from=A-3; to=B-4 |
| E5 | edge | | Không | from=A-3; to=A-4 |
| A-4 | condition | A | Ba? | |
| E6 | edge | | Có | from=A-4; to=B-3 |
| E7 | edge | | Không | from=A-4; to=A-5 |
| A-5 | condition | A | Bốn? | |
| E8 | edge | | Có | from=A-5; to=B-2 |
| E9 | edge | | Không | from=A-5; to=A-6 |
| A-6 | task | A | Qua hết |  |
| E10 | edge | | | from=A-6; to=B-1 |
| B-1 | task | B | Nhận | |
| E11 | edge | | | from=B-1; to=B-2 |
| B-2 | task | B | Gom bốn | |
| E12 | edge | | | from=B-2; to=B-3 |
| B-3 | task | B | Gom ba | |
| E13 | edge | | | from=B-3; to=B-4 |
| B-4 | task | B | Gom hai | |
| E14 | edge | | | from=B-4; to=B-5 |
| B-5 | end | B | Gom một | |
| B-90 | task | B | Mới 0 | |
| E90 | edge | | | from=B-90; to=A-6 |
