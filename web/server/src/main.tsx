import {StrictMode} from 'react'
import {createRoot} from 'react-dom/client'
import App from './App.tsx'
import {I18nProvider} from './i18n'
import '@nextunnel/web-shared/styles/index.css'
import '@nextunnel/web-shared/styles/page.css'
import '@xyflow/react/dist/style.css'

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <I18nProvider>
            <App/>
        </I18nProvider>
    </StrictMode>,
)
