export const MERCHANT_PROFILE_IMAGE_MAX_COUNT = 9

export function createStoredMerchantImageEntry(url) {
  return {
    id: `stored:${url}`,
    kind: 'stored',
    url,
  }
}

export function createPendingMerchantImageEntry(file) {
  return {
    id: file.id || `pending:${Date.now()}:${file.path}`,
    kind: 'pending',
    url: file.path,
    file,
  }
}

export function appendMerchantImageFiles(entries, files, maxCount = MERCHANT_PROFILE_IMAGE_MAX_COUNT) {
  const nextEntries = [...entries]
  for (const file of files) {
    if (nextEntries.length >= maxCount) break
    nextEntries.push(createPendingMerchantImageEntry(file))
  }
  return nextEntries
}

export function removeMerchantImageEntry(entries, entryId) {
  return entries.filter((entry) => entry.id !== entryId)
}

export function getMerchantImagePreviewUrl(entry) {
  return entry?.url || ''
}

export function getMerchantImageUrlsForPreview(entries) {
  return entries.map(getMerchantImagePreviewUrl).filter(Boolean)
}

export function getStoredMerchantImageUrls(entries) {
  return entries
    .filter((entry) => entry.kind === 'stored')
    .map((entry) => entry.url)
    .filter(Boolean)
}

export function resolveImageCompressionOptions(image) {
  const width = Number(image?.width || 0)
  const size = Number(image?.size || 0)
  const compressedWidth = width > 1280 ? 1280 : undefined
  let quality = 90
  if (size > 3_000_000) {
    quality = 75
  } else if (size > 2_000_000) {
    quality = 80
  } else if (size > 500_000) {
    quality = 85
  }
  return {
    shouldCompress: Boolean(compressedWidth || quality < 90),
    compressedWidth,
    quality,
  }
}
