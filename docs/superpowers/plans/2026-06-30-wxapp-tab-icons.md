# wxapp Tab 图标 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按已确认的 A 方案，为 `wplink/wxapp` 替换 5 个 tab 的普通态和选中态图标，并补齐 tabBar 字体与图标尺寸配置。

**Architecture:** 使用一次性 Pillow 绘图脚本生成 10 个 `81 x 81` 透明 PNG，普通态使用 `#454a54`，选中态使用 `#c23a00`。`wxapp/pages.json` 只新增 `fontSize` 和 `iconWidth`，不改 tab 数量、路径、文案或业务逻辑。

**Tech Stack:** uni-app / 微信小程序 `pages.json` tabBar 配置、PNG 静态资源、Python 3 + Pillow。

---

## 文件结构

- Modify: `wxapp/pages.json`
  - 职责：维护小程序全局页面与 tabBar 配置。
  - 本次只在 `tabBar` 内新增 `fontSize: "10px"` 和 `iconWidth: "28px"`。
- Modify: `wxapp/static/tabbar/home.png`
- Modify: `wxapp/static/tabbar/home-active.png`
- Modify: `wxapp/static/tabbar/search.png`
- Modify: `wxapp/static/tabbar/search-active.png`
- Modify: `wxapp/static/tabbar/publish.png`
- Modify: `wxapp/static/tabbar/publish-active.png`
- Modify: `wxapp/static/tabbar/messages.png`
- Modify: `wxapp/static/tabbar/messages-active.png`
- Modify: `wxapp/static/tabbar/my.png`
- Modify: `wxapp/static/tabbar/my-active.png`
  - 职责：微信小程序 tabBar 图标资源。
  - 本次全部保持 `81 x 81` 透明 PNG。

## Task 1: 补齐 tabBar 尺寸配置

**Files:**
- Modify: `wxapp/pages.json`

- [x] **Step 1: 查看当前 tabBar 配置**

Run:

```bash
sed -n '128,165p' wxapp/pages.json
```

Expected: `tabBar` 中已有 `color`、`selectedColor`、`backgroundColor`、`borderStyle` 和 5 个 tab，但没有 `fontSize`、`iconWidth`。

- [x] **Step 2: 添加尺寸配置**

Patch:

```diff
   "tabBar": {
     "color": "#454a54",
     "selectedColor": "#c23a00",
     "backgroundColor": "#ffffff",
     "borderStyle": "black",
+    "fontSize": "10px",
+    "iconWidth": "28px",
     "list": [
```

- [x] **Step 3: 验证 JSON 可解析且配置存在**

Run:

```bash
node -e "const fs=require('fs'); const p=JSON.parse(fs.readFileSync('wxapp/pages.json','utf8')); if(p.tabBar.fontSize!=='10px'||p.tabBar.iconWidth!=='28px'||p.tabBar.list.length!==5) process.exit(1); console.log('tabBar ok')"
```

Expected: prints `tabBar ok`.

## Task 2: 生成并替换 10 个 tab PNG

**Files:**
- Modify: `wxapp/static/tabbar/home.png`
- Modify: `wxapp/static/tabbar/home-active.png`
- Modify: `wxapp/static/tabbar/search.png`
- Modify: `wxapp/static/tabbar/search-active.png`
- Modify: `wxapp/static/tabbar/publish.png`
- Modify: `wxapp/static/tabbar/publish-active.png`
- Modify: `wxapp/static/tabbar/messages.png`
- Modify: `wxapp/static/tabbar/messages-active.png`
- Modify: `wxapp/static/tabbar/my.png`
- Modify: `wxapp/static/tabbar/my-active.png`

- [x] **Step 1: 运行一次性 PNG 生成脚本**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
from PIL import Image, ImageDraw

OUT = Path("wxapp/static/tabbar")
SIZE = 81
VIEWBOX = 32
SCALE = 4
CANVAS = SIZE * SCALE
FACTOR = CANVAS / VIEWBOX
STROKE = round(2.2 * FACTOR)
INACTIVE = "#454a54"
ACTIVE = "#c23a00"

def xy(value):
    return int(round(value * FACTOR))

def p(point):
    return (xy(point[0]), xy(point[1]))

def box(left, top, right, bottom):
    return (xy(left), xy(top), xy(right), xy(bottom))

def draw_polyline(draw, points, color, close=False):
    pts = [p(point) for point in points]
    if close:
        pts.append(pts[0])
    draw.line(pts, fill=color, width=STROKE, joint="curve")

def draw_centered_line(draw, points, color):
    draw.line([p(point) for point in points], fill=color, width=STROKE, joint="curve")

def new_icon():
    return Image.new("RGBA", (CANVAS, CANVAS), (0, 0, 0, 0))

def save_icon(name, color, drawer):
    image = new_icon()
    draw = ImageDraw.Draw(image)
    drawer(draw, color)
    image = image.resize((SIZE, SIZE), Image.Resampling.LANCZOS)
    image.save(OUT / name)

