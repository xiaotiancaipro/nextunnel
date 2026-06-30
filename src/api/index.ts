import type {ApiError, Client, CreateClientRequest, IPFilterMutateRequest, IPFilterRule,} from '../types'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
        headers: {
            Accept: 'application/json',
            ...(init?.body ? {'Content-Type': 'application/json'} : {}),
            ...init?.headers,
        },
        ...init,
    })

    if (!response.ok) {
        let message = response.statusText
        try {
            const payload = (await response.json()) as ApiError
            if (payload.error) {
                message = payload.error
            }
        } catch {
            // ignore non-json error body
        }
        throw new Error(message)
    }

    if (response.status === 204) {
        return undefined as T
    }

    const contentType = response.headers.get('Content-Type') ?? ''
    if (!contentType.includes('application/json')) {
        return response as unknown as T
    }

    return (await response.json()) as T
}

export async function fetchVersion(): Promise<string> {
    const data = await request<{ version: string }>('/version')
    return data.version
}

export async function listClients(): Promise<Client[]> {
    const data = await request<{ items: Client[] }>('/clients')
    return data.items
}

export async function createClient(payload: CreateClientRequest): Promise<Client> {
    return request<Client>('/clients', {
        method: 'POST',
        body: JSON.stringify(payload),
    })
}

export async function downloadClientCerts(name: string): Promise<Blob> {
    const response = await fetch(`${API_BASE}/clients/${encodeURIComponent(name)}/certs`)
    if (!response.ok) {
        let message = response.statusText
        try {
            const payload = (await response.json()) as ApiError
            if (payload.error) {
                message = payload.error
            }
        } catch {
            // ignore
        }
        throw new Error(message)
    }
    return response.blob()
}

export async function listIPFilters(): Promise<IPFilterRule[]> {
    const data = await request<{ items: IPFilterRule[] }>('/ip-filters')
    return data.items
}

export async function addIPFilter(payload: IPFilterMutateRequest): Promise<void> {
    await request('/ip-filters', {
        method: 'POST',
        body: JSON.stringify(payload),
    })
}

export async function deleteIPFilter(payload: IPFilterMutateRequest): Promise<void> {
    await request('/ip-filters', {
        method: 'DELETE',
        body: JSON.stringify(payload),
    })
}

export function formatPortRange(portStart: number, portEnd: number): string {
    if (portStart > 0 && portEnd > 0) {
        return `${portStart}-${portEnd}`
    }
    return '全部端口'
}

export function ruleFieldLabel(field: string, value?: string): string {
    switch (field) {
        case 'ip':
            return 'IP'
        case 'country':
            return '国家'
        case 'region':
            return '地区'
        case 'city':
            return '城市'
        case 'category':
            return value === 'ALL' ? '全部流量' : value === 'LOCAL' ? '本地网络' : '远程网络'
        default:
            return field
    }
}

export function ruleDisplayText(rule: IPFilterRule): string {
    if (rule.field === 'category') {
        return ruleFieldLabel('category', rule.value)
    }
    return `${ruleFieldLabel(rule.field)}: ${rule.value ?? '-'}`
}

export function toMutatePayload(
    status: 0 | 1,
    field: IPFilterMutateRequest['field'],
    value?: string,
): IPFilterMutateRequest {
    return {
        status,
        field,
        value: ['all', 'local', 'remote'].includes(field) ? undefined : value,
    }
}

export function fromRuleToMutate(rule: IPFilterRule): IPFilterMutateRequest {
    if (rule.field === 'category') {
        const category = (rule.value ?? 'ALL').toLowerCase() as 'all' | 'local' | 'remote'
        return {status: rule.status, field: category}
    }
    return {
        status: rule.status,
        field: rule.field,
        value: rule.value,
    }
}
