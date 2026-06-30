export function formatDateToDay(value, placeholder = '-') {
  if (!value) return placeholder
  const text = String(value).trim()
  if (!text) return placeholder

  const dayMatch = text.match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (dayMatch) {
    return `${dayMatch[1]}-${dayMatch[2]}-${dayMatch[3]}`
  }

  const date = new Date(text)
  if (Number.isNaN(date.getTime())) return placeholder

  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function formatListFreshnessDate(value, now = new Date(), placeholder = '近期') {
  if (!value) return placeholder
  const text = String(value).trim()
  if (!text) return placeholder

  const dayMatch = text.match(/^(\d{4})-(\d{2})-(\d{2})/)
  const date = dayMatch ? new Date(Number(dayMatch[1]), Number(dayMatch[2]) - 1, Number(dayMatch[3])) : new Date(text)
  if (Number.isNaN(date.getTime())) return placeholder

  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const dateStart = new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
  const dayDiff = Math.round((todayStart - dateStart) / 86400000)

  if (dayDiff === 0) return '今天'
  if (dayDiff === 1) return '昨天'

  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${month}-${day}`
}
