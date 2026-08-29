import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repositoryRoot = path.resolve(scriptDir, '../..')

function readRepositoryFile(relativePath) {
  return fs.readFileSync(path.join(repositoryRoot, relativePath), 'utf8')
}

function runCIWorkflowPolicy(workflowPath) {
  return spawnSync('go', ['run', './scripts/ci_policy_yaml', '-workflow', workflowPath], {
    cwd: path.join(repositoryRoot, 'backend'),
    encoding: 'utf8',
  })
}

function runCIWorkflowFixture(t, source) {
  const fixtureDir = fs.mkdtempSync(path.join(os.tmpdir(), 'wplink-ci-policy-'))
  t.after(() => fs.rmSync(fixtureDir, { recursive: true, force: true }))
  const workflowPath = path.join(fixtureDir, 'ci.yml')
  fs.writeFileSync(workflowPath, source)
  return runCIWorkflowPolicy(workflowPath)
}

function parseGoMod(source) {
  const directives = { go: [], toolchain: [], requirements: new Map() }
  let inRequireBlock = false

  for (const rawLine of source.split(/\r?\n/)) {
    const line = rawLine.replace(/\/\/.*$/, '').trim()
    if (!line) continue

    const goDirective = line.match(/^go\s+(\S+)$/)
    if (goDirective) {
      directives.go.push(goDirective[1])
      continue
    }
    const toolchainDirective = line.match(/^toolchain\s+(\S+)$/)
    if (toolchainDirective) {
      directives.toolchain.push(toolchainDirective[1])
      continue
    }
    if (line === 'require (') {
      inRequireBlock = true
      continue
    }
    if (inRequireBlock && line === ')') {
      inRequireBlock = false
      continue
    }

    const singleRequirement = line.match(/^require\s+(\S+)\s+(v\S+)$/)
    const blockRequirement = inRequireBlock ? line.match(/^(\S+)\s+(v\S+)$/) : null
    const requirement = singleRequirement || blockRequirement
    if (requirement) directives.requirements.set(requirement[1], requirement[2])
  }

  return directives
}

function parseMakeTargetRecipes(source) {
  const targets = new Map()
  let currentTarget = null

  for (const rawLine of source.split(/\r?\n/)) {
    const target = rawLine.match(/^([A-Za-z0-9_.-]+)\s*:(?:\s.*)?$/)
    if (target) {
      currentTarget = target[1]
      targets.set(currentTarget, [])
      continue
    }
    if (currentTarget && rawLine.startsWith('\t')) {
      const recipe = rawLine.trim()
      if (recipe && !recipe.startsWith('#')) targets.get(currentTarget).push(recipe)
      continue
    }
    if (rawLine.trim() && !rawLine.trim().startsWith('#')) currentTarget = null
  }

  return targets
}

function assertGoModPolicy(goModSource) {
  const goMod = parseGoMod(goModSource)
  assert.deepEqual(goMod.go, ['1.25.0'])
  assert.deepEqual(goMod.toolchain, ['go1.25.13'])
}

test('依赖安全门禁在 Makefile 中阻断 high 及以上漏洞', () => {
  const makefile = readRepositoryFile('Makefile')
  const recipes = parseMakeTargetRecipes(makefile).get('check-dependencies') || []

  assert.deepEqual(recipes, [
    'cd backend && node --test scripts/dependency_security_policy.test.mjs',
    'npm audit --prefix admin-web --audit-level=high',
    'npm audit --prefix wxapp --audit-level=high',
    'cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...',
  ])
})

test('CI 使用两个安全 Go 工具链 step 并调用一次统一依赖安全门禁', () => {
  const result = runCIWorkflowPolicy(path.join(repositoryRoot, '.github/workflows/ci.yml'))

  assert.equal(result.status, 0, result.stderr)
})

test('注释不能伪装依赖安全门禁接线', () => {
  const makefileFixture = `check-dependencies:\n\t# cd backend && node --test scripts/dependency_security_policy.test.mjs\n\techo cd backend && node --test scripts/dependency_security_policy.test.mjs\n\tnpm audit --prefix admin-web --audit-level=high\n`

  const recipes = parseMakeTargetRecipes(makefileFixture).get('check-dependencies') || []
  assert.equal(recipes.includes('cd backend && node --test scripts/dependency_security_policy.test.mjs'), false)
})

test('YAML 内联 with 降级不能由 env 块标量中的安全文本伪装', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    env:\n      NOTE: |\n        go-version: 1.25.13\n        run: make check-dependencies\n    steps:\n      - name: Go 一\n        uses: actions/setup-go@v5\n        with: { go-version: 1.25.12 }\n      - name: Go 二\n        uses: actions/setup-go@v5\n        with:\n          go-version: 1.25.13\n      - name: 真正门禁\n        run: make check-dependencies\n`)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /1\.25\.12/)
})

test('YAML run 块标量只包含统一门禁命令时通过', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with:\n          go-version: 1.25.13\n      - uses: actions/setup-go@v5\n        with:\n          go-version: 1.25.13\n      - name: 统一门禁\n        run: |\n          make check-dependencies\n`)

  assert.equal(result.status, 0, result.stderr)
})

