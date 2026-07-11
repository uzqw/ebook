# ebook

> A lightweight, server-hosted ebook reader with a Vue frontend, a PocketBase-based backend, and multi-device reading over the network.

## Deployment

Recommended flow:

```bash
cp .env.example .env
task docker-deploy:cicd
```

For local development:

```bash
task setup
task backend
task bootstrap
task dev-frontend
```

Default local backend address:

```text
http://127.0.0.1:8090
```

## Features

- PocketBase-based authentication and file storage.
- Upload, parse, list, and delete PDF, EPUB, and MOBI books.
- Page image rendering and extracted page text for reading.
- Bookmarks, notes, reading progress, and per-book metadata.
- Shared reading state across devices through one deployed backend.

## Development

Requirements:

- Go
- Node.js and npm
- Task

Useful commands:

```bash
task setup        # install frontend deps and build backend
task backend      # run the PocketBase extension
task bootstrap    # create collections and demo users
task dev-frontend # run the Vite frontend
task check        # typecheck and go test
task build        # production frontend + backend build
```

Demo account:

```text
demo@reader.local / ebook-reader-user-123
```

## Repository layout

```text
src/                         Vue frontend
backend/cmd/ebook-pocketbase PocketBase Go extension
pb_hooks/                    PocketBase hooks
scripts/                     bootstrap and container scripts
fonts/                       required CJK font asset for EPUB rendering
```

## Repository hygiene

This repository intentionally excludes or ignores:

- PocketBase runtime data under `.local/` and `backend/pb_data/`
- Local environment files such as `.env`
- Build outputs such as `dist/`
- Dependency directories such as `node_modules/`
- Local agent/runtime metadata such as `.omx/`, `.agents/`, and `omx_wiki/`
- Generated backend binary outputs

## Font asset

`fonts/DroidSansFallback.ttf` is kept in the repository because EPUB/CJK rendering depends on it for consistent output across devices.
