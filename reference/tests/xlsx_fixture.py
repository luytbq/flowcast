"""Sinh file .xlsx tối thiểu cho test, không cần thư viện ngoài.

Mỗi ô là một tuple (value, kind): kind 'inline' (inlineStr), 'shared' (sharedStrings),
'num' (số), 'formula' (công thức có giá trị lưu sẵn), 'formula_empty' (chưa có giá trị).
Chuỗi trần được hiểu là ('...', 'inline').
"""
import html
import zipfile

MAIN = 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'
REL = 'http://schemas.openxmlformats.org/officeDocument/2006/relationships'
PKG = 'http://schemas.openxmlformats.org/package/2006/relationships'


def col_name(i):
    s = ''
    i += 1
    while i:
        i, r = divmod(i - 1, 26)
        s = chr(65 + r) + s
    return s


def _cell(ref, value, shared):
    if not isinstance(value, tuple):
        value = (value, 'inline')
    val, kind = value
    if val == '' and kind != 'formula_empty':
        return ''
    if kind == 'inline':
        return f'<c r="{ref}" t="inlineStr"><is><t xml:space="preserve">{html.escape(str(val), quote=True)}</t></is></c>'
    if kind == 'shared':
        if val not in shared:
            shared.append(val)
        return f'<c r="{ref}" t="s"><v>{shared.index(val)}</v></c>'
    if kind == 'num':
        return f'<c r="{ref}"><v>{val}</v></c>'
    if kind == 'formula':
        return f'<c r="{ref}" t="str"><f>A1</f><v>{html.escape(str(val), quote=True)}</v></c>'
    if kind == 'formula_empty':
        return f'<c r="{ref}"><f>A1</f></c>'
    raise ValueError(kind)


def sheet_xml(rows, shared, hidden=(), merges=()):
    out = []
    for r, cells in rows:
        body = ''.join(_cell(f'{col_name(i)}{r}', v, shared) for i, v in enumerate(cells))
        hid = ' hidden="1"' if r in hidden else ''
        out.append(f'<row r="{r}"{hid}>{body}</row>')
    merged = ''
    if merges:
        merged = f'<mergeCells count="{len(merges)}">' + ''.join(f'<mergeCell ref="{m}"/>' for m in merges) + '</mergeCells>'
    return f'<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="{MAIN}"><sheetData>{"".join(out)}</sheetData>{merged}</worksheet>'


def write_xlsx(path, sheets):
    """sheets: list of (name, rows, hidden, merges); rows là list (row_number, [cell...])."""
    shared = []
    parts = []
    for i, (name, rows, hidden, merges) in enumerate(sheets, start=1):
        parts.append((name, i, sheet_xml(rows, shared, hidden, merges)))
    wb_sheets = ''.join(f'<sheet name="{html.escape(n, quote=True)}" sheetId="{i}" r:id="rId{i}"/>' for n, i, _ in parts)
    wb = (f'<?xml version="1.0"?><workbook xmlns="{MAIN}" xmlns:r="{REL}"><sheets>{wb_sheets}</sheets></workbook>')
    rels = (f'<?xml version="1.0"?><Relationships xmlns="{PKG}">'
            + ''.join(f'<Relationship Id="rId{i}" Type="x/worksheet" Target="worksheets/sheet{i}.xml"/>'
                      for _, i, _ in parts) + '</Relationships>')
    ss = ''
    if shared:
        items = ''.join(f'<si><t xml:space="preserve">{html.escape(s, quote=True)}</t></si>' for s in shared)
        ss = f'<?xml version="1.0"?><sst xmlns="{MAIN}" count="{len(shared)}" uniqueCount="{len(shared)}">{items}</sst>'
    with zipfile.ZipFile(path, 'w') as z:
        z.writestr('[Content_Types].xml', '<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>')
        z.writestr('xl/workbook.xml', wb)
        z.writestr('xl/_rels/workbook.xml.rels', rels)
        for _, i, xml in parts:
            z.writestr(f'xl/worksheets/sheet{i}.xml', xml)
        if ss:
            z.writestr('xl/sharedStrings.xml', ss)
