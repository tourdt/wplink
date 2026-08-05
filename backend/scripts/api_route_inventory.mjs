import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const HTTP_METHOD_PATTERN = '(?:get|post|put|delete|patch|head|options)'
const scriptDir = path.dirname(fileURLToPath(import.meta.url))

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
