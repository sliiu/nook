"""Generate Nook's app icon (resources/icon.png) and menu-bar template icon
(frontend/assets/tray.png). Pure stdlib. Run: python3 scripts/genicons.py

The mark is a pixel N: two stems and a staircase stroke that starts inside the
left stem and ends inside the right, so it stays one letter at small sizes.
"""
import struct, zlib

def png(path, w, h, px):
    raw = b''.join(b'\x00' + bytes(c for p in px[y*w:(y+1)*w] for c in p) for y in range(h))
    def ch(t, d): return struct.pack('>I', len(d)) + t + d + struct.pack('>I', zlib.crc32(t+d) & 0xffffffff)
    open(path, 'wb').write(b'\x89PNG\r\n\x1a\n' + ch(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 6, 0, 0, 0)) + ch(b'IDAT', zlib.compress(raw, 9)) + ch(b'IEND', b''))

def hexc(s): return (int(s[1:3],16), int(s[3:5],16), int(s[5:7],16), 255)

# Designed on the 16px grid the menu bar actually shows, so nothing thinner
# than two pixels survives. Stems sit at columns 3-4 and 11-12. The diagonal
# runs from inside the left stem to inside the right, two pixels wide, one row
# of overlap between steps so it reads as a line and not a dotted stair.
ROWS = COLS = 16
GRID = [['.']*COLS for _ in range(ROWS)]
for y in range(3, 13):
    for x in (3, 4, 11, 12): GRID[y][x] = '#'
for x, y in ((4,3),(5,4),(6,5),(7,6),(8,7),(9,8),(10,9)):
    for t in range(2):
        GRID[y+t][x] = '#'
        GRID[y+t][x+1] = '#'

def paint(scale, fg, bg):
    """Scale the 16px grid by an integer, so every pixel lands on a pixel.

    Dividing a target size by the grid rounds the cells away and the letter
    falls apart; scaling up never does. bg is None for a transparent ground.
    """
    W = COLS * scale
    img = [(0,0,0,0) if bg is None else bg]*(W*W)
    for gy in range(ROWS):
        for gx in range(COLS):
            if GRID[gy][gx] != '#': continue
            x0, y0 = gx*scale, gy*scale
            for y in range(y0, y0+scale):
                img[y*W+x0:y*W+x0+scale] = [fg]*scale
    return W, img

# ---------- app icon: cream N on the night ground. 16 x 64 = 1024.
W, img = paint(64, hexc('#fbf7ef'), hexc('#1f1833'))
png('resources/icon.png', W, W, img)

# ---------- menu-bar template: black N, transparent ground, at 2x so the
# two-pixel stems stay two pixels when macOS draws it at 16pt.
W, img = paint(2, (0,0,0,255), None)
png('frontend/assets/tray.png', W, W, img)
