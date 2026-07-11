# Qingjian Bookroom Ebook Reader

## Core idea

This project is a lightweight, server-hosted ebook aggregation service: it centralizes accounts, library management, uploads, parsing, reading progress, bookmarks, and notes in one backend so phones, tablets, and desktop browsers can access the same library and reading state over the network.

It is built as a local-first ebook reading system with a Vue 3 frontend, shadcn-vue style components, PocketBase authentication and file storage, plus a Go backend extension. It currently supports PDF, EPUB, and MOBI uploads. PDF files are parsed and rendered with `go-pdfium`; EPUB and MOBI files are parsed with `go-fitz`/MuPDF, and EPUB rendering includes backend handling for CJK fonts, table of contents, and pagination.

## MVP features

- Authentication: PocketBase `users` auth collection.
- Library management: upload, list, view details, and delete PDF/EPUB/MOBI books.
- Backend parsing: after upload, Go hooks extract page counts, TOC data, and per-page text, then write them into `books.toc` and `book_pages`.
- Reading: the frontend requests rendered PNG pages from the backend and shows extracted page text; it supports TOC/page jumps, click-to-zoom, and drag-to-pan.
- Bookmarks: add a bookmark from the reading page and jump back to it later.
- Notes: save page notes from the reader and aggregate them by book on the notes page.
- Book metadata: display page count, parse status, current page, and error state.
- Reading progress: record current page, progress, and reading time.

## Quick start

```bash
cp .env.example .env
task setup
task backend
# New terminal: initialize schema and demo users
task bootstrap
# New terminal: start the frontend in development mode
task dev-frontend
```

Demo credentials:

```text
demo@reader.local / ebook-reader-user-123
```

Backend default address: `http://127.0.0.1:8090`.

To deploy it as `cicd-local-ebook-reader` in `~/wp/cicd-uzqw`:

```bash
task docker-deploy:cicd
```

If the local `cicd-observability` network does not exist yet, start `~/wp/cicd-uzqw/docker-compose.local.yml` first to create the shared network.

After deployment, the service is available at:

```text
http://127.0.0.1:18094
http://127.0.0.1:18094/api/health
http://127.0.0.1:18094/metrics
```

To integrate with `~/wp/cicd-uzqw/`, copy these files into that project:

```text
../cicd-uzqw/caddy.d/ebook-reader.caddy
../cicd-uzqw/scrape-targets/ebook-reader.yml
```

## Common commands

```bash
task setup        # npm install + build the PocketBase Go backend
task backend      # start the PocketBase extension
task bootstrap    # create collections, superuser, and demo user
task dev-frontend # Vite development server
task check        # vue-tsc + go test
task build        # frontend production build + backend build
task docker-deploy:cicd # deploy through Docker Compose as cicd-local-ebook-reader
```

## Architecture

```text
Browser
  -> Vite dev server / dist
  -> PocketBase REST API
  -> custom route /api/books/{id}/pages/{page}/image?token=...

PocketBase
  -> users auth
  -> books file storage
  -> book_pages/bookmarks/notes/reading_records collections

Go extension
  -> OnRecordAfterCreateSuccess("books")
  -> PDF: go-pdfium WebAssembly parses outline/page text and renders PNG
  -> EPUB/MOBI: go-fitz/MuPDF parses page text and renders PNG
  -> EPUB: injects a CJK font before parse/render so text and image pagination stay aligned
```

## Directory layout

```text
src/                         Vue frontend source
src/components/ui/           shadcn-vue-style base components
src/views/                   library, upload, reader, notes, and summary pages
backend/cmd/ebook-pocketbase PocketBase Go extension backend
scripts/bootstrap-pocketbase.mjs
                             idempotent PocketBase schema/account bootstrap
omx_wiki/                    Chinese project knowledge base
arch-reference -> ...        local reference symlink
```

## Parsing and rendering notes

### PDF

The backend uses the WebAssembly implementation of `go-pdfium`, so no native PDFium dynamic library is required on the host. The parsing flow reads page count, PDF outline TOC, per-page text, and page dimensions; the rendering route outputs PNG on demand and validates the PocketBase auth token so users can only access their own books.

The reading page image quality is controlled by `PDF_RENDER_DPI`, which defaults to `220`. You can raise it to `260` or `300` in `.env` and restart the backend; higher DPI means slower rendering and larger images. The upper safety limit is `360`.

### EPUB / MOBI

EPUB/MOBI are parsed and rendered through `go-fitz`/MuPDF. Notes:

- EPUB 3.0 may log `warning: unknown epub version: 3.0`; this is a non-fatal MuPDF warning, not a parse failure.
- `go-fitz` bundled MuPDF does not ship with CJK fonts by default; the backend injects a CJK font and CSS into a temporary EPUB copy before parsing/rendering.
- Text parsing and image rendering must use the same font-injected EPUB copy; otherwise pagination changes and page text will no longer match the rendered image.
- EPUB TOC parsing prefers EPUB 3 `nav.xhtml`; if that is unavailable, it falls back to `toc.ncx`; only then does it fall back to `go-fitz` `ToC()`.
- After upload, the frontend waits for parsing to settle before entering the reader so users do not confuse “processing” with “TOC missing”.
- `go-fitz`/MuPDF native calls are not safe to run concurrently, so the backend serializes EPUB/MOBI fitz operations with a global mutex to avoid SIGSEGV.

Optional environment variables:

```bash
# CJK font path used for EPUB rendering; if unset, common system font paths are tried
EPUB_RENDER_FONT=/usr/share/fonts/droid/DroidSansFallback.ttf
# Temporary file directory; defaults to the mounted /app/pb_data/tmp
TMPDIR=/app/pb_data/tmp
```

If the deployment environment does not have Droid/Noto CJK fonts, install one or set `EPUB_RENDER_FONT` explicitly. `TMPDIR` must point to a writable directory inside the mounted `pb_data/tmp` volume.

## Font and license

The `fonts/DroidSansFallback.ttf` file is used for EPUB/CJK rendering. It comes from Android open-source font resources and is typically distributed under Apache License 2.0. It is kept in the repository to ensure consistent rendering across devices.

## Knowledge base

Project knowledge is stored in `omx_wiki/`:

- `项目总览.md`: product scope and non-goals.
- `架构.md`: frontend/backend architecture and parsing flow.
- `运行手册.md`: startup, verification, and troubleshooting.
- `epub-渲染排障-cjk-字体与-go-fitz-并发.md`: EPUB CJK font, pagination consistency, and native crash debugging.
- `epub-原生目录解析约定.md`: EPUB `nav.xhtml` / `toc.ncx` TOC parsing and real page mapping.
- `数据模型与解析状态.md`: PocketBase collections, parse state, and field conventions.
