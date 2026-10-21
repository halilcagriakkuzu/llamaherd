#!/usr/bin/env python3
"""Generate the LlamaHerd icon: assets/llamaherd.png (512 px, used by .desktop
entries and the macOS bundle) and the multi-size assets/llamaherd.ico (shipped
in release archives for the Windows shortcuts; also copied to
build/windows/icon.ico for the Windows executable).

Design: a front-facing llama head on a warm orange rounded square, with a green
"server running" dot. Drawn at 2048 px and downsampled for clean edges.
Run: python3 scripts/make-icon.py   (needs Pillow)
"""
import os

from PIL import Image, ImageDraw

S = 2048


def lerp(a, b, t): return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))

def bg(top, bot, radius=0.2, diag=True):
    g = Image.new("RGBA", (S, S))
    px = g.load()
    for y in range(S):
        for x in range(S):
            t = ((x + y) / (2 * S)) if diag else y / S
            px[x, y] = lerp(top, bot, t) + (255,)
    m = Image.new("L", (S, S), 0)
    ImageDraw.Draw(m).rounded_rectangle([0, 0, S - 1, S - 1], radius=int(S * radius), fill=255)
    out = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    out.paste(g, (0, 0), m)
    return out

def P(v): return v * S

def llama(d, cx, cy, s, body, inner, dark, ear_in=None):
    """Front-facing llama head. cx,cy = head centre (unit), s = scale (unit)."""
    def e(x0, y0, x1, y1, fill):
        d.ellipse([P(cx + x0 * s), P(cy + y0 * s), P(cx + x1 * s), P(cy + y1 * s)], fill=fill)
    def r(x0, y0, x1, y1, rad, fill):
        d.rounded_rectangle([P(cx + x0 * s), P(cy + y0 * s), P(cx + x1 * s), P(cy + y1 * s)], radius=P(rad * s), fill=fill)
    def poly(pts, fill):
        d.polygon([(P(cx + x * s), P(cy + y * s)) for x, y in pts], fill=fill)
    # ears: shorter, leaf-shaped, tilting slightly outward
    for sx in (-1, 1):
        poly([(sx * 0.12, -0.24), (sx * 0.38, -0.20), (sx * 0.40, -0.52), (sx * 0.31, -0.64)], body)
        e(sx * 0.35 - 0.06, -0.69, sx * 0.35 + 0.06, -0.58, body)
        poly([(sx * 0.19, -0.27), (sx * 0.33, -0.25), (sx * 0.35, -0.50), (sx * 0.31, -0.58)], ear_in or inner)
    # neck + head
    r(-0.30, 0.20, 0.30, 0.98, 0.18, body)
    e(-0.40, -0.40, 0.40, 0.40, body)
    r(-0.27, 0.0, 0.27, 0.62, 0.22, body)
    # muzzle
    e(-0.235, 0.14, 0.235, 0.54, inner)
    # eyes
    for sx in (-1, 1):
        e(sx * 0.19 - 0.055, -0.08, sx * 0.19 + 0.055, 0.04, dark)
    # nose + nostrils
    e(-0.07, 0.20, 0.07, 0.28, dark)
    # fringe tuft
    poly([(-0.16, -0.32), (0.0, -0.46), (0.16, -0.32), (0.0, -0.20)], inner)


def render():
    c = bg((255, 150, 60), (214, 70, 60))
    d = ImageDraw.Draw(c)
    llama(d, 0.5, 0.57, 0.58, (255, 248, 240), (255, 196, 150), (92, 28, 24))
    d.ellipse([P(.70), P(.70), P(.88), P(.88)], fill=(30, 36, 52))
    d.ellipse([P(.74), P(.74), P(.84), P(.84)], fill=(60, 224, 140))
    return c


if __name__ == "__main__":
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    big = render()
    img = big.resize((512, 512), Image.LANCZOS)

    out_dir = os.path.join(root, "build", "windows")
    os.makedirs(out_dir, exist_ok=True)
    ico = os.path.join(out_dir, "icon.ico")
    # BMP-encoded entries: the most compatible form for Explorer/shortcuts
    # (PNG-compressed small sizes can show up as a blank document icon).
    ico_sizes = [(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)]
    big.save(ico, sizes=ico_sizes, bitmap_format="bmp")
    print("wrote", ico)

    assets_dir = os.path.join(root, "assets")
    os.makedirs(assets_dir, exist_ok=True)
    png = os.path.join(assets_dir, "llamaherd.png")
    img.save(png, format="PNG")
    print("wrote", png)

    # Shipped in release archives so the WSL installer can set the shortcut icon.
    ico_asset = os.path.join(assets_dir, "llamaherd.ico")
    big.save(ico_asset, sizes=ico_sizes, bitmap_format="bmp")
    print("wrote", ico_asset)
