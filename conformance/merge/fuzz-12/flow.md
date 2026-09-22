# Chữ L cần ô rẽ góc chưa có dây ngang

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| B | lane | | Lane B | |
| D | lane | | Lane D | |
| D-1 | start | D | D-1 x | |
| E1 | edge | |  | from=D-1; to=D-3 |
| A-2 | task | A | A-2 x kèm thêm chữ cho dài ra |  |
| E2 | edge | |  | from=A-2; to=B-4 |
| D-3 | task | D | D-3 x kèm thêm chữ cho dài ra |  |
| E3 | edge | |  | from=D-3; to=B-4 |
| B-4 | task | B | B-4 x | |
| B-90 | task | B | Mới 0 | |
| E90 | edge | | | from=B-90; to=A-2 |
