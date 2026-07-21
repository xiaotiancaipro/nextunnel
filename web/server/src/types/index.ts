export type IPFilterField =
    | 'ip'
    | 'country'
    | 'region'
    | 'city'
    | 'all'
    | 'local'
    | 'remote'

export interface Client {
    id: string
    name: string
    portStart: number
    portEnd: number
    createdAt: string
}

export interface IPFilterRule {
    id: string
    status: 0 | 1
    field: 'ip' | 'country' | 'region' | 'city' | 'category'
    value?: string
    createdAt: string
}

export interface ClientCert {
    id: string
    createdAt: string
    expiresAt?: string | null
    serial: string
}

export interface CreateClientCertRequest {
    expiresAt?: string | null
}

export interface CreateClientRequest {
    name: string
    portStart?: number
    portEnd?: number
}

export interface IPFilterMutateRequest {
    status: 0 | 1
    field: IPFilterField
    value?: string
}

export interface ApiError {
    error: string
}
