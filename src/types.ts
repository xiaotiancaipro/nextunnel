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

export type IPFilterField =
    | 'ip'
    | 'country'
    | 'region'
    | 'city'
    | 'all'
    | 'local'
    | 'remote'

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
