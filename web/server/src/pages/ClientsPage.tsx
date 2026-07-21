import {useCallback, useEffect, useMemo, useState} from 'react'
import {useNavigate} from 'react-router-dom'
import {
    Button,
    Drawer,
    Empty,
    Flex,
    Form,
    Input,
    InputNumber,
    message,
    Popconfirm,
    Space,
    Switch,
    Table,
    Tag,
    Typography,
} from 'antd'
import {
    DeleteOutlined,
    DownloadOutlined,
    PlusOutlined,
    ReloadOutlined,
    SafetyCertificateOutlined,
} from '@ant-design/icons'
import type {ColumnsType, TablePaginationConfig} from 'antd/es/table'
import {formatTimestamp, PageCard, PageHeader} from '@nextunnel/web-shared'
import {createClient, deleteClient, downloadCACert, listClients} from '../api'
import {formatPortRange, useI18n} from '../i18n'
import type {Client} from '../types'

interface CreateFormValues {
    name: string
    limitPorts: boolean
    portStart?: number
    portEnd?: number
}

const DRAWER_WIDTH = 480

export default function ClientsPage() {

    const {t} = useI18n()
    const navigate = useNavigate()
    const [clients, setClients] = useState<Client[]>([])
    const [loading, setLoading] = useState(false)
    const [createDrawerOpen, setCreateDrawerOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [downloadingCA, setDownloadingCA] = useState(false)
    const [deleting, setDeleting] = useState<string | null>(null)
    const [page, setPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)
    const [form] = Form.useForm<CreateFormValues>()

    const loadClients = useCallback(async () => {
        setLoading(true)
        try {
            setClients(await listClients())
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.loadFailed'))
        } finally {
            setLoading(false)
        }
    }, [t])

    useEffect(() => {
        void loadClients()
    }, [loadClients])

    const handleCreate = async (values: CreateFormValues) => {
        setSubmitting(true)
        try {
            await createClient({
                name: values.name.trim(),
                portStart: values.limitPorts ? values.portStart : 0,
                portEnd: values.limitPorts ? values.portEnd : 0,
            })
            message.success(t('clients.createSuccess', {name: values.name}))
            setCreateDrawerOpen(false)
            form.resetFields()
            await loadClients()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.createFailed'))
        } finally {
            setSubmitting(false)
        }
    }

    const handleDownloadCA = async () => {
        setDownloadingCA(true)
        try {
            const blob = await downloadCACert()
            const url = URL.createObjectURL(blob)
            const anchor = document.createElement('a')
            anchor.href = url
            anchor.download = 'ca.crt'
            anchor.click()
            URL.revokeObjectURL(url)
            message.success(t('clients.downloadCASuccess'))
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.downloadCAFailed'))
        } finally {
            setDownloadingCA(false)
        }
    }

    const handleDelete = async (name: string) => {
        setDeleting(name)
        try {
            await deleteClient(name)
            message.success(t('clients.deleteSuccess', {name}))
            await loadClients()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.deleteFailed'))
        } finally {
            setDeleting(null)
        }
    }

    const handleTableChange = (pagination: TablePaginationConfig) => {
        setPage(pagination.current ?? 1)
        setPageSize(pagination.pageSize ?? 10)
    }

    const columns: ColumnsType<Client> = useMemo(
        () => [
            {
                title: t('common.index'),
                key: 'index',
                width: 30,
                render: (_value, _record, index) => (page - 1) * pageSize + index + 1,
            },
            {
                title: t('clients.columnClient'),
                dataIndex: 'name',
                key: 'name',
                width: 150,
                ellipsis: true,
            },
            {
                title: t('clients.columnUserId'),
                dataIndex: 'id',
                key: 'id',
                width: 300,
                ellipsis: true,
                render: (id: string) => (
                    <Typography.Text
                        className="console-id-text"
                        copyable={{
                            text: id,
                            tooltips: [t('common.copy'), t('common.copied')],
                        }}
                    >
                        {id}
                    </Typography.Text>
                ),
            },
            {
                title: t('clients.columnPorts'),
                key: 'ports',
                width: 120,
                render: (_, record) => (
                    <Tag
                        bordered={false}
                        color={record.portStart > 0 ? 'processing' : 'default'}
                        style={{marginInlineEnd: 0}}
                    >
                        {formatPortRange(t, record.portStart, record.portEnd)}
                    </Tag>
                ),
            },
            {
                title: t('common.createdAt'),
                dataIndex: 'createdAt',
                key: 'createdAt',
                width: 180,
                render: (value: string) => (
                    <span className="console-id-text">{formatTimestamp(value)}</span>
                ),
            },
            {
                title: t('common.actions'),
                key: 'actions',
                width: 220,
                fixed: 'right',
                render: (_, record) => (
                    <Space size={8}>
                        <Button
                            size="small"
                            icon={<SafetyCertificateOutlined/>}
                            onClick={() => navigate(`/certs?userId=${encodeURIComponent(record.id)}`)}
                        >
                            {t('clients.manageCerts')}
                        </Button>
                        <Popconfirm
                            title={t('clients.deleteConfirmTitle')}
                            description={t('clients.deleteConfirmDesc')}
                            onConfirm={() => void handleDelete(record.name)}
                            okText={t('common.delete')}
                            cancelText={t('common.cancel')}
                            okButtonProps={{danger: true}}
                        >
                            <Button
                                size="small"
                                danger
                                icon={<DeleteOutlined/>}
                                loading={deleting === record.name}
                            >
                                {t('common.delete')}
                            </Button>
                        </Popconfirm>
                    </Space>
                ),
            },
        ],
        [t, deleting, page, pageSize, navigate],
    )

    return (
        <div className="console-page">
            <PageHeader description={t('clients.description')}/>

            <Flex className="console-page-actions" justify="flex-start">
                <Space wrap>
                    <Button type="primary" icon={<PlusOutlined/>} onClick={() => setCreateDrawerOpen(true)}>
                        {t('clients.createClient')}
                    </Button>
                    <Popconfirm
                        title={t('clients.downloadCAConfirmTitle')}
                        description={t('clients.downloadCAConfirmDesc')}
                        onConfirm={() => void handleDownloadCA()}
                        okText={t('common.confirm')}
                        cancelText={t('common.cancel')}
                        placement="bottom"
                    >
                        <Button icon={<DownloadOutlined/>} loading={downloadingCA}>
                            {t('clients.downloadCA')}
                        </Button>
                    </Popconfirm>
                    <Button icon={<ReloadOutlined/>} onClick={() => void loadClients()}>
                        {t('common.refresh')}
                    </Button>
                </Space>
            </Flex>

            <PageCard>
                <Table
                    rowKey="id"
                    size="small"
                    loading={loading}
                    columns={columns}
                    dataSource={clients}
                    tableLayout="fixed"
                    scroll={{x: 980}}
                    onChange={handleTableChange}
                    pagination={{
                        current: page,
                        pageSize,
                        showSizeChanger: true,
                        showTotal: (total) => t('common.total', {total}),
                        position: ['bottomLeft'],
                    }}
                    locale={{
                        emptyText: (
                            <div className="console-empty-hint">
                                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('clients.empty')}/>
                            </div>
                        ),
                    }}
                />
            </PageCard>

            <Drawer
                className="console-drawer"
                title={t('clients.createClient')}
                open={createDrawerOpen}
                onClose={() => {
                    setCreateDrawerOpen(false)
                    form.resetFields()
                }}
                width={DRAWER_WIDTH}
                destroyOnHidden
                footer={
                    <Flex justify="flex-end" gap={8}>
                        <Button
                            onClick={() => {
                                setCreateDrawerOpen(false)
                                form.resetFields()
                            }}
                        >
                            {t('common.cancel')}
                        </Button>
                        <Button type="primary" loading={submitting} onClick={() => form.submit()}>
                            {t('common.create')}
                        </Button>
                    </Flex>
                }
            >
                <Form
                    form={form}
                    layout="vertical"
                    initialValues={{limitPorts: false}}
                    onFinish={(values) => void handleCreate(values)}
                >
                    <Form.Item
                        name="name"
                        label={t('clients.nameLabel')}
                        rules={[{required: true, message: t('clients.nameRequired')}]}
                    >
                        <Input placeholder={t('clients.namePlaceholder')}/>
                    </Form.Item>
                    <Form.Item name="limitPorts" label={t('clients.limitPorts')} valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                    <Form.Item noStyle shouldUpdate={(prev, next) => prev.limitPorts !== next.limitPorts}>
                        {({getFieldValue}) =>
                            getFieldValue('limitPorts') ? (
                                <Flex gap={12}>
                                    <Form.Item
                                        name="portStart"
                                        label={t('clients.portStart')}
                                        rules={[{required: true, message: t('clients.portStartRequired')}]}
                                        style={{flex: 1, marginBottom: 0}}
                                    >
                                        <InputNumber min={1} max={65535} style={{width: '100%'}}/>
                                    </Form.Item>
                                    <Form.Item
                                        name="portEnd"
                                        label={t('clients.portEnd')}
                                        rules={[{required: true, message: t('clients.portEndRequired')}]}
                                        style={{flex: 1, marginBottom: 0}}
                                    >
                                        <InputNumber min={1} max={65535} style={{width: '100%'}}/>
                                    </Form.Item>
                                </Flex>
                            ) : null
                        }
                    </Form.Item>
                </Form>
            </Drawer>
        </div>
    )

}
