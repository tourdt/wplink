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
