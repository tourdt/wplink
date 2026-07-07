import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const buildScriptPath = path.join(scriptDir, 'build-release.sh')

test('build release defaults to a Linux server binary', () => {
  const script = fs.readFileSync(buildScriptPath, 'utf8')

  assert.match(script, /RELEASE_GOOS="\$\{WPLINK_RELEASE_GOOS:-linux\}"/)
  assert.match(script, /RELEASE_GOARCH="\$\{WPLINK_RELEASE_GOARCH:-amd64\}"/)
  assert.match(script, /GOOS="\$RELEASE_GOOS"/)
  assert.match(script, /GOARCH="\$RELEASE_GOARCH"/)
})
