import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const root = path.resolve(import.meta.dirname, '..')
const viewSource = fs.readFileSync(path.join(root, 'src/views/ResourceReportView.vue'), 'utf8')
const apiSource = fs.readFileSync(path.join(root, 'src/api/resource.js'), 'utf8')

test('resource report review dialog previews resource detail before review', () => {
  assert.match(viewSource, /import \{ getResource, listResourceReports, reviewResourceReport \} from '\.\.\/api\/resource'/)
  assert.match(apiSource, /export function getResource\(resourceId\)/)
  assert.match(apiSource, /\/api\/v1\/resources\/\$\{resourceId\}/)
  assert.match(viewSource, /title="处理举报" width="760px"/)
  assert.match(viewSource, /供需详情预览/)
  assert.match(viewSource, /v-loading="previewLoading"/)
  assert.match(viewSource, /previewTitle/)
  assert.match(viewSource, /previewMetaItems/)
  assert.match(viewSource, /previewAttributeItems/)
  assert.match(viewSource, /previewContactItems/)
  assert.match(viewSource, /<el-image[\s\S]*:preview-src-list="previewImages"/)
  assert.match(viewSource, /loadResourcePreview\(row\)/)
  assert.match(viewSource, /const detail = await getResource\(resourceId\)/)
  assert.match(viewSource, /:disabled="previewLoading"/)
  assert.match(viewSource, /if \(!reviewTarget\.value \|\| submitting\.value \|\| previewLoading\.value\) return/)
})

test('resource report review dialog keeps report summary with review controls', () => {
  assert.match(viewSource, /举报摘要/)
  assert.match(viewSource, /reviewTarget\.reportCount/)
  assert.match(viewSource, /reportReasonText\[reviewTarget\.reasonCode\]/)
  assert.match(viewSource, /reviewTarget\.reasonText \|\| '无'/)
  assert.match(viewSource, /处理结论/)
  assert.match(viewSource, /<el-button-group class="review-action-group">/)
  assert.match(viewSource, /@click="setReviewAction\('valid'\)"[\s\S]*举报成立并下架/)
  assert.match(viewSource, /@click="setReviewAction\('invalid'\)"[\s\S]*举报不成立/)
  assert.match(viewSource, /function setReviewAction\(action\) \{[\s\S]*reviewForm\.action = action[\s\S]*\}/)
  assert.match(viewSource, /resourceAction: action === 'valid' \? 'take_down' : 'none'/)
  assert.doesNotMatch(viewSource, /<el-radio-group/)
  assert.doesNotMatch(viewSource, /<el-radio-button label="valid">/)
  assert.doesNotMatch(viewSource, /<el-radio-button label="invalid">/)
  assert.match(viewSource, /确认处理/)
})
