from pathlib import Path

p = Path(__file__).resolve().parents[1] / "scripts/gen_danmaku_manage.py"
text = p.read_text(encoding="utf-8")
src_open = "<m" + "otion "
src_close = "</m" + "otion>"
dst_open = "<d" + "iv "
dst_close = "</d" + "iv>"
line = "out = out.replace(%r, %r).replace(%r, %r)\n" % (
    src_open,
    dst_open,
    src_close,
    dst_close,
)
lines = []
for ln in text.splitlines(True):
    if ln.startswith("out = out.replace"):
        lines.append(line)
    else:
        lines.append(ln)
p.write_text("".join(lines), encoding="utf-8")
print("fixed", repr(line))
