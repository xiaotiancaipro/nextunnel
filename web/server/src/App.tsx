import {useMemo} from 'react'
import {BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate} from 'react-router-dom'
import {CloudServerOutlined, SafetyCertificateOutlined, SafetyOutlined} from '@ant-design/icons'
import {Flex, Layout, Menu, theme, Typography} from 'antd'
import {LanguageSwitcher} from '@nextunnel/web-shared'
import {useI18n} from './i18n'
import ClientsPage from './pages/ClientsPage'
import CertsPage from './pages/CertsPage'
import IpFilterPage from './pages/IpFilterPage'
import '@nextunnel/web-shared/styles/layout.css'

const SIDER_WIDTH = 220

const ROUTE_BY_KEY: Record<string, string> = {
    clients: '/clients',
    certs: '/certs',
    'ip-filters': '/ip-filters',
}

function resolvePageMeta(pathname: string, t: ReturnType<typeof useI18n>['t']) {
    if (pathname.startsWith('/ip-filters')) {
        return {selectedKey: 'ip-filters', pageTitle: t('ipFilters.title')}
    }
    if (pathname.startsWith('/certs')) {
        return {selectedKey: 'certs', pageTitle: t('certs.title')}
    }
    return {selectedKey: 'clients', pageTitle: t('clients.title')}
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
                key: 'certs',
                icon: <SafetyCertificateOutlined/>,
                label: t('nav.certs'),
            },
            {
                key: 'ip-filters',
                icon: <SafetyOutlined/>,
                label: t('nav.ipFilters'),
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
                            onClick={({key}) => navigate(ROUTE_BY_KEY[key] ?? '/clients')}
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
                                <Route path="/" element={<Navigate to="/clients" replace/>}/>
                                <Route path="/clients" element={<ClientsPage/>}/>
                                <Route path="/certs" element={<CertsPage/>}/>
                                <Route path="/ip-filters" element={<IpFilterPage/>}/>
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
