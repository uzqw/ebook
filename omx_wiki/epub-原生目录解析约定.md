---
title: "EPUB 原生目录解析约定"
tags: ["epub", "toc", "nav.xhtml", "ncx", "opf", "go-fitz", "pagination"]
created: 2026-05-30T17:48:46.045Z
updated: 2026-05-31T02:15:00+08:00
sources: []
links: []
category: debugging
confidence: high
schemaVersion: 1
---

# EPUB 原生目录解析约定

## 背景

某些 EPUB 文件本身有完整目录，但 `go-fitz` 的 `doc.ToC()` 只返回退化目录，甚至直接丢失原生目录。项目里出现过两类样本：

- EPUB3：`nav.xhtml` 才是主目录，`toc.ncx` 只是兼容占位。
- EPUB2 / 兼容样本：只有 `toc.ncx`，没有可用的 `nav.xhtml`。

如果只依赖 `go-fitz` 的 `ToC()`，就会出现“目录为空”“目录页码乱”“只有 Start”这些问题。

## 根因

EPUB3 的主目录通常在 manifest 中标记 `properties="nav"` 的 XHTML 文件里，而 EPUB2 常用的是 NCX。两种目录形态都需要支持，否则最近上传的 EPUB 很容易只拿到空目录或占位目录。

另一个陷阱是页码：EPUB 是 reflowable 格式，OPF spine 顺序不是 MuPDF 渲染页码。字体注入后还会重新分页，所以目录页码必须在最终解析用的 EPUB 版本上定位。

## 当前实现约定

- EPUB 解析时优先解析原生 `nav.xhtml` 目录；如果没有可用 nav，再回退到 `toc.ncx`。
- 解析流程：
  1. 读取 `META-INF/container.xml` 找 OPF rootfile。
  2. 解析 OPF manifest，找到 `properties` 包含 `nav` 的 item；同时记录 NCX manifest item 或 spine 的 `toc` 引用。
  3. 解析 nav.xhtml 中 `epub:type="toc"` 的 `<nav>`，递归读取 `<ol>/<li>/<a>`，得到目录标题和层级。
  4. 如果没有可用 nav，再解析 `toc.ncx` 的 `<navMap>/<navPoint>`，得到目录标题和层级。
  5. EPUB 解析前先注入 CJK 字体；目录真实页码也在该注入字体后的文档上定位。
  6. 使用目录标题在 MuPDF 提取的页面文本中反查真实页码，并跳过目录页本身，避免命中 `Table of Contents`。
  7. 将目录写入 `books.toc`，字段形状仍是 `{title,page,level}`。
- 如果 EPUB 原生目录都解析不到，再 fallback 到 go-fitz `doc.ToC()`。

## 代码位置

- `backend/cmd/ebook-pocketbase/main.go`
  - `parseEPUBTOC`
  - `epubRootfilePath`
  - `parseEPUBManifestAndSpine`
  - `parseEPUBNavTOC`
  - `resolveEPUBPath`
  - `tocTitleCandidates`
  - `resolveEPUBTOCPages`

## 验证样例

本地已上传 EPUB 修复前：`books.toc` 只有 1 条 `Start` 或直接为空。修复后：

- 有 `nav.xhtml` 的 EPUB：解析出完整目录，并在注入 CJK 字体后的真实分页上定位。
- 只有 `toc.ncx` 的 EPUB：也能解析出目录并回填页码。

例如 `The Book of Joy` 解析后能得到 39 条目录，并在注入 CJK 字体后的真实分页上定位：

- `序│與我們一同感受喜悅` → 第 6 页
- `ARRIVAL│我們都是脆弱的` → 第 14 页
- `DAY 1│喜悅的本質` → 第 27 页
- `DAY 2 & 3│那些讓喜悅遠離的事物` → 第 69 页
- `觀點：遠近高低各不同` → 第 155 页

常规验证：

```bash
npm run check
cd backend && go test ./...
cd backend && go build -o /tmp/ebook-pocketbase-check ./cmd/ebook-pocketbase
npm run build
```

## 相关页面

- [[epub-渲染排障-cjk-字体与-go-fitz-并发]]
- [[运行手册]]
