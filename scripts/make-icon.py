#!/usr/bin/env python3
"""Generate build/windows/icon.ico for the Windows shortcuts.

Simple flat icon: rounded dark panel with a gauge/accent bar and "LM".
Run: python3 scripts/make-icon.py
"""
import os
from PIL import Image, ImageDraw

BG_TOP = (24, 32, 48)
BG_BOT = (12, 16, 26)
ACCENT = (86, 182, 255)
TEXT = (238, 244, 255)

S = 512
img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
d = ImageDraw.Draw(img)

# Vertical gradient background with rounded corners.
grad = Image.new("RGBA", (S, S))
gd = ImageDraw.Draw(grad)
for y in range(S):
    t = y / (S - 1)
    gd.line(
        [(0, y), (S, y)],
        fill=(
            int(BG_TOP[0] + (BG_BOT[0] - BG_TOP[0]) * t),
            int(BG_TOP[1] + (BG_BOT[1] - BG_TOP[1]) * t),
            int(BG_TOP[2] + (BG_BOT[2] - BG_TOP[2]) * t),
            255,
        ),
    )
mask = Image.new("L", (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle([0, 0, S - 1, S - 1], radius=96, fill=255)
img.paste(grad, (0, 0), mask)

# Accent bar (a little "usage gauge").
d.rounded_rectangle([96, 340, 416, 372], radius=16, fill=(44, 56, 78))
d.rounded_rectangle([96, 340, 300, 372], radius=16, fill=ACCENT)

# "LM" wordmark.
try:
    from PIL import ImageFont

    font = None
    for path in (
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
    ):
        if os.path.exists(path):
            font = ImageFont.truetype(path, 230)
            break
    if font:
        d.text((S / 2, 190), "LM", font=font, fill=TEXT, anchor="mm")
except Exception:
    pass

root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
out_dir = os.path.join(root, "build", "windows")
os.makedirs(out_dir, exist_ok=True)
out = os.path.join(out_dir, "icon.ico")
img.save(out, sizes=[(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)])
print("wrote", out)

# Also emit a PNG for Linux .desktop entries and macOS .app bundles.
assets_dir = os.path.join(root, "assets")
os.makedirs(assets_dir, exist_ok=True)
png = os.path.join(assets_dir, "llamaherd.png")
img.save(png, format="PNG")
print("wrote", png)
