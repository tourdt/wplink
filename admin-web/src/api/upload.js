import http from './http'
import { buildQiniuUploadErrorMessage, buildUploadedFileUrl, inferContentType } from './uploadUtils'

export function createUploadToken(data) {
  return http.post('/api/v1/uploads/token', data)
}

export async function uploadBannerImage(file) {
  const token = await createUploadToken({
    purpose: 'banner',
    fileName: file.name || 'banner.jpg',
    contentType: inferContentType(file),
    fileSize: file.size || 1,
  })
  const formData = new FormData()
  formData.append('token', token.uploadToken)
  formData.append('key', token.objectKey)
  formData.append('file', file)

  const resp = await fetch(token.uploadUrl, {
    method: 'POST',
    body: formData,
  })
  if (!resp.ok) {
    const bodyText = await resp.text().catch(() => '')
    throw new Error(buildQiniuUploadErrorMessage(resp.status, bodyText))
  }
  return buildUploadedFileUrl(token)
}

export async function uploadMapBackgroundImage(file) {
  const token = await createUploadToken({
    purpose: 'map_background',
    fileName: file.name || 'map-background.png',
    contentType: inferContentType(file),
    fileSize: file.size || 1,
  })
  const formData = new FormData()
  formData.append('token', token.uploadToken)
  formData.append('key', token.objectKey)
  formData.append('file', file)

  const resp = await fetch(token.uploadUrl, {
    method: 'POST',
    body: formData,
  })
  if (!resp.ok) {
    const bodyText = await resp.text().catch(() => '')
    throw new Error(buildQiniuUploadErrorMessage(resp.status, bodyText, '底图'))
  }
  return buildUploadedFileUrl(token)
}
