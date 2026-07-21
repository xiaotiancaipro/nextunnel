import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
    plugins: [react()],
    build: {
        outDir: '../../internal/client/controllers/dist',
        emptyOutDir: true,
    },
})
