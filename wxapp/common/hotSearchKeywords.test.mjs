import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'common/hotSearchKeywords.js'), 'utf8')
const normalizeHotSearchKeywords = new Function(`
  ${source
    .replace(/import[^\n]+\n/g, '')
    .replace('export function normalizeHotSearchKeywords', 'function normalizeHotSearchKeywords')
    .replace(/export /g, '')}
  return normalizeHotSearchKeywords
`)()

test('normalizes hot search keyword API items', () => {
  assert.deepEqual(
    normalizeHotSearchKeywords([
      { keyword: ' 夏款现货 ' },
      { keyword: '夏款现货' },
      ' 小单快返 ',
      { keyword: '' },
      null,
    ]),
    ['夏款现货', '小单快返'],
  )
})

test('limits hot search keywords for compact search page display', () => {
  const items = Array.from({ length: 12 }, (_, index) => ({ keyword: `词${index}` }))

  assert.equal(normalizeHotSearchKeywords(items).length, 8)
})

test('app launch refreshes hot search keywords from server config', () => {
  const appSource = fs.readFileSync(path.join(root, 'App.vue'), 'utf8')

  assert.match(appSource, /import \{ loadHotSearchKeywords \} from '\.\/common\/hotSearchKeywords'/)
  assert.match(appSource, /onLaunch\(\) \{[\s\S]*loadHotSearchKeywords\(DEFAULT_CITY_CODE\)[\s\S]*\}/)
})
