"""Đột biến Python tương ứng với các đột biến route và track trong mutate.sh.

Mỗi mục là (chuỗi gốc, chuỗi thay) trên reference/flowtable2drawio.py. Dùng với
tools/fuzz_cases.py --stage route. Hai đột biến đã chứng minh là tương đương
vẫn để ở đây, vì chạy lại chúng là cách kiểm chứng minh đó còn đúng.
"""
M = {
 'A_no_bottom':   ("            if self.side_out[(u.id, 'B')]:\n                continue", "            if False:\n                continue"),
 'B_no_target':   ("            if self.side_out[(v.id, OPP[s])] or self.side_in[(v.id, OPP[s])]:", "            if False:"),
 'B_no_attach':   ("            if s in self.attach_sides(u) or OPP[s] in self.attach_sides(v):", "            if False:"),
 'C_no_attach':   ("            if s in self.attach_sides(u):\n", "            if False:\n"),
 'C_no_turn':     ("hcells = self.cells_between(self.gkey(u), self.gkey(v), u.row) + [turn]", "hcells = self.cells_between(self.gkey(u), self.gkey(v), u.row)"),
 'back_bottom':   ("choices = [s, OPP[s]] if e.back or v.row <= u.row else [s, 'B', OPP[s]]", "choices = [s, 'B', OPP[s]]"),
 'D_bottom_1st':  ("choices = [s, OPP[s]] if e.back or v.row <= u.row else [s, 'B', OPP[s]]", "choices = [s, OPP[s]] if e.back or v.row <= u.row else ['B', s, OPP[s]]"),
 'D_no_vcells':   ("if all(c not in occ and cells_v[c] <= {v.id} for c in vcells) and (u.lane, u.col) != (v.lane, v.col):", "if False:"),
 'nonrect_share': ("            return not used", "            return True"),
 'single_split':  ("if u.kind in NON_RECT or len(es) == 1:", "if u.kind in NON_RECT:"),
 'no_preset':     ("fr = [0.25, 0.75, 0.125, 0.875, 0.375, 0.625][:n] if n <= 6 else [(i + 1) / (n + 1) for i in range(n)]", "fr = [(i + 1) / (n + 1) for i in range(n)]"),
 'side_by_col':   ("(self.items[e.dst].row, e.order) if side in ('L', 'R')", "(self.items[e.dst].row, e.order) if False"),
 'left_frac':     ("(1.0 if side == 'R' else 0.0, f)", "(1.0 if side == 'R' else 1.0, f)"),
 'no_bus_first':  ("chosen = next((ti for ti in range(len(tracks)) if fits(ti, True)), None)", "chosen = None"),
 'samekey_prec':  ("if a.key is not None and a.key == b.key:", "if False:"),
 'overlap_strict':("return a.lo <= b.hi and b.lo <= a.hi", "return a.lo < b.hi and b.lo < a.hi"),
 'insert_last':   ("chosen = max(lo, min(hi, len(tracks))) if lo <= hi else len(tracks)", "chosen = len(tracks)"),
 'orderok_fwd':   ("if must_precede(s, t) and not ti < tj:", "if False:"),
}
