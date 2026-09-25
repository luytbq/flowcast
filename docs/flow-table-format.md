# The Flow Table format

A Flow Table is a way of writing an activity diagram with swimlanes as a single markdown table. Reading the table from top to bottom follows the flow. A tool or another AI session can rebuild the diagram from the table without seeing the original image.

## What columns does the table have?

| Column | Meaning |
|---|---|
| id | Identifier of the row, never changes once assigned |
| type | Element type, see the next section |
| parent | Id of the lane holding the element. Left empty for lanes and edges |
| content | Displayed text: lane name, text in a node, label on an edge, note content |
| metadata | Extra information, as key=value, pairs separated by semicolons, no quotes needed |

In content, a line break is written as <br>, and a pipe character is written as \|, so the markdown table does not break. Text in content is kept verbatim as in the diagram, typos included.

## What element types are there?

The element type also determines the shape, so there is no separate metadata for shape.

| type | Shape | parent | Usable metadata |
|---|---|---|---|
| lane | Vertical lane | empty | none |
| start | Flow start point | lane | style |
| end | Flow end point | lane | style |
| task | Processing step box | lane | style |
| condition | Branching diamond | lane | style |
| db | Data table cylinder, stands next to a node, not connected by edges | lane | attach (required), style |
| external | Ellipse, points to another flow outside the diagram | lane | style |
| text | Note, no frame | lane | attach, style |
| edge | Arrow connecting two elements | empty | from, to (required), style, back |

Meaning of each key:

- **from, to:** ids of the elements at the start and end of the arrow. An edge can connect two different lanes, so an edge belongs to no lane.
- **attach:** id of the node that the data table or note stands next to.
- **style:** highlight if the element is filled with an accent color in the diagram. Edges additionally accept dashed for a dashed line, bold for a thick line, and noarrow for no arrowhead. Multiple values are separated by commas, for example style=dashed,noarrow.
- **back:** write back=true for an edge that goes back to an element already passed, that is, an edge that forms a loop.

## How are ids assigned?

- **Lane:** a short uppercase code, for example USR, API, SVC.
- **Elements in a lane** (every type except lane and edge): the lane code, a hyphen, then a number, for example API-1, SVC-3.
- **Edge:** the letter E followed by a number, for example E1, E12.

Numbers in ids increase in order of appearance when the table is first written. After that ids are never renumbered. To insert an element between two existing elements, add a decimal suffix to the id of the preceding one:

- Insert an edge after E4, before E5: E4.1. Insert another after E4.1: E4.2.
- Insert between E4 and E4.1: E4.0.1.
- Insert a task after API-3 in lane API: API-3.1.

Ids are compared segment by segment numerically, so E4 comes before E4.1, E4.1 comes before E4.2, and E4.9 comes before E4.10. This way the number in an id stays close to its position in the table, easy to find while scrolling the file, and the from, to, attach references do not break.

## In what order are the rows arranged?

All lanes come first in the table, in the diagram's left-to-right order.

After that the table follows the flow, starting from the start elements. Each time an element is written, write in turn:

1. The element itself.
2. The db and text elements attached to it (attach points to it).
3. All edges leaving it. For a condition, these are all the branches, written together right after the condition row.
4. The target elements of the edges just written, in the same order as the edges. Each target element repeats from step 1.

The order of outgoing edges is the author's choice. Put branches that end early, such as returning an error or diverting to another flow, first, so that the longest continuing branch comes last and the flow reads continuously.

A target element is skipped at step 4, not written at that point, in the following cases:

- **It is already in the table:** the edge only points to it by id, it is not written again.
- **It is a merge node, meaning it has several incoming edges, and some source element has not been written yet:** this node is written only right after its last source element has appeared. Edges with back=true do not count toward this condition.

Because of the merge rule, a shared target node such as "return the result to the user" naturally falls below every branch leading to it. The branches above only point to it through to=, and the reader knows that node sits further down.

Elements that cannot be reached from a start are gathered at the end of the table, after a text row whose content is "Phần còn lại".

## Example

The ordering flow below has a branching condition, a merge node, a data table, an inserted note and a dashed edge.

| id | type | parent | content | metadata |
|---|---|---|---|---|
| USR | lane | | Người dùng | |
| API | lane | | API | |
| SVC | lane | | Order Service | |
| USR-1 | start | USR | Gửi yêu cầu đặt hàng | |
| E1 | edge | | POST /orders | from=USR-1; to=API-1 |
| API-1 | task | API | Kiểm tra dữ liệu | |
| API-1.1 | text | API | Chỉ kiểm tra định dạng, chưa kiểm tồn kho | attach=API-1 |
| E2 | edge | | | from=API-1; to=API-2 |
| API-2 | condition | API | Dữ liệu hợp lệ? | style=highlight |
| E3 | edge | | No | from=API-2; to=API-3 |
| E4 | edge | | Yes | from=API-2; to=SVC-1 |
| API-3 | task | API | Trả lỗi 400 | |
| E5 | edge | | | from=API-3; to=USR-2 |
| SVC-1 | task | SVC | Tạo đơn hàng | |
| SVC-2 | db | SVC | DB.ORDER | attach=SVC-1 |
| E6 | edge | | | from=SVC-1; to=SVC-3 |
| SVC-3 | task | SVC | Trả kết quả | |
| E7 | edge | | | from=SVC-3; to=USR-2 |
| E7.1 | edge | | | from=SVC-3; to=SVC-4; style=dashed |
| USR-2 | end | USR | Nhận kết quả | |
| SVC-4 | end | SVC | Gửi email xác nhận | |

Reading this table:

- **E3 and E4 come right after API-2**, and only then the content of each branch in that order: the No branch (API-3) first, the Yes branch (SVC-1) after.
- **E5 points to USR-2 before USR-2 appears.** USR-2 has two sources, API-3 and SVC-3, so it can only be written after SVC-3.
- **API-1.1 and E7.1 are elements inserted after the first writing.** Their ids fall between the ids of the two neighboring elements.

## Checking whether a table is valid

Errors, must be fixed:

- Duplicate id.
- from, to or attach points to an id that does not exist.
- An edge missing from or to.
- A db missing attach.
- An element in a lane whose parent is empty or is not the id of a lane.
- A condition with fewer than 2 outgoing edges.
- An outgoing edge of an element that does not come right after that element (and after the db and text elements attached to it).

Warnings, should be reviewed:

- A task or condition with no outgoing edge.
- A start with an incoming edge, or an end with an outgoing edge.
- A merge node that appears before one of its source elements, where that edge is not marked back=true.
- An outgoing edge of a condition without a label. Allowed when the original diagram has no label, but it is better to add one.
