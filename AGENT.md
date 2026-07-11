# 青简书房协作说明

本项目是 `arch-reference` 的同风格新项目：保留浅绿色侧栏、卡片化页面、中文文档沉淀和本地优先 PocketBase 后端。

## 产品边界

MVP 先保证：注册登录 -> 上传 PDF -> 后端解析 -> 前端阅读。书签、笔记、书籍信息、阅读汇总保留可用闭环，但不追求复杂编辑器和全文检索。

## 技术约定

- 前端：Vue 3 + Vite + Tailwind CSS。
- UI：使用 `src/components/ui/*` 中的 shadcn-vue 风格组件，避免引入额外组件库。
- 后端：`backend/cmd/ebook-pocketbase` 是扩展版 PocketBase。
- PDF：只在后端使用 `github.com/klippa-app/go-pdfium`；默认 WebAssembly 实现。
- Schema：通过 `scripts/bootstrap-pocketbase.mjs` 幂等创建/补齐 collections。

## 数据集合

- `users`：认证用户。
- `books`：PDF 文件、书籍元数据、解析状态、当前阅读页。
- `book_pages`：每页页码、尺寸、文本。
- `bookmarks`：用户书签。
- `notes`：用户页面笔记。
- `reading_records`：阅读进度汇总。

## 验证要求

改动后至少运行：

```bash
npm run check
cd backend && go test ./...
```

涉及前端构建时运行 `npm run build`；涉及 PDF 后端时至少验证一次上传解析或页面 PNG 路由。
