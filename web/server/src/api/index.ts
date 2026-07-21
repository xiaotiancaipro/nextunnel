import type {
    ApiError,
    Client,
    ClientCert,
    CreateClientCertRequest,
    CreateClientRequest,
    IPFilterMutateRequest,
    IPFilterRule,
} from '../types'

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

export async function deleteClient(name: string): Promise<void> {
    await request(`/clients/${encodeURIComponent(name)}`, {
        method: 'DELETE',
    })
}

export async function listClientCerts(name: string): Promise<ClientCert[]> {
    const data = await request<{ items: ClientCert[] }>(`/clients/${encodeURIComponent(name)}/sharedcerts`)
    return data.items
}

export async function createClientCert(name: string, payload: CreateClientCertRequest = {}): Promise<ClientCert> {
    return request<ClientCert>(`/clients/${encodeURIComponent(name)}/sharedcerts`, {
        method: 'POST',
        body: JSON.stringify(payload),
    })
}

export async function deleteClientCert(name: string, certId: string): Promise<void> {
    await request(`/clients/${encodeURIComponent(name)}/sharedcerts/${encodeURIComponent(certId)}`, {
        method: 'DELETE',
    })
}

export async function downloadClientCert(name: string, certId: string): Promise<Blob> {
    const response = await fetch(
        `${API_BASE}/clients/${encodeURIComponent(name)}/sharedcerts/${encodeURIComponent(certId)}/download`,
    )
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

export async function downloadCACert(): Promise<Blob> {
    const response = await fetch(`${API_BASE}/ca`)
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
