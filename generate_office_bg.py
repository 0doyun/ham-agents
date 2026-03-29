#!/usr/bin/env python3
"""Generate pixel art office background 480x180 for ham-agents."""
from PIL import Image, ImageDraw

W, H = 480, 180
P = 3  # pixel unit

# Color palette
WALL        = (245, 230, 211)   # #F5E6D3 cream
WALL_DARK   = (180, 155, 120)   # wall shadow
FLOOR       = (196, 149, 106)   # #C4956A wood
FLOOR_PAT   = (184, 135,  92)   # #B8875C floor pattern
BORDER      = ( 80,  55,  30)   # wall-floor border (dark)
DESK        = (139, 105,  20)   # #8B6914 dark wood desk
CHAIR       = ( 58,  58,  58)   # #3A3A3A dark grey chair
MON_BEZEL   = ( 42,  42,  42)   # #2A2A2A monitor bezel
MON_SCREEN  = ( 26,  58,  42)   # #1A3A2A dark green terminal
MON_CODE    = ( 74, 222, 128)   # #4ADE80 code green
SHELF       = (107,  66,  38)   # #6B4226 dark wood shelf
BOOK_R      = (231,  76,  60)   # #E74C3C red
BOOK_B      = ( 52, 152, 219)   # #3498DB blue
BOOK_G      = ( 46, 204, 113)   # #2ECC71 green
BOOK_Y      = (241, 196,  15)   # #F1C40F yellow
COFFEE_M    = ( 74,  74,  74)   # #4A4A4A coffee machine
ALERT_RED   = (231,  76,  60)   # #E74C3C alert
WIN_FRAME   = (139, 105,  20)   # #8B6914 window frame
WIN_GLASS   = (135, 206, 235)   # #87CEEB sky blue
WHITE       = (255, 255, 255)
BLACK       = (  0,   0,   0)
WATER       = (100, 180, 220)   # water cooler blue
TABLE_SMALL = (160, 120,  60)   # small table
DASH_SCREEN = ( 20,  40,  80)   # dashboard screen
DASH_LINE   = ( 60, 160, 255)   # dashboard line
CLOCK_FACE  = (250, 245, 235)
POSTER_BG   = (200, 185, 165)
POST_IT_1   = (255, 230,  80)
POST_IT_2   = (150, 220, 150)
POST_IT_3   = (255, 160, 130)
ALERT_LAMP  = (255,  80,  50)

WALL_H = 63   # top ~35% of 180

img = Image.new('RGBA', (W, H), (0, 0, 0, 0))
draw = ImageDraw.Draw(img)

def px(x, y, color, w=P, h=P):
    """Draw a pixel-unit rectangle."""
    draw.rectangle([x, y, x + w - 1, y + h - 1], fill=color + (255,))

def rect(x, y, w, h, color):
    draw.rectangle([x, y, x + w - 1, y + h - 1], fill=color + (255,))

def hline(x, y, length, color, thickness=1):
    draw.rectangle([x, y, x + length - 1, y + thickness - 1], fill=color + (255,))

def vline(x, y, length, color, thickness=1):
    draw.rectangle([x, y, x + thickness - 1, y + length - 1], fill=color + (255,))

# ─── WALL ───────────────────────────────────────────────────────────────────
rect(0, 0, W, WALL_H, WALL)

# Wall-floor border (2px dark line)
rect(0, WALL_H, W, 2, BORDER)

# ─── FLOOR ──────────────────────────────────────────────────────────────────
rect(0, WALL_H + 2, W, H - WALL_H - 2, FLOOR)

# Floor plank pattern (horizontal lines every 6px, subtle)
y = WALL_H + 8
while y < H:
    rect(0, y, W, 1, FLOOR_PAT)
    y += 9

# Floor vertical grain every 24px
x = 24
while x < W:
    rect(x, WALL_H + 2, 1, H - WALL_H - 2, FLOOR_PAT)
    x += 24

