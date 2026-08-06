import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const HTTP_METHOD_PATTERN = '(?:get|post|put|delete|patch|head|options)'
const HANDLER_STUB_MARKER = 'WPLINK_API_HANDLER_STUB'
const scriptDir = path.dirname(fileURLToPath(import.meta.url))

function listGoFiles(directory) {
  if (!fs.existsSync(directory)) {
    return []
  }
  const files = []
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      files.push(...listGoFiles(entryPath))
    } else if (entry.name.endsWith('.go')) {
      files.push(entryPath)
    }
  }
  return files.sort()
}

function normalizeRoutePath(routePath) {
  let normalized = String(routePath)
    .trim()
    .replace(/\{([^{}\/]+)\}/g, ':$1')
    .replace(/\/{2,}/g, '/')

  if (!normalized.startsWith('/')) {
    normalized = `/${normalized}`
  }

  return normalized.length > 1 ? normalized.replace(/\/+$/, '') : normalized
}

function normalizeRoute(method, routePath) {
  return {
    method: String(method).toUpperCase(),
    path: normalizeRoutePath(routePath),
  }
}

function joinRoutePath(prefix, routePath) {
  return normalizeRoutePath(`${prefix || '/'}/${routePath || '/'}`)
}

function stripComments(source) {
  let result = ''
  let quote = ''
  let escaped = false

  for (let index = 0; index < source.length; index += 1) {
    const current = source[index]
    const next = source[index + 1]

    if (quote) {
      result += current
      if (quote !== '`' && current === '\\' && !escaped) {
        escaped = true
      } else if (current === quote && !escaped) {
        quote = ''
      } else {
        escaped = false
      }
      continue
    }

    if (current === '"' || current === '\'' || current === '`') {
      quote = current
      result += current
      continue
    }

    if (current === '/' && next === '/') {
      result += '  '
      index += 2
      while (index < source.length && source[index] !== '\n') {
        result += '\r' === source[index] ? '\r' : ' '
        index += 1
      }
      if (index < source.length) {
        result += source[index]
      }
      continue
    }

    if (current === '/' && next === '*') {
      result += '  '
      index += 2
      while (index < source.length && !(source[index] === '*' && source[index + 1] === '/')) {
        // 保留换行，确保盘点结果的行号仍能对应原始 Go 文件。
        result += source[index] === '\n' || source[index] === '\r' ? source[index] : ' '
        index += 1
      }
      if (index < source.length) {
        result += '  '
        index += 1
      }
      continue
    }

    result += current
  }

  return result
}

function findClosingBrace(source, openingBraceIndex) {
  let depth = 0
  for (let index = openingBraceIndex; index < source.length; index += 1) {
    if (source[index] === '{') {
      depth += 1
    } else if (source[index] === '}') {
      depth -= 1
      if (depth === 0) {
        return index
      }
    }
  }
  return source.length
}

function parseAPIFile(filePath, visitedFiles) {
  const resolvedPath = path.resolve(filePath)
  if (visitedFiles.has(resolvedPath)) {
    return []
  }
  visitedFiles.add(resolvedPath)

  const source = stripComments(fs.readFileSync(resolvedPath, 'utf8'))
  const routes = []
  const importPattern = /^\s*import\s+"([^"]+)"/gm
  for (const match of source.matchAll(importPattern)) {
    routes.push(...parseAPIFile(path.resolve(path.dirname(resolvedPath), match[1]), visitedFiles))
  }

  const serverPattern = /@server\s*\(([\s\S]*?)\)\s*service\s+[^\s{]+\s*\{/g
  for (const serverMatch of source.matchAll(serverPattern)) {
    const options = serverMatch[1]
    const prefix = options.match(/\bprefix\s*:\s*([^\s)]+)/)?.[1] || '/'
    const group = options.match(/\bgroup\s*:\s*([^\s)]+)/)?.[1] || ''
    const openingBraceIndex = serverMatch.index + serverMatch[0].length - 1
    const body = source.slice(openingBraceIndex + 1, findClosingBrace(source, openingBraceIndex))
    const routePattern = new RegExp(`@handler\\s+([^\\s]+)\\s*\\r?\\n\\s*(${HTTP_METHOD_PATTERN})\\s+([^\\s(]+)`, 'gi')

    for (const routeMatch of body.matchAll(routePattern)) {
      const route = normalizeRoute(routeMatch[2], joinRoutePath(prefix, routeMatch[3]))
      routes.push({
        ...route,
        handler: routeMatch[1],
        group,
        source: resolvedPath,
      })
    }
  }

  return routes
}

