import fs from 'node:fs'
import path from 'node:path'
import { execFile } from 'node:child_process'
import { promisify } from 'node:util'
import PocketBase from 'pocketbase'

const execFileAsync = promisify(execFile)
const cwd = process.cwd()
function loadDotEnv(filePath) {
  if (!fs.existsSync(filePath)) return
  for (const line of fs.readFileSync(filePath, 'utf8').split(/\r?\n/)) {
    const trimmed = line.trim(); if (!trimmed || trimmed.startsWith('#')) continue
    const eq = trimmed.indexOf('='); if (eq <= 0) continue
    const key = trimmed.slice(0, eq).trim(); let value = trimmed.slice(eq + 1).trim()
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) value = value.slice(1, -1)
    if (!(key in process.env)) process.env[key] = value
  }
}
loadDotEnv(path.join(cwd, '.env'))

const config = {
  url: process.env.POCKETBASE_URL_INTERNAL || `http://${process.env.POCKETBASE_HOST || '127.0.0.1'}:${process.env.POCKETBASE_PORT || '8090'}`,
  pbBin: process.env.PB_BIN || path.join(cwd, 'backend', 'ebook-pocketbase'),
  pbDataDir: process.env.POCKETBASE_DATA_DIR || path.join('.local', 'pb_data'),
  superuserEmail: process.env.PB_SUPERUSER_EMAIL || 'admin@reader.local',
  superuserPassword: process.env.PB_SUPERUSER_PASSWORD || 'ebook-reader-admin-123',
  appUserName: process.env.APP_USER_NAME || 'Reader Demo User',
  appUserEmail: process.env.APP_USER_EMAIL || 'demo@reader.local',
  appUserPassword: process.env.APP_USER_PASSWORD || 'ebook-reader-user-123',
}
const pb = new PocketBase(config.url)
pb.autoCancellation(false)

const createdField = { hidden: false, name: 'created', onCreate: true, onUpdate: false, presentable: false, system: false, type: 'autodate' }
const updatedField = { hidden: false, name: 'updated', onCreate: true, onUpdate: true, presentable: false, system: false, type: 'autodate' }
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
async function waitForServer() { let last; for (let i = 0; i < 30; i++) { try { const r = await fetch(`${config.url}/api/health`); if (r.ok) return } catch (e) { last = e } await sleep(1000) } throw new Error(`PocketBase not ready: ${last?.message || 'unknown'}`) }
async function authSuperuser() { await pb.collection('_superusers').authWithPassword(config.superuserEmail, config.superuserPassword) }
async function ensureSuperuser() { try { await authSuperuser(); return } catch {}
  await execFileAsync(config.pbBin, ['superuser', 'upsert', config.superuserEmail, config.superuserPassword, '--dir', config.pbDataDir])
  await sleep(500); await authSuperuser()
}
function normalizeFields(fields) { return Array.isArray(fields) ? fields.map((f) => ({ ...f })) : [] }
function withBaseTimestampFields(definition) {
  if (definition.type !== 'base') return definition
  const fields = normalizeFields(definition.fields)
  if (!fields.some((f) => f.name === 'created')) fields.push(createdField)
  if (!fields.some((f) => f.name === 'updated')) fields.push(updatedField)
  return { ...definition, fields }
}

