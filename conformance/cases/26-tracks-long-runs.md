# Cạnh dài vượt nhiều hàng, khác đích, chồng máng

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Rẽ sớm? | |
| E2 | edge | | Có | from=A-2; to=B-3 |
| E3 | edge | | Không | from=A-2; to=A-3 |
| A-3 | condition | A | Rẽ giữa? | |
| E4 | edge | | Có | from=A-3; to=B-2 |
| E5 | edge | | Không | from=A-3; to=A-4 |
| A-4 | task | A | Đường dài | |
| E6 | edge | | | from=A-4; to=B-1 |
| B-1 | task | B | Nhận cuối | |
| E7 | edge | | | from=B-1; to=B-2 |
| B-2 | task | B | Nhận giữa | |
| E8 | edge | | | from=B-2; to=B-3 |
| B-3 | end | B | Kết thúc | |
