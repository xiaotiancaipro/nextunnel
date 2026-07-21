export const index = {
    clients: '/clients',
    clientCerts: '/client-certs',
    accessControl: '/access-control',
} as const

export type RouteKey = keyof typeof index
