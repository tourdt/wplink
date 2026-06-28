import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const migrationPattern = /^(\d{6})_(.+)\.(up|down)\.sql$/

export function loadMigrationFiles(migrationsDir) {
  return fs
    .readdirSync(migrationsDir)
    .filter((fileName) => migrationPattern.test(fileName))
    .sort()
    .map((fileName) => {
      const [, version, name, direction] = fileName.match(migrationPattern)
      return {
        version,
        name,
        direction,
        fileName,
        sql: fs.readFileSync(path.join(migrationsDir, fileName), 'utf8'),
      }
    })
}

export function validateMigrationFiles(files) {
  const issues = []
  const groups = groupByMigration(files)
  const knownTables = new Set()

  for (const key of [...groups.keys()].sort()) {
    const group = groups.get(key)
    const up = group.up
    const down = group.down

    if (!up) {
      issues.push(`${key} 缺少 up migration`)
      continue
    }
    if (!down) {
      issues.push(`${key} 缺少 down migration`)
      continue
    }

    for (const table of extractInsertedTables(up.sql)) {
      if (!knownTables.has(table) && !extractCreatedTables(up.sql).has(table)) {
        issues.push(`${up.fileName} 插入表 ${table}，但此前 migration 未创建该表`)
      }
    }

    const createdTables = extractCreatedTables(up.sql)
    const droppedTables = extractDroppedTables(down.sql)
    for (const table of createdTables) {
      if (!droppedTables.has(table)) {
        issues.push(`${down.fileName} down 未删除 up 创建的表 ${table}`)
      }
    }

    for (const table of createdTables) {
      knownTables.add(table)
    }
  }

  return issues
}

function groupByMigration(files) {
  const groups = new Map()
  for (const file of files) {
    const key = `${file.version}_${file.name}`
    if (!groups.has(key)) {
      groups.set(key, {})
    }
    const group = groups.get(key)
    if (group[file.direction]) {
      group[file.direction].duplicate = true
    }
    group[file.direction] = file
  }
  return groups
}

function extractCreatedTables(sql) {
  return extractMatches(sql, /\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z][a-z0-9_]*)\b/gi)
}

function extractDroppedTables(sql) {
  return extractMatches(sql, /\bDROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([a-z][a-z0-9_]*)\b/gi)
}

function extractInsertedTables(sql) {
  return extractMatches(sql, /\bINSERT\s+INTO\s+([a-z][a-z0-9_]*)\b/gi)
}

function extractMatches(sql, regex) {
  const matches = new Set()
  for (const match of sql.matchAll(regex)) {
    matches.add(match[1].toLowerCase())
  }
  return matches
}

function main() {
  const scriptDir = path.dirname(fileURLToPath(import.meta.url))
  const migrationsDir = path.resolve(scriptDir, '../migrations')
  const issues = validateMigrationFiles(loadMigrationFiles(migrationsDir))

  if (issues.length > 0) {
    console.error('migration static check failed:')
    for (const issue of issues) {
      console.error(`- ${issue}`)
    }
    process.exitCode = 1
    return
  }

  console.log('migration static check ok')
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main()
}
