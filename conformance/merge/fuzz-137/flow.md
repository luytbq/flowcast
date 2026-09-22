# Nhánh phụ không rời lane, dồn phải rồi trái

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| N2 | lane | | Lane mới 2 | |
| N1 | lane | | Lane mới 1 | |
| A-1 | start | N1 | Vào |  |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Ba hướng? | |
| E2 | edge | | Một | from=A-2; to=A-3 |
| E3 | edge | | Hai | from=A-2; to=A-4 |
| E4 | edge | | Ba | from=A-2; to=A-5 |
| A-3 | task | A | Việc một |  |
| E5 | edge | | | from=A-3; to=A-6 |
| A-4 | task | N2 | Việc hai |  |
| E6 | edge | | | from=A-4; to=A-6 |
| A-5 | task | A | Việc ba | |
| E7 | edge | | | from=A-5; to=A-6 |
| A-6 | end | A | Gộp | |
