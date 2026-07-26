# ebook

> A lightweight, server-hosted ebook reader with a Vue frontend, a PocketBase-based backend, and multi-device reading over the network.

## Deployment

Recommended flow:

```bash
cp .env.example .env
task deploy
```

Deployment stores PocketBase data under
`${APP_DATA_ROOT}/ebook-reader/pb_data`. Set `APP_DATA_ROOT` to an absolute
application data root, or leave it unset to use the XDG user default:

```text
${XDG_DATA_HOME:-$HOME/.local/share}/uzqw/apps
```

The deployment path resolver rejects repository-local and unsafe roots. Path
parameterization does not migrate existing data; follow the migration runbook
before deploying against a new data directory.

For local development, run the foreground services with:

```bash
task setup
task doctor
task dev
```

Run `task bootstrap` from another terminal after the backend is healthy.

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
task setup       # install locked frontend and Go dependencies
task doctor      # check the runtime environment without modifying it
task dev         # run the frontend and backend in the foreground
task bootstrap   # create collections and demo users
task lint        # run the current static checks
task test        # run frontend and backend tests
task check       # run the CI aggregate checks
task build       # build production frontend and backend artifacts
task deploy:down # stop deployment without deleting persistent data
task deploy:logs # show deployment logs
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
