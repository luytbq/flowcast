# Cạnh nét đứt, nét đậm, không mũi tên và tô nhấn

| id | type | parent | content | metadata |
|---|---|---|---|---|
| A-1 | start | | Vào | |
| E1 | edge | | đứt | from=A-1; to=A-2; style=dashed |
| A-2 | task | | Bước đậm | style=highlight |
| E2 | edge | | đậm và nhấn | from=A-2; to=A-3; style=bold,highlight |
| A-3 | task | | Bước thường | |
| E3 | edge | | không mũi tên | from=A-3; to=A-4; style=noarrow,dashed |
| A-4 | end | | Xong | |
