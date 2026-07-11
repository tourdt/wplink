export const misleadingMerchantNameMessage = '展示名称不能包含认证、官方等容易误导的字样'

const misleadingMerchantNameKeywords = [
  '认证',
  '官方',
  '平台推荐',
  '旗舰',
  '直营',
  '授权',
  '指定',
]

export function validateMerchantName(name) {
  const normalizedName = String(name || '').trim()
  if (!normalizedName) return '请填写展示名称'

  const hasMisleadingKeyword = misleadingMerchantNameKeywords.some((keyword) => normalizedName.includes(keyword))
  if (hasMisleadingKeyword) return misleadingMerchantNameMessage

  return ''
}
