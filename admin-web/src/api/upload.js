import http from './http'
import { buildUploadedFileUrl, inferContentType } from './uploadUtils'

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
    throw new Error('封面上传失败，请稍后重试')
  }
  return buildUploadedFileUrl(token)
}
