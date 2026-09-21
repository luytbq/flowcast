# Nhiều cạnh cùng đích dùng chung track

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-1 | start | A | Vào | |
| E1 | edge | | | from=A-1; to=A-2 |
| A-2 | condition | A | Loại? | |
| E2 | edge | | Một | from=A-2; to=A-3 |
| E3 | edge | | Hai | from=A-2; to=A-4 |
| E4 | edge | | Ba | from=A-2; to=A-5 |
| A-3 | task | A | Việc một | |
| E5 | edge | | | from=A-3; to=A-6 |
| A-4 | task | A | Việc hai | |
| E6 | edge | | | from=A-4; to=A-6 |
| A-5 | task | A | Việc ba | |
| E7 | edge | | | from=A-5; to=A-6 |
| A-6 | task | A | Gom lại | |
| E8 | edge | | | from=A-6; to=A-7 |
| A-7 | end | A | Xong | |