def home(draw, color):
    draw_polyline(draw, [(5, 14.5), (16, 6), (27, 14.5)], color)
    draw_polyline(draw, [(9, 13.5), (9, 26), (23, 26), (23, 13.5)], color)
    draw_polyline(draw, [(13, 26), (13, 19), (19, 19), (19, 26)], color)

def search(draw, color):
    draw.rounded_rectangle(box(6, 7, 20, 25), radius=xy(3), outline=color, width=STROKE)
    draw_centered_line(draw, [(12, 12), (18, 12)], color)
    draw_centered_line(draw, [(12, 17), (17, 17)], color)
    draw.ellipse(box(17.8, 14.8, 26.2, 23.2), outline=color, width=STROKE)
    draw_centered_line(draw, [(24.8, 22.8), (28, 26)], color)

def publish(draw, color):
    draw.rounded_rectangle(box(5.5, 5.5, 26.5, 26.5), radius=xy(5.5), outline=color, width=STROKE)
    draw_centered_line(draw, [(16, 10.5), (16, 21.5)], color)
    draw_centered_line(draw, [(10.5, 16), (21.5, 16)], color)

def messages(draw, color):
    draw_polyline(
        draw,
        [(7, 9), (25, 9), (27, 11), (27, 20), (25, 22), (15, 22), (9, 26), (9, 22), (7, 22), (5, 20), (5, 11), (7, 9)],
        color,
    )
    draw_centered_line(draw, [(10, 14), (22, 14)], color)
    draw_centered_line(draw, [(10, 18), (18, 18)], color)

def my(draw, color):
    draw.ellipse(box(10, 6.5, 22, 18.5), outline=color, width=STROKE)
    points = []
    for index in range(25):
        x = 6.5 + index * (19 / 24)
        y = 26.5 - 8 * (1 - ((x - 16) / 9.5) ** 2)
        points.append((x, y))
    draw_centered_line(draw, points, color)

ICONS = {
    "home": home,
    "search": search,
    "publish": publish,
    "messages": messages,
    "my": my,
}

for name, drawer in ICONS.items():
    save_icon(f"{name}.png", INACTIVE, drawer)
    save_icon(f"{name}-active.png", ACTIVE, drawer)
PY
```

Expected: command exits 0 and overwrites the 10 files in `wxapp/static/tabbar`。

- [x] **Step 2: 验证 PNG 尺寸和透明通道**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
from PIL import Image

files = [
    "home.png", "home-active.png",
    "search.png", "search-active.png",
    "publish.png", "publish-active.png",
    "messages.png", "messages-active.png",
    "my.png", "my-active.png",
]

for file_name in files:
    path = Path("wxapp/static/tabbar") / file_name
    image = Image.open(path)
    assert image.size == (81, 81), f"{file_name} size is {image.size}"
    assert image.mode == "RGBA", f"{file_name} mode is {image.mode}"
    alpha = image.getchannel("A")
    assert alpha.getbbox() is not None, f"{file_name} has no visible pixels"
    assert min(alpha.getdata()) == 0, f"{file_name} is not transparent"
print("icons ok")
PY
```

Expected: prints `icons ok`.

## Task 3: 最终检查

**Files:**
- Verify: `wxapp/pages.json`
- Verify: `wxapp/static/tabbar/*.png`

- [x] **Step 1: 查看资源文件元数据**

Run:

```bash
file wxapp/static/tabbar/home.png wxapp/static/tabbar/home-active.png wxapp/static/tabbar/search.png wxapp/static/tabbar/search-active.png wxapp/static/tabbar/publish.png wxapp/static/tabbar/publish-active.png wxapp/static/tabbar/messages.png wxapp/static/tabbar/messages-active.png wxapp/static/tabbar/my.png wxapp/static/tabbar/my-active.png
```

Expected: every line contains `PNG image data, 81 x 81, 8-bit/color RGBA`。

- [x] **Step 2: 查看本次实际改动范围**

Run:

```bash
git status --short wxapp/pages.json wxapp/static/tabbar
```

Expected: only `wxapp/pages.json` and the 10 `wxapp/static/tabbar/*.png` files are modified.

- [x] **Step 3: 生成本地预览图用于人工检查**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

root = Path("wxapp/static/tabbar")
labels = ["Home", "Resource", "Publish", "Messages", "My"]
names = ["home", "search", "publish", "messages", "my"]
canvas = Image.new("RGBA", (720, 220), "white")
draw = ImageDraw.Draw(canvas)
font = ImageFont.load_default()

for row, suffix in enumerate(["", "-active"]):
    y = 28 + row * 96
    draw.text((24, y + 22), "active" if suffix else "normal", fill="#333333", font=font)
    for index, name in enumerate(names):
        icon = Image.open(root / f"{name}{suffix}.png").resize((28, 28), Image.Resampling.LANCZOS)
        x = 130 + index * 110
        canvas.alpha_composite(icon, (x, y))
        draw.text((x, y + 36), labels[index], fill="#c23a00" if suffix else "#454a54", font=font)

out = Path("/private/tmp/wplink-tab-icons-preview.png")
canvas.save(out)
print(out)
PY
```

Expected: prints `/private/tmp/wplink-tab-icons-preview.png`; use image preview to confirm图标清晰、线条统一、语义可辨。
