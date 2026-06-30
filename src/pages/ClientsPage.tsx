import {useCallback, useEffect, useMemo, useState} from 'react'
import {
    Button,
    DatePicker,
    Divider,
    Empty,
    Form,
    Input,
    InputNumber,
    message,
    Modal,
    Popconfirm,
    Space,
    Switch,
    Table,
    Tag,
} from 'antd'
import {
    DeleteOutlined,
    DownloadOutlined,
    PlusOutlined,
    ReloadOutlined,
    SafetyCertificateOutlined
} from '@ant-design/icons'
import type {ColumnsType} from 'antd/es/table'
import dayjs, {type Dayjs} from 'dayjs'
import PageCard from '../components/PageCard'
import PageHeader from '../components/PageHeader'
import {
    createClient,
    createClientCert,
    deleteClient,
    deleteClientCert,
    downloadCACert,
    downloadClientCert,
    listClientCerts,
    listClients,
} from '../api'
import {formatPortRange, useI18n} from '../i18n'
import type {Client, ClientCert} from '../types'

interface CreateFormValues {
    name: string
    limitPorts: boolean
    portStart?: number
    portEnd?: number
}

interface AddCertFormValues {
    neverExpires: boolean
    expiresAt?: Dayjs
}

function clientInitial(name: string) {
    return name.trim().charAt(0).toUpperCase() || '?'
}

function formatTimestamp(value: string) {
    return value.replace('T', ' ').replace(/\.\d+Z$/, '').replace('Z', '')
}

