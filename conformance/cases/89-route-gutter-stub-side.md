# Chân nối vào máng bên đứng đúng phía, để hai đoạn cùng vị trí không đè nhau

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A | lane | | Lane A | |
| A-2 | condition | A | A-2 x | |
| E2 | edge | |  | from=A-2; to=A-4 |
| E3 | edge | | không | from=A-2; to=A-5 |
| A-3 | task | A | A-3 x | |
| E4 | edge | |  | from=A-3; to=A-6 |
| A-4 | condition | A | A-4 x | |
| E5 | edge | |  | from=A-4; to=A-5 |
| E6 | edge | |  | from=A-4; to=A-6 |
| A-5 | task | A | A-5 x | |
| A-6 | end | A | A-6 x | |
