import {useCallback, useEffect, useMemo, useState} from 'react'
import {Button, Drawer, Empty, Flex, Form, Input, message, Popconfirm, Select, Space, Table, Tag} from 'antd'
import {DeleteOutlined, PlusOutlined, ReloadOutlined} from '@ant-design/icons'
import type {ColumnsType, TablePaginationConfig} from 'antd/es/table'
import {formatTimestamp, PageCard, PageHeader} from '@nextunnel/web-shared'
import {addIPFilter, deleteIPFilter, fromRuleToMutate, listIPFilters, toMutatePayload} from '../api'
import {ruleDisplayText, useI18n} from '../i18n'
import type {IPFilterField, IPFilterRule} from '../types'

interface AddFormValues {
    status: 0 | 1
    field: IPFilterField
    value?: string
}

const categoryFields = new Set<IPFilterField>(['all', 'local', 'remote'])
const DRAWER_WIDTH = 480

export default function AccessControl() {
    const {t} = useI18n()
    const [rules, setRules] = useState<IPFilterRule[]>([])
    const [loading, setLoading] = useState(false)
    const [drawerOpen, setDrawerOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [deletingId, setDeletingId] = useState<string | null>(null)
    const [page, setPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)
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

    const closeDrawer = () => {
        setDrawerOpen(false)
        form.resetFields()
    }

    const handleAdd = async (values: AddFormValues) => {
        setSubmitting(true)
        try {
            await addIPFilter(toMutatePayload(values.status, values.field, values.value?.trim()))
            message.success(t('ipFilters.addSuccess'))
            closeDrawer()
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

    const handleTableChange = (pagination: TablePaginationConfig) => {
        setPage(pagination.current ?? 1)
        setPageSize(pagination.pageSize ?? 10)
    }

    const columns: ColumnsType<IPFilterRule> = useMemo(
        () => [
            {
                title: t('common.index'),
                key: 'index',
                width: 64,
                render: (_value, _record, index) => (page - 1) * pageSize + index + 1,
            },
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
                            size="small"
                            danger
                            icon={<DeleteOutlined/>}
                            loading={deletingId === record.id}
                        >
                            {t('common.delete')}
                        </Button>
                    </Popconfirm>
                ),
            },
        ],
        [t, deletingId, page, pageSize],
    )

    return (
        <div className="console-page">
            <PageHeader description={t('ipFilters.description')}/>

            <Flex className="console-page-actions" justify="flex-start">
                <Space wrap>
                    <Button type="primary" icon={<PlusOutlined/>} onClick={() => setDrawerOpen(true)}>
                        {t('ipFilters.addRule')}
                    </Button>
                    <Button icon={<ReloadOutlined/>} onClick={() => void loadRules()}>
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
                    dataSource={rules}
                    tableLayout="fixed"
                    scroll={{x: 716}}
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
                                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('ipFilters.empty')}/>
                            </div>
                        ),
                    }}
                />
            </PageCard>

            <Drawer
                className="console-drawer"
                title={t('ipFilters.modalTitle')}
                open={drawerOpen}
                onClose={closeDrawer}
                width={DRAWER_WIDTH}
                destroyOnHidden
                footer={
                    <Flex justify="flex-end" gap={8}>
                        <Button onClick={closeDrawer}>{t('common.cancel')}</Button>
                        <Button type="primary" loading={submitting} onClick={() => form.submit()}>
                            {t('common.add')}
                        </Button>
                    </Flex>
                }
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
            </Drawer>
        </div>
    )
}
