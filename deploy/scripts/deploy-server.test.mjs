import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '../..')
const deployScriptPath = path.join(scriptDir, 'deploy-server.sh')
const migrationsDir = path.join(repoRoot, 'backend/migrations')
const gitignorePath = path.join(repoRoot, '.gitignore')

test('deploy script covers local build, remote install, migrations, restart, and health checks', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')

  assert.match(script, /WPLINK_DEPLOY_TARGET/)
  assert.match(script, /WPLINK_SSH_KEY/)
  assert.match(script, /ssh_cmd=\(ssh/)
  assert.match(script, /scp_cmd=\(scp/)
  assert.match(script, /-i "\$WPLINK_SSH_KEY"/)
  assert.match(script, /build-release\.sh/)
  assert.match(script, /scp\b/)
  assert.match(script, /ssh\b/)
  assert.match(script, /systemctl restart "\$REMOTE_SERVICE"/)
  assert.match(script, /\/healthz/)
  assert.match(script, /\/readyz/)
})

test('deploy script migrates every current up migration in sorted order', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')
  const migrationFiles = fs
    .readdirSync(migrationsDir)
    .filter((fileName) => fileName.endsWith('.up.sql'))
    .sort()

  assert(migrationFiles.length > 0)
  assert.match(script, /find "\$ROOT_DIR\/backend\/migrations"/)
  assert.match(script, /-name '\*\.up\.sql'/)
  assert.match(script, /\| sort/)
  assert.match(script, /migrations\.manifest/)
  assert.match(script, /while IFS= read -r migration_file/)
})

test('deploy script serializes and atomically records migrations', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')

  assert.match(script, /BEGIN;/)
  assert.match(
    script,
    /pg_advisory_xact_lock\(hashtext\('wplink_schema_migrations'\)\)/,
  )
  assert.match(script, /SELECT NOT EXISTS .*schema_migrations/)
  assert.match(script, /AS should_apply \\\\gset/)
  assert.match(script, /\\\\if :should_apply/)
  assert.match(script, /INSERT INTO schema_migrations/)
  assert.match(script, /COMMIT;/)
})

test('deploy script fails closed before the first scheduler protocol migration', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')

  assert.match(script, /--confirm-no-legacy-schedulers/)
  assert.match(script, /to_regclass\('public\.resources'\)/)
  assert.match(script, /information_schema\.columns/)
  assert.match(script, /audit_lease_until/)
  assert.match(script, /必须先停止所有旧 API\/Scheduler/)
  assert.match(script, /确认参数只表示运维已在所有主机消除旧 Scheduler/)

  const protocolDetectionIndex = script.indexOf("to_regclass('public.resources')")
  const migrationBatchIndex = script.indexOf('migration_batch="$REMOTE_TMP/run-migrations.sql"')
  const migrationExecutionIndex = script.indexOf('psql_run "$database_url" -q -f "$migration_batch"')
  assert(protocolDetectionIndex >= 0)
  assert(protocolDetectionIndex < migrationBatchIndex)
  assert(migrationBatchIndex < migrationExecutionIndex)
})

test('protocol detection is unconditional before the migration guard and binary install', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')
  const psqlFunctionIndex = script.indexOf('psql_run()')
  const databaseReadIndex = script.indexOf(
    'database_url="$(read_database_url)"',
    psqlFunctionIndex,
  )
  const protocolDetectionIndex = script.indexOf(
    "to_regclass('public.resources')",
    psqlFunctionIndex,
  )
  const migrationGuard = 'if [[ "$RUN_MIGRATIONS" == "1" || "$MARK_MIGRATIONS_APPLIED" == "1" ]]; then'
  const migrationGuardIndex = script.indexOf(migrationGuard, psqlFunctionIndex)
  const binaryInstallIndex = script.indexOf(
    'install -m 0755 "$extract_dir/dist/release/wplink-api"',
  )

  assert.equal(script.split(migrationGuard).length - 1, 1)
  assert(psqlFunctionIndex >= 0)
  assert(databaseReadIndex > psqlFunctionIndex)
  assert(databaseReadIndex < migrationGuardIndex)
  assert(protocolDetectionIndex < migrationGuardIndex)
  assert(protocolDetectionIndex < binaryInstallIndex)
})

