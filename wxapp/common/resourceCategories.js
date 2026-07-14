import { resourceTypeText } from './enums.js'

const DEFAULT_GROUP_CODE = 'other'
const DEFAULT_GROUP_NAME = '其他类目'
const DEFAULT_GROUP_SORT = 999

export function normalizeResourceTypeOption(item = {}) {
  const group = normalizeResourceTypeGroup(item.displayTemplate?.group)
  const typeCode = String(item.typeCode || '').trim()
  const typeName = String(item.typeName || resourceTypeText[typeCode] || '').trim()
  return {
    ...item,
    groupCode: group.code,
    groupName: group.name,
    groupSort: group.sort,
    typeCode,
    typeName,
    label: typeName,
    value: typeCode,
    direction: normalizeResourceDirection(item.direction),
  }
}

export function groupResourceTypes(items = []) {
  const groupMap = new Map()
  items.map(normalizeResourceTypeOption).filter((item) => item.typeCode && item.typeName).forEach((item, index) => {
    if (!groupMap.has(item.groupCode)) {
      groupMap.set(item.groupCode, {
        code: item.groupCode,
        name: item.groupName,
        sort: item.groupSort,
        items: [],
      })
    }
    groupMap.get(item.groupCode).items.push({ ...item, sortIndex: index })
  })
  return Array.from(groupMap.values())
    .map((group) => ({
      ...group,
      items: group.items.sort((left, right) => left.sortIndex - right.sortIndex),
    }))
    .sort((left, right) => left.sort - right.sort)
}

export function flattenGroupedResourceTypes(groups = []) {
  return groups.flatMap((group) => (group.items || []).map((item) => ({
    ...item,
    label: `${group.name} / ${item.typeName}`,
    value: item.typeCode,
  })))
}

export function resourceTypeLabel(resource = {}) {
  const typeCode = String(resource.typeCode || '').trim()
  return String(resource.typeName || resourceTypeText[typeCode] || '').trim()
}

function normalizeResourceTypeGroup(group = {}) {
  const code = String(group?.code || '').trim() || DEFAULT_GROUP_CODE
  const name = String(group?.name || '').trim() || DEFAULT_GROUP_NAME
  const sort = Number(group?.sort)
  return {
    code,
    name,
    sort: Number.isFinite(sort) ? sort : DEFAULT_GROUP_SORT,
  }
}

function normalizeResourceDirection(direction) {
  const value = String(direction || '').trim()
  return value === 'demand' ? 'demand' : 'supply'
}