export default function ClientsPage() {
    const {t} = useI18n()
    const [clients, setClients] = useState<Client[]>([])
    const [loading, setLoading] = useState(false)
    const [modalOpen, setModalOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [downloadingCA, setDownloadingCA] = useState(false)
    const [deleting, setDeleting] = useState<string | null>(null)
    const [form] = Form.useForm<CreateFormValues>()

    const [certModalOpen, setCertModalOpen] = useState(false)
    const [certClientName, setCertClientName] = useState<string | null>(null)
    const [certItems, setCertItems] = useState<ClientCert[]>([])
    const [certLoading, setCertLoading] = useState(false)
    const [certSubmitting, setCertSubmitting] = useState(false)
    const [certDeleting, setCertDeleting] = useState<string | null>(null)
    const [certDownloading, setCertDownloading] = useState<string | null>(null)
    const [certForm] = Form.useForm<AddCertFormValues>()

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

    const loadCerts = useCallback(async (name: string) => {
        setCertLoading(true)
        try {
            setCertItems(await listClientCerts(name))
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certLoadFailed'))
        } finally {
            setCertLoading(false)
        }
    }, [t])

    useEffect(() => {
        void loadClients()
    }, [loadClients])

    useEffect(() => {
        if (certModalOpen && certClientName) {
            void loadCerts(certClientName)
        }
    }, [certModalOpen, certClientName, loadCerts])

    const handleCreate = async (values: CreateFormValues) => {
        setSubmitting(true)
        try {
            await createClient({
                name: values.name.trim(),
                portStart: values.limitPorts ? values.portStart : 0,
                portEnd: values.limitPorts ? values.portEnd : 0,
            })
            message.success(t('clients.createSuccess', {name: values.name}))
            setModalOpen(false)
            form.resetFields()
            await loadClients()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.createFailed'))
        } finally {
            setSubmitting(false)
        }
    }

    const openCertModal = (name: string) => {
        setCertClientName(name)
        setCertModalOpen(true)
        certForm.resetFields()
    }

    const closeCertModal = () => {
        setCertModalOpen(false)
        setCertClientName(null)
        setCertItems([])
        certForm.resetFields()
    }

    const handleAddCert = async (values: AddCertFormValues) => {
        if (!certClientName) {
            return
        }
        setCertSubmitting(true)
        try {
            const payload = values.neverExpires
                ? {}
                : {expiresAt: values.expiresAt?.toDate().toISOString()}
            const created = await createClientCert(certClientName, payload)
            message.success(t('clients.certCreateSuccess', {id: created.id}))
            certForm.resetFields()
            await loadCerts(certClientName)
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certCreateFailed'))
        } finally {
            setCertSubmitting(false)
        }
    }

    const handleDownloadCert = async (certId: string) => {
        if (!certClientName) {
            return
        }
        setCertDownloading(certId)
        try {
            const blob = await downloadClientCert(certClientName, certId)
            const url = URL.createObjectURL(blob)
            const anchor = document.createElement('a')
            anchor.href = url
            anchor.download = `${certClientName}-${certId}-certs.zip`
            anchor.click()
            URL.revokeObjectURL(url)
            message.success(t('clients.downloadSuccess', {name: certClientName}))
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certFailed'))
        } finally {
            setCertDownloading(null)
        }
    }

    const handleDeleteCert = async (certId: string) => {
        if (!certClientName) {
            return
        }
        setCertDeleting(certId)
        try {
            await deleteClientCert(certClientName, certId)
            message.success(t('clients.certDeleteSuccess', {id: certId}))
            await loadCerts(certClientName)
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certDeleteFailed'))
        } finally {
            setCertDeleting(null)
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

    const certColumns: ColumnsType<ClientCert> = useMemo(
        () => [
            {
                title: 'ID',
                dataIndex: 'id',
                key: 'id',
                width: 220,
                ellipsis: true,
                render: (id: string) => <span className="console-id-text">{id}</span>,
            },
            {
                title: t('common.createdAt'),
                dataIndex: 'createdAt',
                key: 'createdAt',
                width: 168,
                render: (value: string) => <span className="console-id-text">{formatTimestamp(value)}</span>,
            },
            {
                title: t('clients.certExpiresAt'),
                dataIndex: 'expiresAt',
                key: 'expiresAt',
                width: 168,
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
                title: t('common.actions'),
                key: 'actions',
                width: 160,
                fixed: 'right',
                render: (_, record) => (
                    <Space size={12}>
                        <Button
                            type="link"
                            icon={<DownloadOutlined/>}
                            loading={certDownloading === record.id}
                            onClick={() => void handleDownloadCert(record.id)}
                            style={{paddingInline: 0, whiteSpace: 'nowrap'}}
                        >
                            {t('common.download')}
                        </Button>
                        <Popconfirm
                            title={t('clients.certDeleteConfirmTitle')}
                            description={t('clients.certDeleteConfirmDesc')}
                            onConfirm={() => void handleDeleteCert(record.id)}
                            okText={t('common.delete')}
                            cancelText={t('common.cancel')}
                            okButtonProps={{danger: true}}
                        >
                            <Button
                                type="link"
                                danger
                                icon={<DeleteOutlined/>}
                                loading={certDeleting === record.id}
                                style={{paddingInline: 0, whiteSpace: 'nowrap'}}
                            >
                                {t('common.delete')}
                            </Button>
                        </Popconfirm>
                    </Space>
                ),
            },
        ],
        [t, certDownloading, certDeleting, certClientName],
    )

    const columns: ColumnsType<Client> = useMemo(
        () => [
            {
                title: t('clients.columnClient'),
                dataIndex: 'name',
                key: 'name',
                width: 168,
                ellipsis: true,
                render: (name: string) => (
                    <span className="console-name-chip">
            <span className="console-name-chip__avatar">{clientInitial(name)}</span>
            <strong>{name}</strong>
          </span>
                ),
            },
            {
                title: 'ID',
                dataIndex: 'id',
                key: 'id',
                width: 280,
                ellipsis: true,
                render: (id: string) => <span className="console-id-text">{id}</span>,
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
                width: 176,
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
                    <Space size={12}>
                        <Button
                            type="link"
                            icon={<SafetyCertificateOutlined/>}
                            onClick={() => openCertModal(record.name)}
                            style={{paddingInline: 0, whiteSpace: 'nowrap'}}
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
                                type="link"
                                danger
                                icon={<DeleteOutlined/>}
                                loading={deleting === record.name}
                                style={{paddingInline: 0, whiteSpace: 'nowrap'}}
                            >
                                {t('common.delete')}
                            </Button>
                        </Popconfirm>
                    </Space>
                ),
            },
        ],
        [t, deleting],
    )

    return (
        <div className="console-page">
            <PageHeader
                description={t('clients.description')}
                badge={<span className="console-count-badge">{t('clients.count', {count: clients.length})}</span>}
                extra={
                    <Space>
                        <Button icon={<ReloadOutlined/>} onClick={() => void loadClients()}>
                            {t('common.refresh')}
                        </Button>
                        <Button
                            icon={<DownloadOutlined/>}
                            loading={downloadingCA}
                            onClick={() => void handleDownloadCA()}
                        >
                            {t('clients.downloadCA')}
                        </Button>
                        <Button type="primary" icon={<PlusOutlined/>} onClick={() => setModalOpen(true)}>
                            {t('clients.createClient')}
                        </Button>
                    </Space>
                }
            />

            <PageCard>
                <Table
                    rowKey="id"
                    size="small"
                    loading={loading}
                    columns={columns}
                    dataSource={clients}
                    tableLayout="fixed"
                    scroll={{x: 980}}
                    pagination={{pageSize: 10, showSizeChanger: true, showTotal: (total) => t('common.total', {total})}}
                    locale={{
                        emptyText: (
                            <div className="console-empty-hint">
                                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('clients.empty')}/>
                            </div>
                        ),
                    }}
                />
            </PageCard>

            <Modal
                className="console-modal"
                title={t('clients.createClient')}
                open={modalOpen}
                onCancel={() => {
                    setModalOpen(false)
                    form.resetFields()
                }}
                onOk={() => form.submit()}
                confirmLoading={submitting}
                destroyOnHidden
                okText={t('common.create')}
                cancelText={t('common.cancel')}
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
                                <Space style={{display: 'flex'}} align="start">
                                    <Form.Item
                                        name="portStart"
                                        label={t('clients.portStart')}
                                        rules={[{required: true, message: t('clients.portStartRequired')}]}
                                    >
                                        <InputNumber min={1} max={65535} style={{width: 148}}/>
                                    </Form.Item>
                                    <Form.Item
                                        name="portEnd"
                                        label={t('clients.portEnd')}
                                        rules={[{required: true, message: t('clients.portEndRequired')}]}
                                    >
                                        <InputNumber min={1} max={65535} style={{width: 148}}/>
                                    </Form.Item>
                                </Space>
                            ) : null
                        }
                    </Form.Item>
                </Form>
            </Modal>

            <Modal
                className="console-modal"
                title={t('clients.certModalTitle', {name: certClientName ?? ''})}
                open={certModalOpen}
                onCancel={closeCertModal}
                footer={null}
                width={860}
                destroyOnHidden
            >
                <Table
                    rowKey="id"
                    size="small"
                    loading={certLoading}
                    columns={certColumns}
                    dataSource={certItems}
                    tableLayout="fixed"
                    scroll={{x: 760}}
                    pagination={false}
                    locale={{
                        emptyText: (
                            <div className="console-empty-hint">
                                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('clients.certEmpty')}/>
                            </div>
                        ),
                    }}
                />

                <Divider style={{margin: '20px 0 16px'}}/>

                <Form
                    form={certForm}
                    layout="vertical"
                    initialValues={{neverExpires: true}}
                    onFinish={(values) => void handleAddCert(values)}
                >
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
                    <Button type="primary" icon={<PlusOutlined/>} htmlType="submit" loading={certSubmitting}>
                        {t('clients.addCert')}
                    </Button>
                </Form>
            </Modal>
        </div>
    )
}