export function parseAPIContracts(apiDir) {
  return parseAPIFile(path.join(apiDir, 'app.api'), new Set())
}

export function parseLegacyRoutes(files) {
  const routes = []
  const routePattern = new RegExp(`\\b[A-Za-z0-9_]+\\.HandleFunc\\s*\\(\\s*"(${HTTP_METHOD_PATTERN})\\s+([^"]+)"`, 'gi')

  for (const filePath of files) {
    const resolvedPath = path.resolve(filePath)
    const source = stripComments(fs.readFileSync(resolvedPath, 'utf8'))
    for (const match of source.matchAll(routePattern)) {
      const route = normalizeRoute(match[1], match[2])
      routes.push({
        ...route,
        source: resolvedPath,
        line: source.slice(0, match.index).split('\n').length,
      })
    }
  }

  return routes
}

export function routeFingerprint(route) {
  const normalized = normalizeRoute(route.method, route.path)
  return `${normalized.method} ${normalized.path}`
}

function tokenizeGo(source) {
  const tokens = []
  let line = 1

  for (let index = 0; index < source.length;) {
    const current = source[index]
    const next = source[index + 1]
    if (/\s/.test(current)) {
      if (current === '\n') {
        line += 1
      }
      index += 1
      continue
    }
    if (current === '/' && next === '/') {
      index += 2
      while (index < source.length && source[index] !== '\n') {
        index += 1
      }
      continue
    }
    if (current === '/' && next === '*') {
      index += 2
      while (index < source.length && !(source[index] === '*' && source[index + 1] === '/')) {
        if (source[index] === '\n') {
          line += 1
        }
        index += 1
      }
      index = Math.min(source.length, index + 2)
      continue
    }
    if (current === '"' || current === '`' || current === '\'') {
      const quote = current
      const tokenLine = line
      let raw = current
      index += 1
      let escaped = false
      while (index < source.length) {
        const char = source[index]
        raw += char
        index += 1
        if (char === '\n') {
          line += 1
        }
        if (quote !== '`' && char === '\\' && !escaped) {
          escaped = true
          continue
        }
        if (char === quote && !escaped) {
          break
        }
        escaped = false
      }
      let value = raw.slice(1, -1)
      if (quote === '"') {
        try {
          value = JSON.parse(raw)
        } catch {
          // 生成路由只应出现 goctl 输出的普通双引号字符串；保留原值以便错误包含具体指纹。
        }
      }
      tokens.push({ type: quote === '\'' ? 'rune' : 'string', value, line: tokenLine })
      continue
    }
    if (/[A-Za-z_]/.test(current)) {
      const tokenLine = line
      let value = current
      index += 1
      while (index < source.length && /[A-Za-z0-9_]/.test(source[index])) {
        value += source[index]
        index += 1
      }
      tokens.push({ type: 'identifier', value, line: tokenLine })
      continue
    }
    tokens.push({ type: 'symbol', value: current, line })
    index += 1
  }
  return tokens
}

function matchingTokenIndex(tokens, openingIndex, opening, closing) {
  let depth = 0
  for (let index = openingIndex; index < tokens.length; index += 1) {
    if (tokens[index].value === opening) {
      depth += 1
    } else if (tokens[index].value === closing) {
      depth -= 1
      if (depth === 0) {
        return index
      }
    }
  }
  return tokens.length - 1
}

