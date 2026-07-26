import { readFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'
import { parseDocument } from 'yaml'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '..')

function fail(message) {
  console.error(`CONFIG_ERROR ${message}`)
  process.exit(1)
}

function assert(condition, message) {
  if (!condition) fail(message)
}

function equal(actual, expected, field) {
  assert(actual === expected, `${field} expected=${expected} actual=${actual}`)
}

const manifestPath = path.join(repoRoot, 'ops/platform/app.yaml')
const manifestDocument = parseDocument(readFileSync(manifestPath, 'utf8'), {
  uniqueKeys: true,
})
assert(manifestDocument.errors.length === 0, 'manifest YAML is invalid')
const manifest = manifestDocument.toJS()

equal(manifest.apiVersion, 'platform.uzqw/v1', 'apiVersion')
equal(manifest.kind, 'Application', 'kind')
equal(manifest.metadata?.id, 'ebook-reader', 'metadata.id')
equal(manifest.deployment?.type, 'docker-compose', 'deployment.type')
equal(manifest.deployment?.managedBy, 'application', 'deployment.managedBy')
assert(
  JSON.stringify(manifest.deployment?.composeFiles) ===
    JSON.stringify(['compose.yaml', 'compose.build.yaml', 'compose.observability.yaml']),
  'deployment.composeFiles must preserve the validated merge order',
)
equal(manifest.deployment?.workflow, '.gitea/workflows/ci.yml', 'deployment.workflow')
const workflowDocument = parseDocument(
  readFileSync(path.join(repoRoot, manifest.deployment.workflow), 'utf8'),
  { uniqueKeys: true },
)
assert(workflowDocument.errors.length === 0, 'CI workflow YAML is invalid')
equal(manifest.network?.name, 'cicd-observability', 'network.name')
equal(manifest.network?.external, true, 'network.external')

const web = manifest.services?.find((service) => service.id === 'web')
assert(web, 'services.web is required')
equal(web.composeService, 'ebook-reader-uzqw', 'services.web.composeService')
equal(web.dnsName, 'ebook-reader-uzqw', 'services.web.dnsName')
equal(web.port, 18093, 'services.web.port')
equal(web.protocol, 'http', 'services.web.protocol')

const readiness = manifest.healthChecks?.find((check) => check.id === 'readiness')
assert(readiness, 'healthChecks.readiness is required')
equal(readiness.service, 'web', 'healthChecks.readiness.service')
equal(readiness.path, '/api/health', 'healthChecks.readiness.path')

const route = manifest.caddy?.routes?.find((item) => item.id === 'public')
assert(route, 'caddy.routes.public is required')
equal(route.listener?.host, '${PLATFORM_HOST}', 'caddy.routes.public.listener.host')
equal(route.listener?.port, 18094, 'caddy.routes.public.listener.port')
equal(route.upstream?.service, 'web', 'caddy.routes.public.upstream.service')

const metric = manifest.metrics?.find((item) => item.id === 'application')
assert(metric, 'metrics.application is required')
equal(metric.service, 'web', 'metrics.application.service')
equal(metric.path, '/metrics', 'metrics.application.path')
equal(manifest.logs?.labels?.['uzqw.app'], 'ebook-reader', 'logs.labels.uzqw.app')

const data = manifest.data?.find((item) => item.id === 'pocketbase')
assert(data, 'data.pocketbase is required')
equal(data.path, '${APP_DATA_ROOT}/ebook-reader/pb_data', 'data.pocketbase.path')
equal(data.lifecycle, 'retain', 'data.pocketbase.lifecycle')
equal(data.backup?.strategy, 'application-consistent', 'data.pocketbase.backup.strategy')

const caddyPath = path.join(repoRoot, manifest.caddy.source)
const caddySource = readFileSync(caddyPath, 'utf8')
assert(caddySource.includes('https://${PLATFORM_HOST}:18094'), 'Caddy listener claim mismatch')
assert(caddySource.includes('reverse_proxy ebook-reader-uzqw:18093'), 'Caddy upstream mismatch')
assert(!caddySource.includes('cicd-uzqw'), 'Caddy source references the cicd repository')
assert(!/\b10(?:\.\d{1,3}){3}\b/.test(caddySource), 'Caddy source contains a fixed host IP')

const fixtureDataPath = path.join(tmpdir(), 'ebook-reader-config-check', 'pb_data')
const compose = spawnSync(
  'docker',
  [
    'compose',
    '-p',
    'ebook-reader-uzqw',
    '-f',
    'compose.yaml',
    '-f',
    'compose.build.yaml',
    '-f',
    'compose.observability.yaml',
    'config',
    '--format',
    'json',
  ],
  {
    cwd: repoRoot,
    encoding: 'utf8',
    env: { ...process.env, DOCKER_PB_DATA_PATH: fixtureDataPath },
  },
)
assert(compose.status === 0, `Compose config failed: ${compose.stderr.trim()}`)
const composeConfig = JSON.parse(compose.stdout)
const composeService = composeConfig.services?.['ebook-reader-uzqw']
assert(composeService, 'Compose service ebook-reader-uzqw is required')
assert(!composeService.container_name, 'Compose must not set container_name')
assert(
  Object.hasOwn(composeService.networks ?? {}, 'cicd-observability'),
  'Compose service must join cicd-observability',
)
equal(composeConfig.networks?.['cicd-observability']?.external, true, 'Compose network.external')
equal(composeService.labels?.['uzqw.observe'], 'true', 'Compose labels.uzqw.observe')
equal(composeService.labels?.['uzqw.app'], 'ebook-reader', 'Compose labels.uzqw.app')

const dataMount = composeService.volumes?.find((volume) => volume.target === '/app/pb_data')
assert(dataMount, 'Compose PocketBase data mount is required')
equal(dataMount.type, 'bind', 'Compose PocketBase volume.type')
equal(dataMount.source, fixtureDataPath, 'Compose PocketBase volume.source')

console.log('PASS config validation')
