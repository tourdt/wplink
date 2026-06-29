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
