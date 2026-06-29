export function inferContentType(file) {
  const type = String(file?.type || '').trim().toLowerCase()
  if (type.startsWith('image/')) return type

  const name = String(file?.name || '').toLowerCase()
  if (name.endsWith('.png')) return 'image/png'
  if (name.endsWith('.webp')) return 'image/webp'
  return 'image/jpeg'
}

export function buildUploadedFileUrl(token) {
  const baseUrl = String(token?.publicBaseUrl || '').replace(/\/+$/, '')
  const objectKey = String(token?.objectKey || '').replace(/^\/+/, '')
  return `${baseUrl}/${objectKey}`
}

export function buildQiniuUploadErrorMessage(status, bodyText = '') {
  const detail = parseQiniuErrorDetail(bodyText)
  if (!detail) return `封面上传失败（七牛 ${status}）`
  return `封面上传失败（七牛 ${status}：${detail}）`
}

function parseQiniuErrorDetail(bodyText) {
  const text = String(bodyText || '').trim()
  if (!text) return ''
  try {
    const data = JSON.parse(text)
    return String(data?.error || data?.message || '').trim()
  } catch {
    return text
  }
}