function parseGoImports(tokens) {
  const imports = new Map()
  for (let index = 0; index < tokens.length; index += 1) {
    if (tokens[index].value !== 'import') {
      continue
    }
    const importTokens = []
    if (tokens[index + 1]?.value === '(') {
      const closingIndex = matchingTokenIndex(tokens, index + 1, '(', ')')
      importTokens.push(...tokens.slice(index + 2, closingIndex))
      index = closingIndex
    } else {
      importTokens.push(...tokens.slice(index + 1, index + 3))
    }
    for (let cursor = 0; cursor < importTokens.length; cursor += 1) {
      let alias = ''
      let pathToken = importTokens[cursor]
      if (pathToken?.type !== 'string' && importTokens[cursor + 1]?.type === 'string') {
        alias = pathToken?.value || ''
        pathToken = importTokens[cursor + 1]
        cursor += 1
      }
      if (pathToken?.type !== 'string') {
        continue
      }
      const packageName = alias || pathToken.value.split('/').at(-1)
      imports.set(packageName, pathToken.value)
    }
  }
  return imports
}

const generatedMethodNames = new Map([
  ['MethodGet', 'GET'],
  ['MethodPost', 'POST'],
  ['MethodPut', 'PUT'],
  ['MethodDelete', 'DELETE'],
  ['MethodPatch', 'PATCH'],
  ['MethodHead', 'HEAD'],
  ['MethodOptions', 'OPTIONS'],
])

function routeFieldValue(tokens, startIndex, endIndex, fieldName) {
  let depth = 0
  for (let index = startIndex + 1; index < endIndex; index += 1) {
    if (tokens[index].value === '{' || tokens[index].value === '(' || tokens[index].value === '[') {
      depth += 1
      continue
    }
    if (tokens[index].value === '}' || tokens[index].value === ')' || tokens[index].value === ']') {
      depth = Math.max(0, depth - 1)
      continue
    }
    if (depth === 0 && tokens[index].value === fieldName && tokens[index + 1]?.value === ':') {
      return { index: index + 2, token: tokens[index + 2] }
    }
  }
  return null
}

function parseGeneratedRouteBlock(tokens, openingIndex, closingIndex, prefix, imports, sourcePath) {
  const methodField = routeFieldValue(tokens, openingIndex, closingIndex, 'Method')
  const pathField = routeFieldValue(tokens, openingIndex, closingIndex, 'Path')
  const handlerField = routeFieldValue(tokens, openingIndex, closingIndex, 'Handler')
  if (!methodField || !pathField || !handlerField || pathField.token?.type !== 'string') {
    return null
  }

  let method = ''
  if (methodField.token?.type === 'string') {
    method = methodField.token.value
  } else if (tokens[methodField.index + 1]?.value === '.' && imports.get(methodField.token?.value) === 'net/http') {
    method = generatedMethodNames.get(tokens[methodField.index + 2]?.value) || ''
  }
  const handlerAlias = handlerField.token?.value || ''
  const handlerFunction = tokens[handlerField.index + 1]?.value === '.'
    ? tokens[handlerField.index + 2]?.value || ''
    : handlerAlias
  if (!method || !handlerFunction) {
    return null
  }

  const importPath = imports.get(handlerAlias) || ''
  return {
    ...normalizeRoute(method, joinRoutePath(prefix, pathField.token.value)),
    // goctl 将 @handler Foo 生成为 FooHandler；门禁比较的是契约中的原始 Handler 名称。
    handler: handlerFunction.endsWith('Handler') ? handlerFunction.slice(0, -'Handler'.length) : handlerFunction,
    group: importPath.split('/').at(-1) || handlerAlias,
    handlerPackage: importPath,
    source: sourcePath,
    line: methodField.token.line,
  }
}

