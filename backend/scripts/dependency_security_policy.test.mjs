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

function stripYamlComment(line) {
  let singleQuoted = false
  let doubleQuoted = false
  for (let index = 0; index < line.length; index += 1) {
    const character = line[index]
    if (character === "'" && !doubleQuoted) singleQuoted = !singleQuoted
    if (character === '"' && !singleQuoted && line[index - 1] !== '\\') doubleQuoted = !doubleQuoted
    if (character === '#' && !singleQuoted && !doubleQuoted) return line.slice(0, index)
  }
  return line
}

function unquoteYamlScalar(value) {
  const trimmed = value.trim()
  if ((trimmed.startsWith('"') && trimmed.endsWith('"')) || (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
    return trimmed.slice(1, -1)
  }
  return trimmed
}

function parseYamlScalarAssignments(source, key) {
  const assignments = []
  const assignmentPattern = new RegExp(`^\\s*${key}:\\s*(.*?)\\s*$`)
  for (const rawLine of source.split(/\r?\n/)) {
    const line = stripYamlComment(rawLine)
    const match = line.match(assignmentPattern)
    if (match && match[1] && match[1] !== '|' && match[1] !== '>') {
      assignments.push(unquoteYamlScalar(match[1]))
    }
  }
  return assignments
}

function parseSetupGoVersions(source) {
  const steps = []
  const allGoVersions = []
  let currentStep = null

  for (const rawLine of source.split(/\r?\n/)) {
    const line = stripYamlComment(rawLine)
    if (!line.trim()) continue

    const indentation = line.match(/^\s*/)[0].length
    const stepStart = line.match(/^(\s*)-\s+(.*)$/)
    if (stepStart && (!currentStep || indentation <= currentStep.indent)) {
      currentStep = { indent: stepStart[1].length, uses: '', goVersions: [] }
      steps.push(currentStep)
      const inlineUses = stepStart[2].match(/^uses:\s*(\S+)\s*$/)
      if (inlineUses) currentStep.uses = unquoteYamlScalar(inlineUses[1])
    } else if (currentStep && indentation <= currentStep.indent) {
      currentStep = null
    }

    const uses = line.match(/^\s*uses:\s*(\S+)\s*$/)
    if (uses && currentStep && indentation > currentStep.indent) currentStep.uses = unquoteYamlScalar(uses[1])

    const goVersion = line.match(/^\s*go-version:\s*(.*?)\s*$/)
    if (goVersion && goVersion[1]) {
      const value = unquoteYamlScalar(goVersion[1])
      allGoVersions.push(value)
      if (currentStep && indentation > currentStep.indent) currentStep.goVersions.push(value)
    }
  }

  return {
    allGoVersions,
    setupGoSteps: steps.filter((step) => /^actions\/setup-go@/.test(step.uses)),
  }
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

function assertGoToolchainPolicy(goModSource, workflowSource) {
  const goMod = parseGoMod(goModSource)
  assert.deepEqual(goMod.go, ['1.25.0'])
  assert.deepEqual(goMod.toolchain, ['go1.25.13'])

  const workflow = parseSetupGoVersions(workflowSource)
  assert.equal(workflow.setupGoSteps.length, 2)
  assert.deepEqual(workflow.allGoVersions, ['1.25.13', '1.25.13'])
  for (const setupGoStep of workflow.setupGoSteps) {
    assert.deepEqual(setupGoStep.goVersions, ['1.25.13'])
  }
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

test('CI 调用统一依赖安全门禁', () => {
  const workflow = readRepositoryFile('.github/workflows/ci.yml')

  assert.equal(parseYamlScalarAssignments(workflow, 'run').filter((run) => run === 'make check-dependencies').length, 1)
  assert.equal(parseYamlScalarAssignments(workflow, 'continue-on-error').includes('true'), false)
})

test('注释不能伪装依赖安全门禁接线', () => {
  const makefileFixture = `check-dependencies:\n\t# cd backend && node --test scripts/dependency_security_policy.test.mjs\n\tnpm audit --prefix admin-web --audit-level=high\n`
  const workflowFixture = `jobs:\n  dependency-security:\n    steps:\n      # run: make check-dependencies\n      - name: 伪装步骤\n        run: echo skipped\n`

  const recipes = parseMakeTargetRecipes(makefileFixture).get('check-dependencies') || []
  assert.equal(recipes.includes('cd backend && node --test scripts/dependency_security_policy.test.mjs'), false)
  assert.equal(parseYamlScalarAssignments(workflowFixture, 'run').includes('make check-dependencies'), false)
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
  const workflowSource = readRepositoryFile('.github/workflows/ci.yml')
  const goMod = parseGoMod(goModSource)

  assertGoToolchainPolicy(goModSource, workflowSource)
  for (const [modulePath, expectedVersion] of Object.entries({
    'github.com/golang-jwt/jwt/v4': 'v4.5.2',
    'google.golang.org/grpc': 'v1.82.1',
    'go.opentelemetry.io/otel/sdk': 'v1.44.0',
    'go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp': 'v1.44.0',
  })) {
    assert.equal(goMod.requirements.get(modulePath), expectedVersion)
  }
})

test('真实 Go 工具链与 CI 版本降级时不能由 1.25.13 注释伪装通过', () => {
  const downgradedGoMod = `module example.com/policy\n\ngo 1.25.0\ntoolchain go1.25.12 // toolchain go1.25.13\n`
  const downgradedWorkflow = `jobs:\n  first:\n    steps:\n      - name: Go 一\n        uses: actions/setup-go@v5\n        with:\n          go-version: 1.25.12 # go-version: 1.25.13\n  second:\n    steps:\n      - name: Go 二\n        uses: actions/setup-go@v5\n        with:\n          go-version: 1.25.12 # go-version: 1.25.13\n# go-version: 1.25.13\n`

  assert.throws(() => assertGoToolchainPolicy(downgradedGoMod, downgradedWorkflow))
})

test('setup-go step 外的 go-version 标量不能伪装成 step 配置', () => {
  const validGoMod = `module example.com/policy\n\ngo 1.25.0\ntoolchain go1.25.13\n`
  const misplacedWorkflow = `jobs:\n  first:\n    steps:\n      - name: Go 一\n        uses: actions/setup-go@v5\n    env:\n      go-version: 1.25.13\n  second:\n    steps:\n      - name: Go 二\n        uses: actions/setup-go@v5\n    env:\n      go-version: 1.25.13\n`

  assert.throws(() => assertGoToolchainPolicy(validGoMod, misplacedWorkflow))
})
