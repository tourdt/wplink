import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repositoryRoot = path.resolve(scriptDir, '../..')

function readRepositoryFile(relativePath) {
  return fs.readFileSync(path.join(repositoryRoot, relativePath), 'utf8')
}

test('依赖安全门禁在 Makefile 中阻断 high 及以上漏洞', () => {
  const makefile = readRepositoryFile('Makefile')

  assert.match(makefile, /^check-dependencies:/m)
  assert.match(makefile, /npm audit --prefix admin-web --audit-level=high/)
  assert.match(makefile, /npm audit --prefix wxapp --audit-level=high/)
  assert.match(makefile, /go run golang\.org\/x\/vuln\/cmd\/govulncheck@v1\.1\.4 \.\/\.\.\./)
})

test('CI 调用统一依赖安全门禁', () => {
  const workflow = readRepositoryFile('.github/workflows/ci.yml')

  assert.match(workflow, /make check-dependencies/)
  assert.doesNotMatch(workflow, /continue-on-error:\s*true/)
})

test('Dependabot 覆盖两个前端项目和 Go 模块', () => {
  const dependabot = readRepositoryFile('.github/dependabot.yml')

  for (const expected of [
    'package-ecosystem: npm',
    'directory: /admin-web',
    'directory: /wxapp',
    'package-ecosystem: gomod',
    'directory: /backend',
    'interval: weekly',
  ]) {
    assert.match(dependabot, new RegExp(expected))
  }
})

test('小程序对 Alpha 工具链的高危传递依赖使用安全覆盖版本', () => {
  const wxappPackage = JSON.parse(readRepositoryFile('wxapp/package.json'))

  assert.equal(wxappPackage.devDependencies.vite, '6.4.3')
  for (const [packageName, version] of Object.entries({
    '@intlify/core-base': '9.14.5',
    '@intlify/message-compiler': '9.14.5',
    '@intlify/message-resolver': '9.14.5',
    '@intlify/runtime': '9.14.5',
    '@intlify/vue-devtools': '9.14.5',
    'adm-zip': '0.6.0',
    'jpeg-js': '0.4.4',
    'path-to-regexp': '0.1.13',
    'postcss': '8.5.26',
    ws: '8.21.0',
  })) {
    assert.equal(wxappPackage.overrides?.[packageName], version, `wxapp must override ${packageName}`)
  }
})

test('小程序安装配置仅为已知 Alpha 工具链 peer 冲突启用兼容模式', () => {
  const npmrc = readRepositoryFile('wxapp/.npmrc').trim()

  assert.equal(npmrc, 'legacy-peer-deps=true')
})

test('Go 模块与 CI 工具链满足已扫描到的可达漏洞修复下限', () => {
  const goMod = readRepositoryFile('backend/go.mod')
  const workflow = readRepositoryFile('.github/workflows/ci.yml')

  for (const expected of [
    'go 1.25.0',
    'toolchain go1.25.12',
    'github.com/golang-jwt/jwt/v4 v4.5.2',
    'google.golang.org/grpc v1.82.1',
    'go.opentelemetry.io/otel/sdk v1.44.0',
    'go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0',
  ]) {
    assert(goMod.includes(expected), `backend/go.mod must include ${expected}`)
  }
  assert.match(workflow, /go-version: 1\.25\.12/)
})
