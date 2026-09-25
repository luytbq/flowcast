# Design: accepting csv and xlsx input

Status: implemented (read_markdown, read_csv, read_xlsx, table_from_grid in flowtable2drawio.py) and tested in tests/test_inputs.py. Usage: README.md, section Input formats.

## Problem

A Flow Table can currently only be written in a markdown file. The person writing the table may be more used to Excel, or receive tables from others as .csv or .xlsx. The tool needs to build diagrams directly from those files.

## Settled decisions

### 1. csv and xlsx are sources on par with md

- check and build accept .md, .csv, .xlsx directly; the format is inferred from the file extension.
- No intermediate file is generated and there is no convert command, so there are never two copies that drift apart.
- The target file still just changes extension: flow.xlsx --> flow.drawio.
- Reading only. No writing to .csv or .xlsx.

### 2. Finding the table and getting the title

- The header row is the first row, scanning from the top, that has all 5 cells id, type, parent, content, metadata. Case-insensitive, extra whitespace ignored.
- All rows above the header are skipped, so the top of the file may contain a title, notes, the author's name.
- The table ends at the first completely empty row after the header; notes can still be written below it.
- xlsx: take the first sheet that has a header row. The --sheet flag picks a sheet by name.
- Diagram title: --title first, then the first non-empty cell above the header row, finally the file name without extension.

### 3. Cell content conventions in csv/xlsx

- Line breaks: accept both real line breaks inside a cell and the br tag, so a table pasted over from md still works.
- The pipe is an ordinary character.
- No markdown-style escape handling: a backslash is a literal character. A backslash followed by a pipe triggers a warning, because it is almost certainly pasted from md without cleanup.
- Numeric cells: read by displayed value, trimming a superfluous .0. For id, from, to, attach cells of numeric type, warn, because an id like 4.10 gets turned into 4.1 by Excel and breaks references. The workaround: format that column as Text.
- Trim leading and trailing whitespace from every cell.

### 4. csv: guess the delimiter and encoding, with override flags

- Delimiter: try comma, semicolon, tab; pick whichever yields a header row with all 5 cells. If none does, report an error suggesting --delimiter.
- Encoding: try utf-8 with BOM, utf-8, then cp1252. Falling back to cp1252 prints a warning, because it usually means a wrongly saved file and the Vietnamese text may already be corrupted.
- The --delimiter and --encoding flags override.
- Quoting follows the csv standard (double quotes wrap a cell, a doubled quote is one quote), exactly as Excel exports, so line breaks inside cells can be read.

### 5. Reading xlsx files

- Formula cells: take the computed value stored in the file, do not compute. With no stored value, treat the cell as empty and warn with the cell address.
- Extra columns to the right of the table: ignored, so private columns such as notes or owner can be kept.
- Merged cells: the value belongs to the top-left cell, the other cells are empty. A merged cell inside the table area triggers a warning, because it easily makes the table end early or lose content.
- Hidden rows and columns: still read normally; a hidden row inside the table area triggers a warning. The table is the diagram's only source, so whatever is still in the table must appear in the picture.
- Print the name of the sheet used in the report.

## Implementation notes

- Read xlsx with the standard library's zipfile and ElementTree: xl/workbook.xml for the sheet list, xl/sharedStrings.xml for shared strings, xl/worksheets/sheetN.xml for cells. Cell types s (shared string), inlineStr, n, b and formula cells with a v tag must be handled.
- Error and warning messages must point to the exact place in the source file: md uses line numbers, csv uses line numbers, xlsx uses the sheet name plus the cell address (e.g. Sheet1!C12).
