import type { IPFilterRule } from '../types'
import type { TFunction } from './context'

export function formatPortRange(t: TFunction, portStart: number, portEnd: number): string {
  if (portStart > 0 && portEnd > 0) {
    return `${portStart}-${portEnd}`
  }
  return t('clients.allPorts')
}

export function ruleDisplayText(t: TFunction, rule: IPFilterRule): string {
  if (rule.field === 'category') {
    switch (rule.value) {
      case 'ALL':
        return t('ipFilters.ruleField.allTraffic')
      case 'LOCAL':
        return t('ipFilters.ruleField.localNetwork')
      case 'REMOTE':
        return t('ipFilters.ruleField.remoteNetwork')
      default:
        return rule.value ?? '-'
    }
  }

  const fieldKey = `ipFilters.ruleField.${rule.field}` as const
  const label = t(fieldKey)
  return `${label}: ${rule.value ?? '-'}`
}
