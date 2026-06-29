import { createUploadToken } from '../api/upload'

export function chooseAndUploadImage(purpose) {
  return chooseImageFile().then((file) => uploadSelectedImage(file, purpose))
}

export function chooseAndCropSquareImageFile(options = {}) {
  return chooseImageFile().then((file) => cropImageToSquare(file, options))
}

export function chooseImageFile() {
  return new Promise((resolve, reject) => {
    uni.chooseImage({
      count: 1,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
      success: (chooseRes) => {
        const filePath = chooseRes.tempFilePaths[0]
        const file = chooseRes.tempFiles?.[0] || {}
        resolve(createImageFileFromPath(filePath, file))
      },
      fail: reject,
    })
  })
}

export function createImageFileFromPath(filePath, file = {}) {
  return {
    id: `${Date.now()}-${filePath}`,
    path: filePath,
    fileName: filePath.split('/').pop() || 'image.jpg',
    contentType: inferContentType(filePath),
    size: file.size || 1,
  }
}

export function cropImageToSquare(file, options = {}) {
  return new Promise((resolve, reject) => {
    const cropImage = getCropImageApi()
    if (!cropImage) {
      centerCropImageToSquare(file, options).then(resolve).catch(reject)
      return
    }
    cropImage({
      src: file.path,
      cropScale: '1:1',
      success: (res) => {
        const croppedPath = res.tempFilePath || file.path
        resolve({
          ...file,
          id: `${Date.now()}-${croppedPath}`,
          path: croppedPath,
          fileName: croppedPath.split('/').pop() || file.fileName || 'logo.jpg',
          contentType: inferContentType(croppedPath),
        })
      },
      fail: () => {
        centerCropImageToSquare(file, options).then(resolve).catch(reject)
      },
    })
  })
}

export function centerCropImageToSquare(file, options = {}) {
  return new Promise((resolve, reject) => {
    if (!canUseCanvasCrop()) {
      reject(new Error('当前环境不支持图片裁剪，请升级微信后重试'))
      return
    }
    const canvasId = options.canvasId || 'merchantLogoCropCanvas'
    const outputSize = options.outputSize || 512
    uni.getImageInfo({
      src: file.path,
      success: (info) => {
        const sourcePath = info.path || file.path
        const sourceSize = Math.min(info.width, info.height)
        const sourceX = Math.max(0, (info.width - sourceSize) / 2)
        const sourceY = Math.max(0, (info.height - sourceSize) / 2)
        const ctx = uni.createCanvasContext(canvasId)
        ctx.clearRect(0, 0, outputSize, outputSize)
        ctx.drawImage(sourcePath, sourceX, sourceY, sourceSize, sourceSize, 0, 0, outputSize, outputSize)
        let exported = false
        const finishCanvasCrop = () => {
          if (exported) return
          exported = true
          uni.canvasToTempFilePath({
            canvasId,
            width: outputSize,
            height: outputSize,
            destWidth: outputSize,
            destHeight: outputSize,
            fileType: 'jpg',
            quality: 0.92,
            success: (res) => {
              const croppedPath = res.tempFilePath || sourcePath
              resolve({
                ...file,
                id: `${Date.now()}-${croppedPath}`,
                path: croppedPath,
                fileName: croppedPath.split('/').pop() || file.fileName || 'logo.jpg',
                contentType: 'image/jpeg',
              })
            },
            fail: reject,
          })
        }
        ctx.draw(false, finishCanvasCrop)
        setTimeout(finishCanvasCrop, 300)
      },
      fail: reject,
    })
  })
}

export async function uploadSelectedImage(file, purpose) {
  const token = await createUploadToken({
    purpose,
    fileName: file.fileName || file.path.split('/').pop() || `${purpose}.jpg`,
    contentType: file.contentType || inferContentType(file.path),
    fileSize: file.size || 1,
  })
  await uploadToQiniu(file.path, token)
  return `${token.publicBaseUrl}/${token.objectKey}`
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

function getCropImageApi() {
  if (typeof uni !== 'undefined' && typeof uni.cropImage === 'function') return uni.cropImage.bind(uni)
  if (typeof wx !== 'undefined' && typeof wx.cropImage === 'function') return wx.cropImage.bind(wx)
  return null
}

function canUseCanvasCrop() {
  return typeof uni !== 'undefined'
    && typeof uni.getImageInfo === 'function'
    && typeof uni.createCanvasContext === 'function'
    && typeof uni.canvasToTempFilePath === 'function'
}

function inferContentType(filePath) {
  const lower = filePath.toLowerCase()
  if (lower.endsWith('.png')) return 'image/png'
  if (lower.endsWith('.webp')) return 'image/webp'
  return 'image/jpeg'
}
