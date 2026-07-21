import type {TFunction} from '@nextunnel/web-shared'
import type {IPFilterRule} from '../types'

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
                return t('accessControl.ruleField.allTraffic')
            case 'LOCAL':
                return t('accessControl.ruleField.localNetwork')
            case 'REMOTE':
                return t('accessControl.ruleField.remoteNetwork')
            default:
                return rule.value ?? '-'
        }
    }

    const fieldKey = `accessControl.ruleField.${rule.field}` as const
    const label = t(fieldKey)
    return `${label}: ${rule.value ?? '-'}`
}
