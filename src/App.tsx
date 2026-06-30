import {useEffect, useState} from 'react'
import {BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate} from 'react-router-dom'
import {CloudServerOutlined, SafetyOutlined} from '@ant-design/icons'
import {Layout, Menu, Spin, theme, Typography} from 'antd'
import {fetchVersion} from './api'
import ClientsPage from './pages/ClientsPage'
import IpFilterPage from './pages/IpFilterPage'

const {Header, Sider, Content, Footer} = Layout

function AppLayout() {
    const location = useLocation()
    const navigate = useNavigate()
    const {token} = theme.useToken()
    const [version, setVersion] = useState<string>()
    const [versionLoading, setVersionLoading] = useState(true)

    useEffect(() => {
        void fetchVersion()
            .then(setVersion)
            .catch(() => setVersion(undefined))
            .finally(() => setVersionLoading(false))
    }, [])

    const selectedKey = location.pathname.startsWith('/ip-filters') ? 'ip-filters' : 'clients'

    return (
        <Layout style={{minHeight: '100vh'}}>
            <Sider breakpoint="lg" collapsedWidth={64} theme="dark">
                <div
                    style={{
                        height: 64,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        color: '#fff',
                        fontWeight: 600,
                        fontSize: 16,
                        padding: '0 12px',
                        textAlign: 'center',
                    }}
                >
                    nextunnel-server
                </div>
                <Menu
                    theme="dark"
                    mode="inline"
                    selectedKeys={[selectedKey]}
                    items={[
                        {
                            key: 'clients',
                            icon: <CloudServerOutlined/>,
                            label: '客户端管理',
                            onClick: () => navigate('/clients'),
                        },
                        {
                            key: 'ip-filters',
                            icon: <SafetyOutlined/>,
                            label: '访问控制',
                            onClick: () => navigate('/ip-filters'),
                        },
                    ]}
                />
            </Sider>
            <Layout>
                <Content style={{margin: 24}}>
                    <div
                        style={{
                            background: token.colorBgContainer,
                            borderRadius: token.borderRadiusLG,
                            padding: 24,
                            minHeight: 360,
                        }}
                    >
                        <Routes>
                            <Route path="/" element={<Navigate to="/clients" replace/>}/>
                            <Route path="/clients" element={<ClientsPage/>}/>
                            <Route path="/ip-filters" element={<IpFilterPage/>}/>
                        </Routes>
                    </div>
                </Content>
                <Footer style={{textAlign: 'center'}}>
                    {versionLoading ? (
                        <Spin size="small"/>
                    ) : (
                        <Typography.Text type="secondary">
                            nextunnel-server {version ?? 'API 未连接'}
                        </Typography.Text>
                    )}
                </Footer>
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
