# AimMeow 品牌素材

认可参考：2026-10-04 的「瞄瞄品牌板：石墨灰 AK」。正式素材使用内置 imagegen 生成。

## 使用位置

- `frontend/src/assets/brand/aimmeow-mascot-master.png`：透明主形象原图。
- `aimmeow-mascot-512.png`：欢迎页、训练工作台与 README。
- `aimmeow-icon-master.png`：应用图标原图。
- `aimmeow-icon-{32,128,256,512}.png`：页面图标、侧栏及发行素材。
- `build/appicon.png`：Wails 应用图标。
- `build/windows/icon.ico`：16、24、32、48、64、128、256 像素 Windows 多尺寸图标。

品牌色：柔紫 `#A69BFF`、石墨灰 `#17181F`、奶白 `#F1EFE9`、灰紫 `#77748B`。深色按钮使用石墨灰文字；浅色主题的主色为较深的 `#6B52C7`，确保白字和链接可读。状态与辅助图表颜色保留语义区分。用户自定义主题仍可覆盖默认配色。

素材通过 Vite 本地打包，不依赖在线图床。修改主图后可运行 `python3 scripts/prepare_brand_assets.py` 重建尺寸与 ICO（需要 Pillow）。该脚本只进行尺寸及格式转换，不修改设计。

## 验证记录

2026-10-05：TypeScript / Vite 生产构建与 8 项既有前端回归通过；运行时 PNG 的尺寸和透明通道、Windows ICO 的 7 个尺寸通过检查。128 像素图标已目视检查。深色主色按钮文字对比度 7.38:1，浅色主色按钮 5.74:1。

本轮尚未完成完整界面截图和 Windows 打包验证。

## 生成提示词

### 透明主形象

```text
Use case: logo-brand
Asset type: production transparent mascot mark for the AimMeow / 瞄瞄 desktop application.
Input image: approved brand board, use the LARGE CAT at left as exact identity and style reference, not the UI or lettering.
Primary request: Extract and faithfully redraw ONLY the approved black cat holding the graphite-grey AK with both small front paws, as one isolated high-resolution transparent PNG. Keep the same triangular ears with lavender inner ears, thick lavender contour, large ivory focused eyes looking forward, expressive angled brows, little lavender nose, ivory whiskers, round paws, short claw details and the same AK pose with stock to left and barrel to right. The cat head is the main silhouette. The AK is dark graphite with subtle grey-lavender detailing, NO brown or warm wood. Preserve the proportions and charming determined expression in the reference. Clean sticker-like vector-friendly illustration with subtle shading, bold smooth contours.
Composition: tightly centered square canvas, entire mascot including ears, whiskers, stock, barrel and curved magazine visible, 6% clear margin. Cat and weapon should fill the frame naturally. No lettering, no Chinese characters, no branding board, no UI, no frame, no tile background, no shadows outside the subject, no decorative sparks. Actual transparent background; clean anti-aliased alpha edges.
Palette: black/graphite #17181F, lavender #A69BFF, ivory #F1EFE9, muted grey-purple #77748B.
```

### 应用图标

```text
Use case: logo-brand
Asset type: AimMeow Windows application icon, square production PNG.
Input image: finalized transparent mascot. Use this exact cat identity and pose.
Primary request: Make a polished square application icon using the SAME black cat and graphite AK from the reference, centered on a deep graphite #17181F softly rounded-square tile with a restrained grey-lavender top edge highlight. The cat face dominates; keep both paws and the AK with stock left, barrel right and curved magazine visible. Same ivory focused eyes, lavender ear inserts and lavender contour; preserve the approved determined charming expression. Slightly simplify very fine weapon details and whiskers for small-size readability; bold clear silhouettes. Cat and gun fully within the tile with enough breathing room, no cropping. Tile fills about 94% of the square, rounded corner exterior genuinely transparent. Calm professional desktop-app icon, no neon glow, no letters, no extra symbols, no brown wood. Palette #A69BFF lavender, #17181F graphite, #F1EFE9 ivory, #77748B grey-purple. Output ONE icon only, not an icon sheet or mockup.
```
