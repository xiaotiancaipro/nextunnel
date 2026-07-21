import path from 'node:path'
import {fileURLToPath} from 'node:url'
import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

const rootDir = path.dirname(fileURLToPath(import.meta.url))
const sharedRoot = path.resolve(rootDir, '../shared')

export default defineConfig({
    plugins: [react()],
    publicDir: path.resolve(sharedRoot, 'public'),
    resolve: {
        alias: [
            {
                find: /^@nextunnel\/web-shared\/(.*)$/,
                replacement: path.resolve(sharedRoot, 'src/$1'),
            },
            {
                find: '@nextunnel/web-shared',
                replacement: path.resolve(sharedRoot, 'src/index.ts'),
            },
        ],
    },
    server: {
        fs: {
            allow: [rootDir, sharedRoot],
        },
    },
    build: {
        outDir: '../../internal/client/controllers/dist',
        emptyOutDir: true,
    },
})
