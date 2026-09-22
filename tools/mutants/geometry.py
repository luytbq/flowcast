"""Đột biến Python tương ứng với các đột biến geometry và labels trong mutate.sh.

Dùng với tools/fuzz_cases.py --stage geometry. Ba đột biến đã chứng minh là tương
đương vẫn để ở đây, vì chạy lại chúng là cách kiểm chứng minh đó còn đúng. Đột biến
pyMax không có mặt: max của Python có ngữ nghĩa cố định, không thay bằng một chuỗi.
"""
M = {
 'hug_busy':     ("                if self.side_out[(u.id, side)] or self.side_in[(u.id, side)]:\n                    continue\n                hugs[",
                  "                if False:\n                    continue\n                hugs["),
 'head20':       ("head = self.tm.w(lrow.text) + 30", "head = self.tm.w(lrow.text) + 20"),
 'head_space':   ("head = self.tm.w(lrow.text) + 30", "head = self.tm.w(' '.join(lrow.lines).strip()) + 30"),
 'need_first':   ("                widths[0] += need / 2", "                widths[0] += need"),
 'col_edge':     ("else cx0 + cw / 2", "else cx0"),
 'area2':        ("                    cost += area(box, bb) * 4", "                    cost += area(box, bb) * 2"),
 'own_wire':     ("if eid != e.id and seg_hits(a, b, box):", "if seg_hits(a, b, box):"),
 'no_break':     ("                if cost == 0:\n                    break", "                if False:\n                    break"),
 'tie_later':    ("if best is None or cost < best[0]:", "if best is None or cost <= best[0]:"),
 'short20':      ("            if L >= 12:", "            if L >= 20:"),
 'no_header':    ("boxes.append(('header', (-1e6, -1e6, 1e6,", "boxes.append(('header', (-1e6, -1e6, -1e6,"),
 'no_lane_sep':  ("for x in self.lane_x[1:]]", "for x in self.lane_x[:0]]"),
 'seg_touch_x':  ("return x1 < box[2] and x2 > box[0] and y1 < box[3] and y2 > box[1]",
                  "return x1 <= box[2] and x2 >= box[0] and y1 < box[3] and y2 > box[1]"),
 'seg_touch_y':  ("return x1 < box[2] and x2 > box[0] and y1 < box[3] and y2 > box[1]",
                  "return x1 < box[2] and x2 > box[0] and y1 <= box[3] and y2 >= box[1]"),
}