test('YAML run 中 echo 或多命令不能伪装统一门禁', (t) => {
  for (const runValue of [
    'echo make check-dependencies',
    'make check-dependencies\\necho done',
  ]) {
    const indentedRun = runValue.split('\\n').map((line) => `          ${line}`).join('\\n')
    const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with:\n          go-version: 1.25.13\n      - uses: actions/setup-go@v5\n        with:\n          go-version: 1.25.13\n      - name: 伪装门禁\n        run: |\n${indentedRun}\n`)

    assert.notEqual(result.status, 0)
  }
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

test('两端锁文件使用已修复无限循环漏洞的 nanoid 补丁版', () => {
  for (const project of ['admin-web', 'wxapp']) {
    const packageJson = JSON.parse(readRepositoryFile(`${project}/package.json`))
    const packageLock = JSON.parse(readRepositoryFile(`${project}/package-lock.json`))

    assert.equal(packageLock.packages?.['node_modules/nanoid']?.version, '3.3.18')
    assert.equal(packageJson.dependencies?.nanoid, undefined, `${project} must not add nanoid as a direct dependency`)
    assert.equal(packageJson.devDependencies?.nanoid, undefined, `${project} must not add nanoid as a direct dev dependency`)
  }
})

test('Go 模块与 CI 工具链满足已扫描到的可达漏洞修复下限', () => {
  const goModSource = readRepositoryFile('backend/go.mod')
  const goMod = parseGoMod(goModSource)

  assertGoModPolicy(goModSource)
  for (const [modulePath, expectedVersion] of Object.entries({
    'github.com/golang-jwt/jwt/v4': 'v4.5.2',
    'google.golang.org/grpc': 'v1.82.1',
    'go.opentelemetry.io/otel/sdk': 'v1.44.0',
    'go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp': 'v1.44.0',
    'gopkg.in/yaml.v3': 'v3.0.1',
  })) {
    assert.equal(goMod.requirements.get(modulePath), expectedVersion)
  }
})

test('真实 Go 工具链降级时不能由 1.25.13 注释伪装通过', () => {
  const downgradedGoMod = `module example.com/policy\n\ngo 1.25.0\ntoolchain go1.25.12 // toolchain go1.25.13\n`

  assert.throws(() => assertGoModPolicy(downgradedGoMod))
})

test('setup-go step 外的 go-version 标量不能伪装成 step 配置', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    env:\n      first-go-version: 1.25.13\n      second-go-version: 1.25.13\n    steps:\n      - name: Go 一\n        uses: actions/setup-go@v5\n      - name: Go 二\n        uses: actions/setup-go@v5\n      - name: 统一门禁\n        run: make check-dependencies\n`)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /缺少 with\.go-version/)
})

test('大小写不同的 Actions/setup-go 仍计入版本门禁', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: Actions/setup-go@v5\n        with: { go-version: 1.25.12 }\n      - run: make check-dependencies\n`)

  assert.notEqual(result.status, 0)
})

test('大小写不同的 actions/SETUP-GO 与包含 @ 的合法 ref 可通过', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/SETUP-GO@release@candidate\n        with: { go-version: 1.25.13 }\n      - run: make check-dependencies\n`)

  assert.equal(result.status, 0, result.stderr)
})

test('setup-go 相似 action 名不计入工具链 step', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go-fork@v5\n        with: { go-version: 1.25.12 }\n      - run: make check-dependencies\n`)

  assert.equal(result.status, 0, result.stderr)
})

test('缺少 ref suffix 的 setup-go uses 不得计为有效 action ref', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go@\n        with: { go-version: 1.25.13 }\n      - run: make check-dependencies\n`)

  assert.notEqual(result.status, 0)
})

test('setup-go 根目录等价引用仍计入版本门禁', (t) => {
  for (const rootEquivalentUses of [
    'actions/setup-go/.@v5',
    'actions/setup-go/./@v5',
    'actions/setup-go//@v5',
    'actions/setup-go//./@v5',
  ]) {
    const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: ${rootEquivalentUses}\n        with: { go-version: 1.25.12 }\n      - run: make check-dependencies\n`)

    assert.notEqual(result.status, 0, `${rootEquivalentUses} 必须按第三个 setup-go step 计入`)
  }
})

test('action 与 ref 分隔符两侧的内部空白使 uses 无效', (t) => {
  for (const invalidUses of ['actions/setup-go @v5', 'actions/setup-go@ v5']) {
    const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: '${invalidUses}'\n        with: { go-version: 1.25.13 }\n      - run: make check-dependencies\n`)

    assert.notEqual(result.status, 0, `${invalidUses} 不得计为有效 setup-go step`)
  }
})

test('uses YAML 标量的外层空白不影响 setup-go 识别', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: '  actions/setup-go@v5  '\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - run: make check-dependencies\n`)

  assert.equal(result.status, 0, result.stderr)
})

test('setup-go 非根 subpath、本地、Docker、绝对路径、空仓库与上级逃逸引用均不计数', (t) => {
  const result = runCIWorkflowFixture(t, `jobs:\n  dependency-security:\n    steps:\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go@v5\n        with: { go-version: 1.25.13 }\n      - uses: actions/setup-go/subaction@v5\n        with: { go-version: 1.25.12 }\n      - uses: actions/setup-go-fork@v5\n        with: { go-version: 1.25.12 }\n      - uses: ./actions/setup-go@v5\n        with: { go-version: 1.25.12 }\n      - uses: 'docker://actions/setup-go@v5'\n        with: { go-version: 1.25.12 }\n      - uses: /actions/setup-go@v5\n        with: { go-version: 1.25.12 }\n      - uses: actions/@v5\n        with: { go-version: 1.25.12 }\n      - uses: actions/setup-go/../other@v5\n        with: { go-version: 1.25.12 }\n      - run: make check-dependencies\n`)

  assert.equal(result.status, 0, result.stderr)
})
