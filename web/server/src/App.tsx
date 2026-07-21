import {useMemo} from 'react'
import {BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate, useSearchParams,} from 'react-router-dom'
import {CloudServerOutlined, SafetyCertificateOutlined, SafetyOutlined} from '@ant-design/icons'
import {Flex, Layout, Menu, theme, Typography} from 'antd'
import {LanguageSwitcher} from '@nextunnel/web-shared'
import {useI18n} from './i18n'
import {index} from './routers'
import Clients from './pages/Clients.tsx'
import ClientCerts from './pages/ClientCerts.tsx'
import AccessControl from './pages/AccessControl.tsx'
import '@nextunnel/web-shared/styles/layout.css'

const SIDER_WIDTH = 220

const ROUTE_BY_KEY: Record<string, string> = {
    clients: index.clients,
    'client-certs': index.clientCerts,
    'access-control': index.accessControl,
}

function resolvePageMeta(pathname: string, t: ReturnType<typeof useI18n>['t']) {
    if (pathname.startsWith(index.accessControl)) {
        return {selectedKey: 'access-control', pageTitle: t('accessControl.title')}
    }
    if (pathname.startsWith(index.clientCerts)) {
        return {selectedKey: 'client-certs', pageTitle: t('certs.title')}
    }
    return {selectedKey: 'clients', pageTitle: t('clients.title')}
}

function LegacyCertsRedirect() {
    const [searchParams] = useSearchParams()
    const userId = searchParams.get('userId')
    const to = userId
        ? `${index.clientCerts}?clientId=${encodeURIComponent(userId)}`
        : index.clientCerts
    return <Navigate to={to} replace/>
}

function AppLayout() {
    const location = useLocation()
    const navigate = useNavigate()
    const {t} = useI18n()
    const {token: themeToken} = theme.useToken()

    const {Header, Sider, Content} = Layout

    const {selectedKey, pageTitle} = resolvePageMeta(location.pathname, t)

    const menuItems = useMemo(
        () => [
            {
                key: 'clients',
                icon: <CloudServerOutlined/>,
                label: t('nav.clients'),
            },
            {
                key: 'client-certs',
                icon: <SafetyCertificateOutlined/>,
                label: t('nav.certs'),
            },
            {
                key: 'access-control',
                icon: <SafetyOutlined/>,
                label: t('nav.accessControl'),
            },
        ],
        [t],
    )

    return (
        <Layout className="console-shell">
            <Header className="console-top-header">
                <Flex align="center" gap={10} className="console-top-brand">
                    <img src="/favicon.svg" alt="" className="console-sidebar-brand-icon"/>
                    <span className="console-sidebar-brand-text">Nextunnel Server</span>
                </Flex>
                <Flex align="center" gap={12}>
                    <LanguageSwitcher/>
                </Flex>
            </Header>

            <Layout className="console-body">
                <Sider
                    width={SIDER_WIDTH}
                    breakpoint="lg"
                    collapsedWidth={64}
                    theme="light"
                    className="console-sider"
                    style={{borderRight: `1px solid ${themeToken.colorBorderSecondary}`}}
                >
                    <div className="console-sider-inner">
                        <Menu
                            mode="inline"
                            inlineIndent={12}
                            selectedKeys={[selectedKey]}
                            items={menuItems}
                            onClick={({key}) => navigate(ROUTE_BY_KEY[key] ?? index.clients)}
                            className="console-sider-menu"
                        />
                    </div>
                </Sider>

                <Content className="console-content">
                    <div className="page-enter console-page-wrapper">
                        <header className="console-page-header">
                            <Typography.Title level={4} className="console-page-title">
                                {pageTitle}
                            </Typography.Title>
                        </header>
                        <div className="console-page-body">
                            <Routes>
                                <Route path="/" element={<Navigate to={index.clients} replace/>}/>
                                <Route path={index.clients} element={<Clients/>}/>
                                <Route path={index.clientCerts} element={<ClientCerts/>}/>
                                <Route path={index.accessControl} element={<AccessControl/>}/>
                                <Route path="/certs" element={<LegacyCertsRedirect/>}/>
                                <Route
                                    path="/ip-filters"
                                    element={<Navigate to={index.accessControl} replace/>}
                                />
                            </Routes>
                        </div>
                    </div>
                </Content>
            </Layout>
        </Layout>
    )
}

export default function App() {
    return (
        <BrowserRouter>
            <AppLayout/>
        </BrowserRouter>
    )
}
