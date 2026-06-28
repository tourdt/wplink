import { createUploadToken } from '../api/upload'

export function chooseAndUploadImage(purpose) {
  return new Promise((resolve, reject) => {
    uni.chooseImage({
      count: 1,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
      success: async (chooseRes) => {
        try {
          const filePath = chooseRes.tempFilePaths[0]
          const file = chooseRes.tempFiles?.[0] || {}
          const token = await createUploadToken({
            purpose,
            fileName: filePath.split('/').pop() || `${purpose}.jpg`,
            contentType: inferContentType(filePath),
            fileSize: file.size || 1,
          })
          await uploadToQiniu(filePath, token)
          resolve(`${token.publicBaseUrl}/${token.objectKey}`)
        } catch (err) {
          reject(err)
        }
      },
      fail: reject,
    })
  })
}

function uploadToQiniu(filePath, token) {
  return new Promise((resolve, reject) => {
    uni.uploadFile({
      url: token.uploadUrl,
      filePath,
      name: 'file',
      formData: {
        token: token.uploadToken,
        key: token.objectKey,
      },
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res)
          return
        }
        reject(new Error('图片上传失败，请稍后重试'))
      },
      fail: reject,
    })
  })
}

function inferContentType(filePath) {
  const lower = filePath.toLowerCase()
  if (lower.endsWith('.png')) return 'image/png'
  if (lower.endsWith('.webp')) return 'image/webp'
  return 'image/jpeg'
}
