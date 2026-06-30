import {useCallback, useEffect, useMemo, useState} from 'react'
import {Button, Empty, Form, Input, InputNumber, message, Modal, Popconfirm, Space, Switch, Table, Tag} from 'antd'
import {DeleteOutlined, DownloadOutlined, PlusOutlined, ReloadOutlined} from '@ant-design/icons'
import type {ColumnsType} from 'antd/es/table'
import PageCard from '../components/PageCard'
import PageHeader from '../components/PageHeader'
import {createClient, deleteClient, downloadCACert, downloadClientCerts, listClients} from '../api'
import {formatPortRange, useI18n} from '../i18n'
import type {Client} from '../types'

interface CreateFormValues {
    name: string
    limitPorts: boolean
    portStart?: number
    portEnd?: number
}

function clientInitial(name: string) {
    return name.trim().charAt(0).toUpperCase() || '?'
}

export default function ClientsPage() {
    const {t} = useI18n()
    const [clients, setClients] = useState<Client[]>([])
    const [loading, setLoading] = useState(false)
    const [modalOpen, setModalOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [downloading, setDownloading] = useState<string | null>(null)
    const [downloadingCA, setDownloadingCA] = useState(false)
    const [deleting, setDeleting] = useState<string | null>(null)
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
            setModalOpen(false)
            form.resetFields()
            await loadClients()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.createFailed'))
        } finally {
            setSubmitting(false)
        }
    }

    const handleDownloadCerts = async (name: string) => {
        setDownloading(name)
        try {
            const blob = await downloadClientCerts(name)
            const url = URL.createObjectURL(blob)
            const anchor = document.createElement('a')
            anchor.href = url
            anchor.download = `${name}-certs.zip`
            anchor.click()
            URL.revokeObjectURL(url)
            message.success(t('clients.downloadSuccess', {name}))
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('clients.certFailed'))
        } finally {
            setDownloading(null)
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
                    <span
                        className="console-id-text">{value.replace('T', ' ').replace(/\.\d+Z$/, '').replace('Z', '')}</span>
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
                            icon={<DownloadOutlined/>}
                            loading={downloading === record.name}
                            onClick={() => void handleDownloadCerts(record.name)}
                            style={{paddingInline: 0, whiteSpace: 'nowrap'}}
                        >
                            {t('clients.generateCerts')}
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
        [t, downloading, deleting],
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
        </div>
    )
}