async function upsertCollection(definition) {
  definition = withBaseTimestampFields(definition)
  const found = await pb.collections.getFullList({ filter: `name = "${definition.name}"`, requestKey: null })
  if (!found.length) { console.log(`create ${definition.name}`); return pb.collections.create(definition) }
  const current = found[0]
  const merged = normalizeFields(current.fields)
  let changed = false
  for (const field of normalizeFields(definition.fields)) {
    const idx = merged.findIndex((f) => f.name === field.name)
    if (idx === -1) {
      merged.push(field)
      changed = true
    } else {
      // Ensure specific field config properties like mimeTypes are updated
      const currentFieldStr = JSON.stringify(merged[idx])
      const targetFieldStr = JSON.stringify({ ...merged[idx], ...field })
      if (currentFieldStr !== targetFieldStr) {
        merged[idx] = { ...merged[idx], ...field }
        changed = true
      }
    }
  }
  if (definition.type === 'base') {
    if (!merged.some((f) => f.name === 'created')) { merged.push(createdField); changed = true }
    if (!merged.some((f) => f.name === 'updated')) { merged.push(updatedField); changed = true }
  }
  const rulesChanged = ['listRule','viewRule','createRule','updateRule','deleteRule','authRule','manageRule'].some((k) => (definition[k] ?? null) !== (current[k] ?? null))
  if (changed || rulesChanged) { console.log(`update ${definition.name}`); return pb.collections.update(current.id, { ...current, ...definition, fields: merged }) }
  console.log(`skip ${definition.name}`); return current
}
const rel = (name, collectionId, required = true) => ({ name, type: 'relation', required, maxSelect: 1, collectionId, cascadeDelete: false })
const text = (name, required = false, max = 0) => ({ name, type: 'text', required, max })
const num = (name, required = false) => ({ name, type: 'number', required, min: null, max: null, onlyInt: false })
async function ensureCollections() {
  const users = await upsertCollection({ name: 'users', type: 'auth', listRule: 'id = @request.auth.id', viewRule: 'id = @request.auth.id', createRule: '', updateRule: 'id = @request.auth.id', deleteRule: 'id = @request.auth.id', authRule: '', manageRule: null, passwordAuth: { enabled: true, identityFields: ['email'] }, fields: [text('name', true, 120)] })
  const books = await upsertCollection({ name: 'books', type: 'base', listRule: 'user = @request.auth.id', viewRule: 'user = @request.auth.id', createRule: '@request.auth.id != "" && user = @request.auth.id', updateRule: 'user = @request.auth.id', deleteRule: 'user = @request.auth.id', fields: [rel('user', users.id), text('title', true, 240), text('author', false, 180), { name: 'description', type: 'editor', required: false, maxSize: 20000 }, { name: 'file', type: 'file', required: true, maxSelect: 1, maxSize: 104857600, mimeTypes: ['application/pdf', 'application/epub+zip', 'application/x-mobipocket-ebook'] }, num('page_count'), { name: 'parse_status', type: 'select', required: true, maxSelect: 1, values: ['pending','processing','completed','failed'] }, text('parse_error', false, 1000), num('current_page'), { name: 'toc', type: 'json', required: false, maxSize: 200000 }, { name: 'last_read_at', type: 'date', required: false } ] })
  await upsertCollection({ name: 'book_pages', type: 'base', listRule: 'book.user = @request.auth.id', viewRule: 'book.user = @request.auth.id', createRule: null, updateRule: null, deleteRule: null, fields: [rel('book', books.id), num('page_number', true), { name: 'text', type: 'editor', required: false, maxSize: 200000 }, num('width'), num('height')] })
  await upsertCollection({ name: 'bookmarks', type: 'base', listRule: 'user = @request.auth.id', viewRule: 'user = @request.auth.id', createRule: '@request.auth.id != "" && user = @request.auth.id && book.user = @request.auth.id', updateRule: 'user = @request.auth.id', deleteRule: 'user = @request.auth.id', fields: [rel('book', books.id), rel('user', users.id), num('page_number', true), text('title', true, 200), text('note', false, 1000)] })
  await upsertCollection({ name: 'notes', type: 'base', listRule: 'user = @request.auth.id', viewRule: 'user = @request.auth.id', createRule: '@request.auth.id != "" && user = @request.auth.id && book.user = @request.auth.id', updateRule: 'user = @request.auth.id', deleteRule: 'user = @request.auth.id', fields: [rel('book', books.id), rel('user', users.id), num('page_number', true), { name: 'content', type: 'editor', required: true, maxSize: 50000 }] })
  await upsertCollection({ name: 'reading_records', type: 'base', listRule: 'user = @request.auth.id', viewRule: 'user = @request.auth.id', createRule: '@request.auth.id != "" && user = @request.auth.id && book.user = @request.auth.id', updateRule: 'user = @request.auth.id', deleteRule: 'user = @request.auth.id', fields: [rel('book', books.id), rel('user', users.id), num('page_number', true), num('progress'), num('read_seconds')] })
}
async function ensureDemoUser() { try { await pb.collection('users').getFirstListItem(`email = "${config.appUserEmail}"`); return } catch {} await pb.collection('users').create({ name: config.appUserName, email: config.appUserEmail, password: config.appUserPassword, passwordConfirm: config.appUserPassword, verified: true }); console.log(`demo user: ${config.appUserEmail}`) }
await waitForServer(); await ensureSuperuser(); await ensureCollections(); await ensureDemoUser(); console.log('PocketBase schema ready')
