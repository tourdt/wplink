import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(new URL('../..', import.meta.url).pathname)
const source = fs.readFileSync(path.join(root, 'pages/resource/report.vue'), 'utf8')
const pagesConfig = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8'))
const apiSource = fs.readFileSync(path.join(root, 'api/resource.js'), 'utf8')

test('resource report page is registered and submits existing report API', () => {
  const pagePaths = (pagesConfig.pages || []).map((item) => item.path)

  assert.equal(pagePaths.includes('pages/resource/report'), true)
  assert.match(source, /import \{ reportResource \} from '\.\.\/\.\.\/api\/resource'/)
  assert.match(source, /import \{ requireLogin \} from '\.\.\/\.\.\/common\/auth'/)
  assert.match(apiSource, /export function reportResource\(resourceId, data\)/)
  assert.match(apiSource, /\/api\/v1\/resources\/\$\{resourceId\}\/reports/)
})

test('resource report page validates reason and optional contact', () => {
  assert.match(source, /const maxReasonTextLength = 200/)
  assert.match(source, /const maxReportContactLength = 80/)
  assert.match(source, /const reasonOptions = \[/)
  for (const token of ['fake_info', 'unreachable', 'inaccurate_price_quantity', 'image_infringement', 'illegal_content', 'malicious_redirect', 'other']) {
    assert.match(source, new RegExp(token))
  }
  assert.match(source, /if \(!selectedReason\.value\) \{[\s\S]*请选择举报原因[\s\S]*return false[\s\S]*\}/)
  assert.match(source, /if \(isOtherReason\.value && !trimText\(form\.reasonText\)\) \{[\s\S]*请填写举报说明[\s\S]*return false[\s\S]*\}/)
  assert.match(source, /if \(reasonTextLength\.value > maxReasonTextLength\) \{[\s\S]*举报说明请控制在 200 字以内[\s\S]*\}/)
  assert.match(source, /if \(Array\.from\(trimText\(form\.reporterContact\)\)\.length > maxReportContactLength\) \{[\s\S]*联系方式请控制在 80 字以内[\s\S]*\}/)
})

test('resource report page builds reason text and contact evidence', () => {
  assert.match(source, /reasonCode: form\.reasonCode,[\s\S]*reasonText: buildReportReasonText\(\),[\s\S]*evidence: buildReportEvidence\(\)/)
  assert.match(source, /function buildReportReasonText\(\) \{[\s\S]*return trimText\(form\.reasonText\)[\s\S]*\}/)
  assert.match(source, /function buildReportEvidence\(\) \{[\s\S]*const evidence = \{ source: 'resource_report_page' \}[\s\S]*evidence\.reporterContact = reporterContact[\s\S]*evidence\.resourceTitle = resourceTitle\.value[\s\S]*return evidence[\s\S]*\}/)
  assert.match(source, /uni\.showToast\(\{ title: resp\?\.message \|\| '举报已提交'/)
  assert.match(source, /uni\.navigateBack\(\{ delta: 1 \}\)/)
})

test('resource report page keeps optional copy concise', () => {
  assert.doesNotMatch(source, /请选择最接近的问题/)
  assert.doesNotMatch(source, /处理说明/)
  assert.doesNotMatch(source, /不填写也可以提交举报/)
  assert.doesNotMatch(source, /notice-card/)
  assert.doesNotMatch(source, /summary-desc/)
  assert.doesNotMatch(source, /notice-desc/)
  assert.doesNotMatch(source, /notice-title/)
})
