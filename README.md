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

### Internal HTTPS certificate

The deployed `https://${PLATFORM_HOST}:18094` endpoint uses Caddy's internal CA.
Every client must trust the gateway's **root CA**; merely bypassing the browser
warning for one leaf certificate is not sufficient. Caddy rotates its default
12-hour leaf certificates, and an untrusted replacement makes in-page requests
fail with `Failed to fetch` until a top-level navigation handles the certificate
again.

For the local platform deployment, download the public root certificate over the
non-TLS gateway endpoint, install it in the client operating system, and restart
the browser:

```bash
curl http://${PLATFORM_HOST}:18089/caddy-local-root.crt \
  -o caddy-local-root.crt
```

Install it on Arch/Manjaro Linux:

```bash
sudo trust anchor --store caddy-local-root.crt
sudo update-ca-trust
```

Or on Debian/Ubuntu Linux:

```bash
sudo install -m 0644 caddy-local-root.crt \
  /usr/local/share/ca-certificates/caddy-local-root.crt
sudo update-ca-certificates
```

Fully quit and reopen the browser after changing the trust store. See the
platform `ACCESS.md` for macOS, Fedora/RHEL, Chrome/Chromium NSS, and Firefox
instructions. Preserve the platform Caddy data directory; replacing it creates
a new root CA that must be installed on every client again.

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
task fmt:check   # verify frontend and Go formatting
task lint        # run frontend, shell, and Go linters
task typecheck   # run frontend type checking
task test        # run frontend and backend tests
task build       # build production frontend and backend artifacts
task config      # validate manifest and merged Compose configuration
task check       # run all checks above in the documented order
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