test('confirmed first scheduler protocol upgrade stops the target service before migrations and binary switch', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')

  const legacyBranchIndex = script.indexOf('legacy)')
  const confirmationCheckIndex = script.indexOf('CONFIRM_NO_LEGACY_SCHEDULERS', legacyBranchIndex)
  const serviceStopIndex = script.indexOf('systemctl stop "$REMOTE_SERVICE"', confirmationCheckIndex)
  const migrationBatchIndex = script.indexOf('migration_batch="$REMOTE_TMP/run-migrations.sql"')
  const binaryInstallIndex = script.indexOf(
    'install -m 0755 "$extract_dir/dist/release/wplink-api"',
  )

  assert(legacyBranchIndex >= 0)
  assert(confirmationCheckIndex > legacyBranchIndex)
  assert(serviceStopIndex > confirmationCheckIndex)
  assert(serviceStopIndex < migrationBatchIndex)
  assert(migrationBatchIndex < binaryInstallIndex)
})

test('clean databases reject skip and mark-only while normal migrations remain allowed', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')
  const cleanBranchIndex = script.indexOf('clean)')
  const compatibleBranchIndex = script.indexOf('compatible)', cleanBranchIndex)
  const binaryInstallIndex = script.indexOf(
    'install -m 0755 "$extract_dir/dist/release/wplink-api"',
  )
  const cleanBranch = script.slice(cleanBranchIndex, compatibleBranchIndex)

  assert(cleanBranchIndex < binaryInstallIndex)
  assert.match(script, /THEN 'clean'/)
  assert.match(cleanBranch, /RUN_MIGRATIONS.*!= "1"/)
  assert.match(cleanBranch, /MARK_MIGRATIONS_APPLIED.*== "1"/)
  assert.match(cleanBranch, /干净新库不能跳过 migration/)
  assert.match(cleanBranch, /干净新库不能只标记 migration/)
  assert.match(cleanBranch, /干净新库将执行完整 migration/)
})

test('legacy databases reject skip and mark-only before confirmation', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')
  const legacyBranchIndex = script.indexOf('legacy)')
  const unknownBranchIndex = script.indexOf('*)', legacyBranchIndex)
  const binaryInstallIndex = script.indexOf(
    'install -m 0755 "$extract_dir/dist/release/wplink-api"',
  )
  const legacyBranch = script.slice(legacyBranchIndex, unknownBranchIndex)

  assert(legacyBranchIndex < binaryInstallIndex)
  assert.match(legacyBranch, /RUN_MIGRATIONS.*!= "1"/)
  assert.match(legacyBranch, /MARK_MIGRATIONS_APPLIED.*== "1"/)
  assert.match(legacyBranch, /旧协议数据库不能跳过 migration/)
  assert.match(legacyBranch, /旧协议数据库不能只标记 migration/)
  assert(
    legacyBranch.indexOf('RUN_MIGRATIONS') <
      legacyBranch.indexOf('CONFIRM_NO_LEGACY_SCHEDULERS'),
  )
})

test('compatible databases are the only state that permits skipping migrations', () => {
  const script = fs.readFileSync(deployScriptPath, 'utf8')
  const compatibleBranchIndex = script.indexOf('compatible)')
  const legacyBranchIndex = script.indexOf('legacy)', compatibleBranchIndex)
  const binaryInstallIndex = script.indexOf(
    'install -m 0755 "$extract_dir/dist/release/wplink-api"',
  )
  const compatibleBranch = script.slice(compatibleBranchIndex, legacyBranchIndex)

  assert(compatibleBranchIndex < binaryInstallIndex)
  assert.match(script, /THEN 'compatible'/)
  assert.match(compatibleBranch, /已具备审核租约字段/)
  assert.doesNotMatch(compatibleBranch, /exit 1/)
})

test('repository ignores private tls certificate and key material', () => {
  const gitignoreLines = fs
    .readFileSync(gitignorePath, 'utf8')
    .split(/\r?\n/)

  for (const pattern of ['*.crt', '*.key', '*.pem']) {
    assert(
      gitignoreLines.includes(pattern),
      `.gitignore should include ${pattern}`,
    )
  }
})
