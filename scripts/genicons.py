"""Generate Nook's app icon (resources/icon.png) and menu-bar template icon
(frontend/assets/tray.png). Pure stdlib. Run: python3 scripts/genicons.py"""
import struct, zlib

def png(path, w, h, px):
    raw = b''.join(b'\x00' + bytes(c for p in px[y*w:(y+1)*w] for c in p) for y in range(h))
    def ch(t, d): return struct.pack('>I', len(d)) + t + d + struct.pack('>I', zlib.crc32(t+d) & 0xffffffff)
    open(path, 'wb').write(b'\x89PNG\r\n\x1a\n' + ch(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 6, 0, 0, 0)) + ch(b'IDAT', zlib.compress(raw, 9)) + ch(b'IEND', b''))

def hexc(s): return (int(s[1:3],16), int(s[3:5],16), int(s[5:7],16), 255)

# ---------- the dollhouse: 24x24 cells, two floors
N = 24
C = dict(o='#1b1230', p='#ffc58a', g='#ffe0b0', b='#7f93cf', f='#a8693c', w='#bfe3ff', F='#fbf7ef',
         y='#ffd36b', t='#2fb5a0', s='#ffd9b8', h='#3a2a5a', c='#ff7b3a', W='#ffffff', d='#3b2f55', B='#5f73b0')
G = [[None]*N for _ in range(N)]
def rect(x0, y0, x1, y1, k):
    for y in range(y0, y1+1):
        for x in range(x0, x1+1): G[y][x] = k
rect(1,1,22,10,'p'); rect(1,12,22,21,'b'); rect(1,11,22,11,'f'); rect(1,22,22,22,'f')
# upper floor: window, pendant lamp with glow, dancer, speaker
rect(3,2,8,7,'F'); rect(4,3,7,6,'w'); rect(5,3,5,6,'F'); rect(4,4,7,4,'F'); G[4][5]='F'
rect(13,1,13,2,'o'); rect(12,3,14,3,'y'); rect(11,4,15,4,'g')
rect(17,4,18,4,'h'); rect(17,5,18,6,'s'); rect(17,7,18,9,'c'); G[10][17]='h'; G[10][18]='h'; G[6][16]='c'; G[5][19]='c'
rect(20,6,22,10,'d'); G[8][21]='y'
# lower floor: window, small lamp, bed with a sleeper
rect(3,14,6,18,'F'); rect(4,15,5,17,'w')
rect(12,12,12,13,'o'); rect(11,14,13,14,'y')
rect(9,19,22,21,'t'); rect(19,17,21,18,'W'); rect(19,18,20,19,'s'); rect(19,17,20,17,'h')
rect(9,18,9,21,'f')
# rounded box corners
for x, y in [(0,0),(23,0),(0,23),(23,23),(1,0),(0,1),(22,0),(23,1),(0,22),(1,23),(23,22),(22,23)]: G[y][x] = None

# ---------- app icon 1024
W = 1024
cell = 34
HS = N * cell
off = (W - HS) // 2
img = []
R = 224; cx = W / 2
for y in range(W):
    for x in range(W):
        dx = max(abs(x-cx) - (W/2-R-24), 0); dy = max(abs(y-cx) - (W/2-R-24), 0)
        d = (dx*dx + dy*dy) ** 0.5
        if d > R: img.append((0,0,0,0)); continue
        t = y / W
        gx = (x - W*0.5) / W; gy = (y - W*0.45) / W
        glow = max(0.0, 1 - (gx*gx + gy*gy) * 5)
        base = (46 + 22*t, 33 + 16*t, 72 + 40*t)
        a = 255 if d < R - 1 else int(255 * (R - d))
        img.append(tuple(min(255, int(base[i] + glow * (60, 40, 20)[i])) for i in range(3)) + (a,))
def blend(x, y, c, a):
    if 0 <= x < W and 0 <= y < W and img[y*W+x][3]:
        o = img[y*W+x]
        img[y*W+x] = tuple(int(o[i]*(1-a) + c[i]*a) for i in range(3)) + (o[3],)
def put(x, y, c):
    if 0 <= x < W and 0 <= y < W and img[y*W+x][3]: img[y*W+x] = c
# block shadow
for gy in range(N):
    for gx in range(N):
        if G[gy][gx]:
            for yy in range(cell):
                for xx in range(cell):
                    blend(off + gx*cell + xx + 10, off + gy*cell + yy + 14, (12, 6, 24), 0.55)
for gy in range(N):
    for gx in range(N):
        k = G[gy][gx]
        if not k: continue
        r, g, b, _ = hexc(C[k])
        for yy in range(cell):
            for xx in range(cell):
                px, py = gx*cell + xx, gy*cell + yy
                s = (px + py) / (2 * HS)          # glass sheen from the top-left
                lift = 0.0
                if k == 'o': lift = 0.0
                elif s < 0.16: lift = 0.22 * (1 - s / 0.16) + 0.06
                elif 0.20 < s < 0.235: lift = 0.20
                put(off + px, off + py, (int(r + (255-r)*lift), int(g + (255-g)*lift), int(b + (255-b)*lift), 255))
png('resources/icon.png', W, W, img)

# ---------- menu-bar template: 16x16 grid at 2x = 32px, black + alpha
T = [[0]*16 for _ in range(16)]
def tr(x0, y0, x1, y1, v=1):
    for y in range(y0, y1+1):
        for x in range(x0, x1+1): T[y][x] = v
# roof (stepped triangle), body, then cut-outs
for k in range(7):
    tr(7-k, 1+k, 8+k, 1+k)
tr(1,8,14,14)
tr(3,10,6,12,0)          # window, upper floor
tr(9,10,12,12,0)         # window / resident
tr(10,11,11,11,1)        # resident dot
tr(2,13,13,13,0)         # floor line cut
px = []
for y in range(32):
    for x in range(32):
        px.append((0,0,0,255) if T[y//2][x//2] else (0,0,0,0))
png('frontend/assets/tray.png', 32, 32, px)