# ─── WINDOW (centered-left: x≈130) ─────────────────────────────────────────
wx, wy = 130, 8
ww, wh = 60, 42  # outer frame

# Frame (dark wood)
rect(wx, wy, ww, wh, WIN_FRAME)
# Glass panes (inner)
rect(wx + 3, wy + 3, ww - 6, wh - 6, WIN_GLASS)
# Cross divider
rect(wx + 3, wy + wh // 2 - 1, ww - 6, 2, WIN_FRAME)   # horizontal
rect(wx + ww // 2 - 1, wy + 3, 2, wh - 6, WIN_FRAME)   # vertical
# Window glint
rect(wx + 6, wy + 6, 6, 3, (200, 235, 255))

# ─── CLOCK (right of window) ────────────────────────────────────────────────
cx_c, cy_c, cr = 220, 12, 15
draw.ellipse([cx_c - cr, cy_c - cr, cx_c + cr, cy_c + cr], fill=CLOCK_FACE + (255,))
draw.ellipse([cx_c - cr, cy_c - cr, cx_c + cr, cy_c + cr], outline=WIN_FRAME + (255,), width=2)
# Clock hands (pixel style)
rect(cx_c - 1, cy_c - 10, 2, 10, WIN_FRAME)   # minute hand (up)
rect(cx_c, cy_c - 1, 7, 2, WIN_FRAME)          # hour hand (right)
rect(cx_c - 1, cy_c - 1, 2, 2, BLACK)          # center dot

# ─── WHITEBOARD / BULLETIN BOARD (right wall) ───────────────────────────────
bx, by = 310, 6
bw, bh = 80, 48
rect(bx, by, bw, bh, POSTER_BG)
rect(bx - 2, by - 2, bw + 4, bh + 4, WIN_FRAME)  # frame
# Post-its
rect(bx + 4,  by + 4,  18, 12, POST_IT_1)
rect(bx + 26, by + 4,  18, 12, POST_IT_2)
rect(bx + 48, by + 4,  18, 12, POST_IT_3)
rect(bx + 4,  by + 20, 18, 12, POST_IT_2)
rect(bx + 26, by + 20, 18, 12, POST_IT_1)
rect(bx + 48, by + 20, 18, 12, POST_IT_3)
# Lines on post-its (tiny)
for pi_x, pi_y in [(bx+4, by+4), (bx+26, by+4), (bx+48, by+4),
                   (bx+4, by+20), (bx+26, by+20), (bx+48, by+20)]:
    rect(pi_x + 2, pi_y + 4, 14, 1, (0, 0, 0))
    rect(pi_x + 2, pi_y + 7, 10, 1, (0, 0, 0))

# ─── WORKSTATION A (x=20..100) ───────────────────────────────────────────────
# Desk top: x=20, y=floor+30 = 95, w=80, h=6
DA_X, DA_Y = 20, 95
rect(DA_X, DA_Y, 80, 6, DESK)
# Desk legs
rect(DA_X + 3,  DA_Y + 6, 5, 20, DESK)
rect(DA_X + 72, DA_Y + 6, 5, 20, DESK)
# Monitor (on desk top)
MX_A = DA_X + 18
MY_A = DA_Y - 33
rect(MX_A, MY_A, 36, 27, MON_BEZEL)
rect(MX_A + 2, MY_A + 2, 32, 21, MON_SCREEN)
# Code lines on screen A
for li, col in enumerate([MON_CODE, (40, 180, 80), MON_CODE, (40, 180, 80), MON_CODE]):
    lx = MX_A + 4
    lw = [22, 14, 18, 10, 20][li]
    ly = MY_A + 3 + li * 4
    rect(lx, ly, lw, 2, col)
# Monitor stand
rect(MX_A + 15, MY_A + 27, 6, 4, MON_BEZEL)
rect(MX_A + 10, MY_A + 31, 16, 3, MON_BEZEL)
# Chair A (in front of desk)
CA_X, CA_Y = DA_X + 25, DA_Y + 10
rect(CA_X, CA_Y, 26, 4, CHAIR)           # seat
rect(CA_X + 3, CA_Y - 18, 20, 18, CHAIR) # back
rect(CA_X + 2, CA_Y + 4, 4, 12, CHAIR)   # left leg
rect(CA_X + 20, CA_Y + 4, 4, 12, CHAIR)  # right leg

# ─── WORKSTATION B (x=120..200) ─────────────────────────────────────────────
DB_X, DB_Y = 118, 95
rect(DB_X, DB_Y, 80, 6, DESK)
rect(DB_X + 3,  DB_Y + 6, 5, 20, DESK)
rect(DB_X + 72, DB_Y + 6, 5, 20, DESK)
# Monitor B
MX_B = DB_X + 18
MY_B = DB_Y - 33
rect(MX_B, MY_B, 36, 27, MON_BEZEL)
rect(MX_B + 2, MY_B + 2, 32, 21, MON_SCREEN)
for li, col in enumerate([MON_CODE, MON_CODE, (40, 180, 80), MON_CODE, (40, 180, 80)]):
    lw = [18, 24, 12, 20, 16][li]
    rect(MX_B + 4, MY_B + 3 + li * 4, lw, 2, col)
rect(MX_B + 15, MY_B + 27, 6, 4, MON_BEZEL)
rect(MX_B + 10, MY_B + 31, 16, 3, MON_BEZEL)
# Chair B
CB_X, CB_Y = DB_X + 25, DB_Y + 10
rect(CB_X, CB_Y, 26, 4, CHAIR)
rect(CB_X + 3, CB_Y - 18, 20, 18, CHAIR)
rect(CB_X + 2, CB_Y + 4, 4, 12, CHAIR)
rect(CB_X + 20, CB_Y + 4, 4, 12, CHAIR)

# ─── COFFEE / BREAK AREA (x=215..280) ───────────────────────────────────────
# Small table
ST_X, ST_Y = 215, 110
rect(ST_X, ST_Y, 55, 5, TABLE_SMALL)
rect(ST_X + 4,  ST_Y + 5, 5, 18, TABLE_SMALL)
rect(ST_X + 46, ST_Y + 5, 5, 18, TABLE_SMALL)
# Coffee machine (on table)
CM_X, CM_Y = ST_X + 4, ST_Y - 30
rect(CM_X, CM_Y, 20, 30, COFFEE_M)
rect(CM_X + 2, CM_Y + 2, 16, 12, (50, 50, 50))   # front panel
rect(CM_X + 6, CM_Y + 16, 8, 6, (80, 60, 40))    # coffee dispensing area
rect(CM_X + 4, CM_Y + 4, 4, 4, (255, 80, 0))     # power led orange
rect(CM_X + 10, CM_Y + 4, 3, 3, (0, 200, 100))   # status led green
# Steam from coffee machine
for si in range(3):
    sx = CM_X + 8 + si * 3
    rect(sx, CM_Y - 6, 2, 4, (200, 200, 200))

# Water cooler (right of table)
WC_X, WC_Y = ST_X + 36, ST_Y - 42
rect(WC_X, WC_Y, 14, 42, (220, 235, 245))        # body
rect(WC_X + 2, WC_Y + 2, 10, 18, WATER)          # water tank
rect(WC_X + 3, WC_Y + 22, 8, 14, (180, 200, 215)) # dispenser body
rect(WC_X + 4, WC_Y + 26, 6, 4, (140, 180, 210)) # tap area

# ─── BOOKSHELF (x=290..370) ─────────────────────────────────────────────────
SH_X, SH_Y = 290, 65
SH_W, SH_H = 72, H - SH_Y - 5  # tall floor-to-near-top
rect(SH_X, SH_Y, SH_W, SH_H, SHELF)
# 3 shelves
for si in range(3):
    sy = SH_Y + 6 + si * 28
    rect(SH_X + 3, sy, SH_W - 6, 4, (90, 55, 28))   # shelf plank
    # Books on shelf
    books = [BOOK_R, BOOK_B, BOOK_G, BOOK_Y, BOOK_R, BOOK_B, BOOK_G]
    bk_x = SH_X + 5
    for bk_col in books:
        bk_w = 6 + (hash(bk_col) % 3)
        rect(bk_x, sy - 20, bk_w, 20, bk_col)
        bk_x += bk_w + 1
        if bk_x > SH_X + SH_W - 8:
            break
# Shelf sides
rect(SH_X, SH_Y, 3, SH_H, (80, 48, 22))
rect(SH_X + SH_W - 3, SH_Y, 3, SH_H, (80, 48, 22))
# Small book pile in front
for bi, bkc in enumerate([BOOK_Y, BOOK_R, BOOK_B]):
    rect(SH_X + 5 + bi * 3, H - 25 + bi * 2, 14, 4, bkc)

# ─── ALERT / DASHBOARD CORNER (x=380..460) ──────────────────────────────────
# Mounted dashboard monitor (on wall)
DM_X, DM_Y = 382, 8
DM_W, DM_H = 72, 48
rect(DM_X - 3, DM_Y - 3, DM_W + 6, DM_H + 6, MON_BEZEL)
rect(DM_X, DM_Y, DM_W, DM_H, DASH_SCREEN)
# Dashboard graphs
for gi in range(3):
    gx = DM_X + 4 + gi * 22
    # Bar chart style
    for bi in range(4):
        bh_dash = 5 + (gi * 3 + bi * 2) % 18
        rect(gx + bi * 4, DM_Y + DM_H - 4 - bh_dash, 3, bh_dash,
             DASH_LINE if gi == 0 else ((255, 200, 50) if gi == 1 else ALERT_RED))
    # Label line
    rect(gx, DM_Y + 4, 18, 1, (80, 100, 140))

# Alert lamp (below dashboard monitor, on wall)
AL_X, AL_Y = 430, 60
rect(AL_X - 6, AL_Y, 12, 4, (60, 60, 60))   # mount bracket
draw.ellipse([AL_X - 8, AL_Y + 4, AL_X + 8, AL_Y + 20], fill=ALERT_LAMP + (255,))
draw.ellipse([AL_X - 5, AL_Y + 7, AL_X + 5, AL_Y + 17], fill=(255, 160, 120) + (255,))  # glow

# Small desk under dashboard
AD_X, AD_Y = 378, 95
rect(AD_X, AD_Y, 74, 6, DESK)
rect(AD_X + 3,  AD_Y + 6, 5, 20, DESK)
rect(AD_X + 66, AD_Y + 6, 5, 20, DESK)
# Small items on desk
rect(AD_X + 8,  AD_Y - 12, 12, 12, MON_BEZEL)    # small monitor
rect(AD_X + 10, AD_Y - 10, 8, 8, MON_SCREEN)
rect(AD_X + 11, AD_Y - 9, 6, 2, DASH_LINE)
rect(AD_X + 11, AD_Y - 6, 4, 2, ALERT_RED)
rect(AD_X + 28, AD_Y - 8, 10, 8, (60, 60, 60))   # keyboard-like
for ki in range(4):
    rect(AD_X + 29 + ki * 2, AD_Y - 6, 1, 4, (90, 90, 90))
# Chair alert desk
CC_X, CC_Y = AD_X + 22, AD_Y + 10
rect(CC_X, CC_Y, 26, 4, CHAIR)
rect(CC_X + 3, CC_Y - 18, 20, 18, CHAIR)
rect(CC_X + 2, CC_Y + 4, 4, 12, CHAIR)
rect(CC_X + 20, CC_Y + 4, 4, 12, CHAIR)

# ─── Save ────────────────────────────────────────────────────────────────────
out_path = '/Users/User/projects/ham-agents/apps/macos/HamMenuBarApp/Sources/Resources/office_background.png'
img.save(out_path)
print(f"Saved {out_path} ({W}x{H})")
