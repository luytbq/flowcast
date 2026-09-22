# Hướng nhánh dừng ở node hợp nhánh

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | A | |
| B | lane | | B | |
| C | lane | | C | |
| B-1 | start | B | Vào | |
| E1 | edge | | | from=B-1; to=B-2 |
| B-2 | condition | B | Rẽ? | |
| E2 | edge | | phụ | from=B-2; to=B-3 |
| E3 | edge | | chính | from=B-2; to=B-5 |
| B-3 | task | B | Nhánh phụ | |
| E4 | edge | | | from=B-3; to=B-4 |
| B-5 | task | B | Nhánh chính | |
| E5 | edge | | | from=B-5; to=B-4 |
| B-4 | task | B | Hợp | |
| E6 | edge | | | from=B-4; to=A-1 |
| A-1 | end | A | Ra trái | |