export function parseGeneratedRoutes(routesFile) {
  const resolvedPath = path.resolve(routesFile)
  const tokens = tokenizeGo(fs.readFileSync(resolvedPath, 'utf8'))
  const imports = parseGoImports(tokens)
  const routes = []

  for (let index = 0; index < tokens.length - 3; index += 1) {
    if (tokens[index + 1]?.value !== '.' || tokens[index + 2]?.value !== 'AddRoutes' || tokens[index + 3]?.value !== '(') {
      continue
    }
    const callEnd = matchingTokenIndex(tokens, index + 3, '(', ')')
    let prefix = '/'
    for (let cursor = index + 4; cursor < callEnd - 3; cursor += 1) {
      if (tokens[cursor].value === 'WithPrefix' && tokens[cursor + 1]?.value === '(' && tokens[cursor + 2]?.type === 'string') {
        prefix = tokens[cursor + 2].value
      }
    }
    for (let cursor = index + 4; cursor < callEnd; cursor += 1) {
      if (tokens[cursor].value !== '{') {
        continue
      }
      const blockEnd = matchingTokenIndex(tokens, cursor, '{', '}')
      const route = parseGeneratedRouteBlock(tokens, cursor, blockEnd, prefix, imports, resolvedPath)
      if (route) {
        routes.push(route)
      }
    }
    index = callEnd
  }
  return routes
}

function routeDetail(route) {
  const expectedPackage = route.group ? `wplink/backend/app/internal/handler/${route.group}` : ''
  const handlerPackage = route.handlerPackage ?? expectedPackage
  const handlerLabel = handlerPackage && handlerPackage !== expectedPackage
    ? `${handlerPackage}.${route.handler || '未知 Handler'}`
    : route.handler || '未知 Handler'
  return `${routeFingerprint(route)} -> ${handlerLabel}`
}

function exactRouteKey(route) {
  const expectedPackage = route.group ? `wplink/backend/app/internal/handler/${route.group}` : ''
  const handlerPackage = route.handlerPackage ?? expectedPackage
  return `${routeFingerprint(route)}\u0000${handlerPackage}\u0000${route.handler}`
}

function duplicateRouteDetails(routes) {
  const byFingerprint = new Map()
  for (const route of routes) {
    const fingerprint = routeFingerprint(route)
    const matches = byFingerprint.get(fingerprint) || []
    matches.push(route)
    byFingerprint.set(fingerprint, matches)
  }
  return [...byFingerprint.values()]
    .filter((matches) => matches.length > 1)
    .flatMap((matches) => matches.map(routeDetail))
    .sort()
}

export function checkNoDuplicateRoutes(routes, label) {
  const duplicates = duplicateRouteDetails(routes)
  if (duplicates.length > 0) {
    throw new Error(`API 路由存在重复:\n${label}重复:\n- ${duplicates.join('\n- ')}`)
  }
}

export function checkContractGeneratedParity(contractRoutes, generatedRoutes) {
  const contractDuplicates = duplicateRouteDetails(contractRoutes)
  const generatedDuplicates = duplicateRouteDetails(generatedRoutes)
  if (contractDuplicates.length > 0 || generatedDuplicates.length > 0) {
    const sections = ['API 路由存在重复:']
    if (contractDuplicates.length > 0) {
      sections.push(`契约重复:\n- ${contractDuplicates.join('\n- ')}`)
    }
    if (generatedDuplicates.length > 0) {
      sections.push(`生成路由重复:\n- ${generatedDuplicates.join('\n- ')}`)
    }
    throw new Error(sections.join('\n'))
  }

  const generatedKeys = new Set(generatedRoutes.map(exactRouteKey))
  const contractKeys = new Set(contractRoutes.map(exactRouteKey))
  const missing = contractRoutes.filter((route) => !generatedKeys.has(exactRouteKey(route))).map(routeDetail).sort()
  const extra = generatedRoutes.filter((route) => !contractKeys.has(exactRouteKey(route))).map(routeDetail).sort()
  if (missing.length === 0 && extra.length === 0) {
    return
  }
  const sections = ['API 契约与生成路由不一致:']
  if (missing.length > 0) {
    sections.push(`缺失:\n- ${missing.join('\n- ')}`)
  }
  if (extra.length > 0) {
    sections.push(`多余:\n- ${extra.join('\n- ')}`)
  }
  throw new Error(sections.join('\n'))
}

