---
title: "EPUB 渲染排障：CJK 字体与 go-fitz 并发"
tags: ["epub", "go-fitz", "mupdf", "cjk", "rendering", "panic", "pagination"]
created: 2026-05-30T17:27:36.418Z
updated: 2026-05-31T02:11:29+08:00
sources: []
links: []
category: debugging
confidence: high
schemaVersion: 1
---

# EPUB 渲染排障：CJK 字体与 go-fitz 并发

## 背景

EPUB 上传后，后端用 `go-fitz`/MuPDF 解析页面文本，并通过 `/api/books/{id}/pages/{page}/image` 返回页面 PNG。

## 典型现象

- 日志反复出现 `warning: unknown epub version: 3.0`。
- `book_pages.text` / 阅读页“本页文本”有内容。
- EPUB 页面图片缺中文，常表现为只有英文、链接下划线或大片空白。
- 图片和“本页文本”页码对不上。
- 上传后立即进入阅读页时，后台解析和图片渲染并发调用 `go-fitz`，可能触发 cgo native SIGSEGV，Go `recover()` 无法捕获。

## 根因

1. `warning: unknown epub version: 3.0` 是 MuPDF 对 EPUB3 的非致命警告，不代表解析失败。
2. `github.com/gen2brain/go-fitz` bundled MuPDF 库默认不带 CJK 字体；如果 EPUB 没有可用中文字体，图片渲染缺中文字形。
3. EPUB 是 reflowable 格式。注入字体/CSS 会改变 MuPDF 分页；如果图片渲染使用“注入字体后的 EPUB”，但文本解析使用原始 EPUB，就会出现图片和“本页文本”页码错位。
4. `go-fitz`/MuPDF native 层在本项目 EPUB/MOBI 场景下不能安全并发；并发 `fitz.New`/`NumPage`/`Text`/`ImagePNG` 可导致进程级 SIGSEGV。

## 当前约定

- EPUB/MOBI 仍应由后端渲染 PNG 返回给前端；不要把 EPUB 简化成纯文本模式。
- 前端“本页文本”只是辅助展示，不应替代 EPUB 图片渲染能力。
- EPUB 文本解析和图片渲染必须使用同一份注入 CJK 字体后的临时 EPUB。
- 后端所有 `go-fitz` 入口必须通过全局互斥串行化，避免 native 并发崩溃。

## 实现位置

- `backend/cmd/ebook-pocketbase/main.go`
  - `fitzMu sync.Mutex`：串行化 EPUB/MOBI 的 go-fitz 解析与渲染。
  - `parseFitzBook`：EPUB 解析前调用 `injectEPUBRenderFont`，确保 `book_pages.text` 和图片分页一致。
  - `renderFitzPagePNG`：EPUB 渲染前调用同一套字体注入逻辑。
  - `injectEPUBRenderFont`：重写临时 EPUB zip，在 XHTML/HTML 中注入 `@font-face` 和 `font-family`，并添加字体文件。
  - `cjkRenderFontPath`：字体查找顺序为 `EPUB_RENDER_FONT`、DroidSansFallback、Noto CJK。

## 运维注意

- 部署环境应安装可用 CJK 字体，或设置 `EPUB_RENDER_FONT=/path/to/font.ttf`。
- 推荐字体：`/usr/share/fonts/droid/DroidSansFallback.ttf`、`/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc`。
- 如果 EPUB 图片仍缺中文，先确认字体文件存在，再检查后端是否已重启并运行最新构建。
- 如果 EPUB 图片和“本页文本”对不上，检查解析和渲染是否都经过 `injectEPUBRenderFont`。
- 如果旧 EPUB 记录停在 `processing`，通常是之前 native 崩溃遗留；删除后重新上传或补跑解析。

## 验证

- 本地用已上传 EPUB 复现：修复前 `ImagePNG` 输出图片缺中文；注入 CJK 字体后生成的 PNG 能显示中文。
- 重新解析后，`book_pages.text` 与页面 PNG 应来自同一分页结果。
- 常规验证命令：

```bash
npm run check
cd backend && go test ./...
cd backend && go build -o /tmp/ebook-pocketbase-check ./cmd/ebook-pocketbase
npm run build
```

## 相关页面

- [[epub-原生目录解析约定]]
- [[运行手册]]
