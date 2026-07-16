import {useCallback, useEffect, useMemo, useState} from 'react'
import {Button, Empty, Form, Input, message, Modal, Popconfirm, Select, Space, Table, Tag} from 'antd'
import {DeleteOutlined, PlusOutlined, ReloadOutlined} from '@ant-design/icons'
import type {ColumnsType} from 'antd/es/table'
import PageCard from '../components/PageCard'
import PageHeader from '../components/PageHeader'
import {addIPFilter, deleteIPFilter, fromRuleToMutate, listIPFilters, toMutatePayload} from '../api'
import {ruleDisplayText, useI18n} from '../i18n'
import type {IPFilterField, IPFilterRule} from '../types'
import {formatTimestamp} from '../utils/formatTime'

interface AddFormValues {
    status: 0 | 1
    field: IPFilterField
    value?: string
}

const categoryFields = new Set<IPFilterField>(['all', 'local', 'remote'])

export default function IpFilterPage() {
    const {t} = useI18n()
    const [rules, setRules] = useState<IPFilterRule[]>([])
    const [loading, setLoading] = useState(false)
    const [modalOpen, setModalOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [deletingId, setDeletingId] = useState<string | null>(null)
    const [form] = Form.useForm<AddFormValues>()

    const fieldOptions = useMemo(
        () => [
            {label: t('ipFilters.field.ip'), value: 'ip' as const},
            {label: t('ipFilters.field.country'), value: 'country' as const},
            {label: t('ipFilters.field.region'), value: 'region' as const},
            {label: t('ipFilters.field.city'), value: 'city' as const},
            {label: t('ipFilters.field.all'), value: 'all' as const},
            {label: t('ipFilters.field.local'), value: 'local' as const},
            {label: t('ipFilters.field.remote'), value: 'remote' as const},
        ],
        [t],
    )

    const loadRules = useCallback(async () => {
        setLoading(true)
        try {
            setRules(await listIPFilters())
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('ipFilters.loadFailed'))
        } finally {
            setLoading(false)
        }
    }, [t])

    useEffect(() => {
        void loadRules()
    }, [loadRules])

    const handleAdd = async (values: AddFormValues) => {
        setSubmitting(true)
        try {
            await addIPFilter(toMutatePayload(values.status, values.field, values.value?.trim()))
            message.success(t('ipFilters.addSuccess'))
            setModalOpen(false)
            form.resetFields()
            await loadRules()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('ipFilters.addFailed'))
        } finally {
            setSubmitting(false)
        }
    }

    const handleDelete = async (rule: IPFilterRule) => {
        setDeletingId(rule.id)
        try {
            await deleteIPFilter(fromRuleToMutate(rule))
            message.success(t('ipFilters.deleteSuccess'))
            await loadRules()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('ipFilters.deleteFailed'))
        } finally {
            setDeletingId(null)
        }
    }

    const allowCount = rules.filter((rule) => rule.status === 1).length
    const blockCount = rules.length - allowCount

    const columns: ColumnsType<IPFilterRule> = useMemo(
        () => [
            {
                title: t('ipFilters.columnAction'),
                dataIndex: 'status',
                key: 'status',
                width: 80,
                render: (status: 0 | 1) => (
                    <Tag bordered={false} color={status === 1 ? 'success' : 'error'}>
                        {status === 1 ? t('common.allow') : t('common.block')}
                    </Tag>
                ),
            },
            {
                title: t('ipFilters.columnRule'),
                key: 'rule',
                ellipsis: true,
                render: (_, record) => <span className="console-rule-text">{ruleDisplayText(t, record)}</span>,
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
                width: 100,
                fixed: 'right',
                render: (_, record) => (
                    <Popconfirm
                        title={t('ipFilters.deleteConfirmTitle')}
                        description={t('ipFilters.deleteConfirmDesc')}
                        onConfirm={() => void handleDelete(record)}
                        okText={t('common.delete')}
                        cancelText={t('common.cancel')}
                        okButtonProps={{danger: true}}
                    >
                        <Button
                            type="link"
                            danger
                            icon={<DeleteOutlined/>}
                            loading={deletingId === record.id}
                            style={{paddingInline: 0, whiteSpace: 'nowrap'}}
                        >
                            {t('common.delete')}
                        </Button>
                    </Popconfirm>
                ),
            },
        ],
        [t, deletingId],
    )

    return (
        <div className="console-page">
            <PageHeader
                description={t('ipFilters.description')}
                badge={
                    <Space size={8}>
                        <span className="console-count-badge">{t('ipFilters.allowCount', {count: allowCount})}</span>
                        <span className="console-count-badge console-count-badge--danger">
              {t('ipFilters.blockCount', {count: blockCount})}
            </span>
                    </Space>
                }
                extra={
                    <Space>
                        <Button icon={<ReloadOutlined/>} onClick={() => void loadRules()}>
                            {t('common.refresh')}
                        </Button>
                        <Button type="primary" icon={<PlusOutlined/>} onClick={() => setModalOpen(true)}>
                            {t('ipFilters.addRule')}
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
                    dataSource={rules}
                    tableLayout="fixed"
                    scroll={{x: 652}}
                    pagination={{pageSize: 10, showSizeChanger: true, showTotal: (total) => t('common.total', {total})}}
                    locale={{
                        emptyText: (
                            <div className="console-empty-hint">
                                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('ipFilters.empty')}/>
                            </div>
                        ),
                    }}
                />
            </PageCard>

            <Modal
                className="console-modal"
                title={t('ipFilters.modalTitle')}
                open={modalOpen}
                onCancel={() => {
                    setModalOpen(false)
                    form.resetFields()
                }}
                onOk={() => form.submit()}
                confirmLoading={submitting}
                destroyOnHidden
                okText={t('common.add')}
                cancelText={t('common.cancel')}
            >
                <Form
                    form={form}
                    layout="vertical"
                    initialValues={{status: 0, field: 'ip'}}
                    onFinish={(values) => void handleAdd(values)}
                >
                    <Form.Item name="status" label={t('ipFilters.actionLabel')} rules={[{required: true}]}>
                        <Select
                            options={[
                                {label: t('ipFilters.allowOption'), value: 1},
                                {label: t('ipFilters.blockOption'), value: 0},
                            ]}
                        />
                    </Form.Item>
                    <Form.Item name="field" label={t('ipFilters.fieldLabel')} rules={[{required: true}]}>
                        <Select options={fieldOptions}/>
                    </Form.Item>
                    <Form.Item noStyle shouldUpdate={(prev, next) => prev.field !== next.field}>
                        {({getFieldValue}) => {
                            const field = getFieldValue('field') as IPFilterField
                            if (categoryFields.has(field)) {
                                return null
                            }
                            return (
                                <Form.Item
                                    name="value"
                                    label={t('ipFilters.valueLabel')}
                                    rules={[{required: true, message: t('ipFilters.valueRequired')}]}
                                >
                                    <Input placeholder={t('ipFilters.valuePlaceholder')}/>
                                </Form.Item>
                            )
                        }}
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    )
}