export function checkNoMigrationStubs(rootHandlerDir) {
  const scannerPath = path.join(scriptDir, 'api_not_migrated_scanner.go')
  const result = spawnSync('go', ['run', scannerPath, path.resolve(rootHandlerDir)], {
    encoding: 'utf8',
    env: {
      ...process.env,
      GOCACHE: path.join(os.tmpdir(), 'wplink-api-route-inventory-go-cache'),
      GO111MODULE: 'off',
      GOTOOLCHAIN: 'local',
    },
  })
  if (result.error) {
    throw new Error(`无法执行 NotMigrated AST 扫描: ${result.error.message}`)
  }
  if (result.status !== 0) {
    const detail = [result.stdout, result.stderr].filter(Boolean).join('\n').trim()
    throw new Error(`NotMigrated AST 扫描失败${detail ? `:\n${detail}` : ''}`)
  }
  const findings = JSON.parse(result.stdout)
    .map((finding) => `${finding.file}:${finding.line}:${finding.column} -> ${finding.label}`)
    .sort()
  if (findings.length > 0) {
    throw new Error(`Handler 树仍存在 NotMigrated 引用:\n- ${findings.join('\n- ')}`)
  }

  const stubFindings = []
  for (const filePath of listGoFiles(rootHandlerDir)) {
    if (filePath.endsWith('_test.go')) {
      continue
    }
    const lines = fs.readFileSync(filePath, 'utf8').split(/\r?\n/)
    for (let index = 0; index < lines.length; index += 1) {
      if (lines[index].includes(HANDLER_STUB_MARKER)) {
        stubFindings.push(`${path.relative(rootHandlerDir, filePath).split(path.sep).join('/')}:${index + 1} -> ${HANDLER_STUB_MARKER}`)
      }
    }
  }
  if (stubFindings.length > 0) {
    throw new Error(`Handler 树仍存在代码生成占位骨架:\n- ${stubFindings.sort().join('\n- ')}`)
  }
}

function compareRoutes(contractRoutes, legacyRoutes) {
  const legacyFingerprints = new Set(legacyRoutes.map(routeFingerprint))
  const contractFingerprints = new Set(contractRoutes.map(routeFingerprint))

  return {
    contractOnly: contractRoutes.filter((route) => !legacyFingerprints.has(routeFingerprint(route))),
    legacyOnly: legacyRoutes.filter((route) => !contractFingerprints.has(routeFingerprint(route))),
    shared: contractRoutes.filter((route) => legacyFingerprints.has(routeFingerprint(route))),
  }
}

function printTable(title, routes) {
  console.log(`\n${title}（${routes.length}）`)
  if (routes.length === 0) {
    console.log('无')
    return
  }
  console.table(routes.map((route) => ({
    route: routeFingerprint(route),
    handler: route.handler || '',
    group: route.group || '',
    source: route.source,
    line: route.line || '',
  })))
}

function runCLI() {
  const backendDir = path.resolve(scriptDir, '..')
  const contractRoutes = parseAPIContracts(path.join(backendDir, 'app/api'))
  const legacyRoutes = parseLegacyRoutes([
    path.join(backendDir, 'app/internal/server/api.go'),
    path.join(backendDir, 'app/internal/server/auth_routes.go'),
    path.join(backendDir, 'app/internal/server/domain_routes.go'),
    path.join(backendDir, 'app/internal/server/map_routes.go'),
  ])
  const inventory = compareRoutes(contractRoutes, legacyRoutes)

  if (process.argv.includes('--json')) {
    console.log(JSON.stringify(inventory, null, 2))
    return
  }

  printTable('契约独有', inventory.contractOnly)
  printTable('旧 Router 独有', inventory.legacyOnly)
  printTable('双方共有', inventory.shared)
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    runCLI()
  } catch (error) {
    console.error(`API 路由盘点失败: ${error.message}`)
    process.exitCode = 1
  }
}
