import {useCallback, useEffect, useMemo, useState} from 'react'
import {useSearchParams} from 'react-router-dom'
import {
    Button,
    DatePicker,
    Drawer,
    Empty,
    Flex,
    Form,
    message,
    Popconfirm,
    Select,
    Space,
    Switch,
    Table,
    Tag,
} from 'antd'
import {DeleteOutlined, DownloadOutlined, PlusOutlined, ReloadOutlined} from '@ant-design/icons'
import type {ColumnsType, TablePaginationConfig} from 'antd/es/table'
import dayjs, {type Dayjs} from 'dayjs'
import {formatTimestamp, PageCard, PageHeader} from '@nextunnel/web-shared'
import {createClientCert, deleteClientCert, downloadClientCert, listClientCerts, listClients,} from '../api'
import {useI18n} from '../i18n'
import type {Client, ClientCert} from '../types'

interface AddCertFormValues {
    clientId: string
    neverExpires: boolean
    expiresAt?: Dayjs
}

interface CertRow extends ClientCert {
    clientId: string
    clientName: string
}

const DRAWER_WIDTH = 480

export default function CertsPage() {

    const {t} = useI18n()
    const [searchParams, setSearchParams] = useSearchParams()
    const userIdFilter = searchParams.get('userId')?.trim() || undefined

    const [clients, setClients] = useState<Client[]>([])
    const [certItems, setCertItems] = useState<CertRow[]>([])
    const [loading, setLoading] = useState(false)
    const [addDrawerOpen, setAddDrawerOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [deleting, setDeleting] = useState<string | null>(null)
    const [downloading, setDownloading] = useState<string | null>(null)
    const [page, setPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)
    const [form] = Form.useForm<AddCertFormValues>()

    const loadData = useCallback(async () => {
        setLoading(true)
        try {
            const clientList = await listClients()
            setClients(clientList)

            const nested = await Promise.all(
                clientList.map(async (client) => {
                    try {
                        const certs = await listClientCerts(client.name)
                        return certs.map((cert) => ({
                            ...cert,
                            clientId: client.id,
                            clientName: client.name,
                        }))
                    } catch {
                        return [] as CertRow[]
                    }
                }),
            )
            setCertItems(nested.flat())
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('certs.loadFailed'))
        } finally {
            setLoading(false)
        }
    }, [t])

    useEffect(() => {
        void loadData()
    }, [loadData])

    useEffect(() => {
        setPage(1)
    }, [userIdFilter])

    const filteredItems = useMemo(() => {
        if (!userIdFilter) {
            return certItems
        }
        return certItems.filter((item) => item.clientId === userIdFilter)
    }, [certItems, userIdFilter])

    const clientOptions = useMemo(
        () =>
            clients.map((client) => ({
                value: client.id,
                label: `${client.name} (${client.id})`,
            })),
        [clients],
    )

    const setUserIdFilter = (userId?: string) => {
        if (userId) {
            setSearchParams({userId})
        } else {
            setSearchParams({})
        }
    }

    const closeAddDrawer = () => {
        setAddDrawerOpen(false)
        form.resetFields()
    }

    const openAddDrawer = () => {
        form.resetFields()
        form.setFieldsValue({
            neverExpires: true,
            clientId: userIdFilter,
        })
        setAddDrawerOpen(true)
    }

    const resolveClientName = (clientId: string) => clients.find((client) => client.id === clientId)?.name

    const handleAddCert = async (values: AddCertFormValues) => {
        const clientName = resolveClientName(values.clientId)
        if (!clientName) {
            message.error(t('certs.clientRequired'))
            return
        }
        setSubmitting(true)
        try {
            const payload = values.neverExpires
                ? {}
                : {expiresAt: values.expiresAt?.toDate().toISOString()}
            const created = await createClientCert(clientName, payload)
            message.success(t('clients.certCreateSuccess', {id: created.id}))
            closeAddDrawer()
            await loadData()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certCreateFailed'))
        } finally {
            setSubmitting(false)
        }
    }

    const handleDownloadCert = async (record: CertRow) => {
        setDownloading(record.id)
        try {
            const blob = await downloadClientCert(record.clientName, record.id)
            const url = URL.createObjectURL(blob)
            const anchor = document.createElement('a')
            anchor.href = url
            anchor.download = `${record.clientName}-${record.id}-certs.zip`
            anchor.click()
            URL.revokeObjectURL(url)
            message.success(t('clients.downloadSuccess', {name: record.clientName}))
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certFailed'))
        } finally {
            setDownloading(null)
        }
    }

    const handleDeleteCert = async (record: CertRow) => {
        setDeleting(record.id)
        try {
            await deleteClientCert(record.clientName, record.id)
            message.success(t('clients.certDeleteSuccess', {id: record.id}))
            await loadData()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certDeleteFailed'))
        } finally {
            setDeleting(null)
        }
    }

    const handleTableChange = (pagination: TablePaginationConfig) => {
        setPage(pagination.current ?? 1)
        setPageSize(pagination.pageSize ?? 10)
    }

    const columns: ColumnsType<CertRow> = useMemo(
        () => [
            {
                title: t('common.index'),
                key: 'index',
                width: 30,
                render: (_value, _record, index) => (page - 1) * pageSize + index + 1,
            },
            {
                title: t('certs.columnClient'),
                dataIndex: 'clientName',
                key: 'clientName',
                width: 150,
                ellipsis: true,
            },
            {
                title: t('certs.columnCertId'),
                dataIndex: 'id',
                key: 'id',
                width: 300,
                ellipsis: true,
                render: (id: string) => <span className="console-id-text">{id}</span>,
            },
            {
                title: t('clients.certExpiresAt'),
                dataIndex: 'expiresAt',
                key: 'expiresAt',
                width: 120,
                render: (value?: string | null) =>
                    value ? (
                        <span className="console-id-text">{formatTimestamp(value)}</span>
                    ) : (
                        <Tag bordered={false} color="success">
                            {t('clients.certNeverExpires')}
                        </Tag>
                    ),
            },
            {
                title: t('common.createdAt'),
                dataIndex: 'createdAt',
                key: 'createdAt',
                width: 180,
                render: (value: string) => <span className="console-id-text">{formatTimestamp(value)}</span>,
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
                            icon={<DownloadOutlined/>}
                            loading={downloading === record.id}
                            onClick={() => void handleDownloadCert(record)}
                        >
                            {t('common.download')}
                        </Button>
                        <Popconfirm
                            title={t('clients.certDeleteConfirmTitle')}
                            description={t('clients.certDeleteConfirmDesc')}
                            onConfirm={() => void handleDeleteCert(record)}
                            okText={t('common.delete')}
                            cancelText={t('common.cancel')}
                            okButtonProps={{danger: true}}
                        >
                            <Button
                                size="small"
                                danger
                                icon={<DeleteOutlined/>}
                                loading={deleting === record.id}
                            >
                                {t('common.delete')}
                            </Button>
                        </Popconfirm>
                    </Space>
                ),
            },
        ],
        [t, downloading, deleting, page, pageSize],
    )

    return (
        <div className="console-page">
            <PageHeader description={t('certs.description')}/>

            <Flex className="console-page-actions" justify="space-between" align="center" wrap="wrap" gap={12}>
                <Space wrap>
                    <Button type="primary" icon={<PlusOutlined/>} onClick={openAddDrawer}>
                        {t('clients.addCert')}
                    </Button>
                    <Button icon={<ReloadOutlined/>} onClick={() => void loadData()}>
                        {t('common.refresh')}
                    </Button>
                </Space>
                <Space wrap>
                    <Select
                        allowClear
                        showSearch
                        optionFilterProp="label"
                        placeholder={t('certs.filterUserId')}
                        style={{minWidth: 280}}
                        value={userIdFilter}
                        options={clientOptions}
                        onChange={(value) => setUserIdFilter(value)}
                    />
                </Space>
            </Flex>

            <PageCard>
                <Table
                    rowKey={(record) => `${record.clientId}-${record.id}`}
                    size="small"
                    loading={loading}
                    columns={columns}
                    dataSource={filteredItems}
                    tableLayout="fixed"
                    scroll={{x: 1200}}
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
                                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('certs.empty')}/>
                            </div>
                        ),
                    }}
                />
            </PageCard>

            <Drawer
                className="console-drawer"
                title={t('clients.addCert')}
                open={addDrawerOpen}
                onClose={closeAddDrawer}
                width={DRAWER_WIDTH}
                destroyOnHidden
                footer={
                    <Flex justify="flex-end" gap={8}>
                        <Button onClick={closeAddDrawer}>{t('common.cancel')}</Button>
                        <Button type="primary" loading={submitting} onClick={() => form.submit()}>
                            {t('common.create')}
                        </Button>
                    </Flex>
                }
            >
                <Form
                    form={form}
                    layout="vertical"
                    initialValues={{neverExpires: true}}
                    onFinish={(values) => void handleAddCert(values)}
                >
                    <Form.Item
                        name="clientId"
                        label={t('certs.columnClient')}
                        rules={[{required: true, message: t('certs.clientRequired')}]}
                    >
                        <Select
                            showSearch
                            optionFilterProp="label"
                            placeholder={t('certs.selectClient')}
                            options={clientOptions}
                        />
                    </Form.Item>
                    <Form.Item name="neverExpires" label={t('clients.certNeverExpires')} valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                    <Form.Item noStyle shouldUpdate={(prev, next) => prev.neverExpires !== next.neverExpires}>
                        {({getFieldValue}) =>
                            !getFieldValue('neverExpires') ? (
                                <Form.Item
                                    name="expiresAt"
                                    label={t('clients.certExpiresAt')}
                                    rules={[{required: true, message: t('clients.certExpiresAtRequired')}]}
                                >
                                    <DatePicker
                                        showTime
                                        style={{width: '100%'}}
                                        disabledDate={(current) => current && current < dayjs().startOf('day')}
                                    />
                                </Form.Item>
                            ) : null
                        }
                    </Form.Item>
                </Form>
            </Drawer>
        </div>
    )

}
