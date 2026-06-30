import {useCallback, useEffect, useState} from 'react'
import {Button, Form, Input, message, Modal, Popconfirm, Select, Space, Table, Tag, Typography,} from 'antd'
import {DeleteOutlined, PlusOutlined, ReloadOutlined} from '@ant-design/icons'
import type {ColumnsType} from 'antd/es/table'
import {addIPFilter, deleteIPFilter, fromRuleToMutate, listIPFilters, ruleDisplayText, toMutatePayload,} from '../api'
import type {IPFilterField, IPFilterRule} from '../types'

interface AddFormValues {
    status: 0 | 1
    field: IPFilterField
    value?: string
}

const fieldOptions = [
    {label: 'IP 地址', value: 'ip'},
    {label: '国家', value: 'country'},
    {label: '地区', value: 'region'},
    {label: '城市', value: 'city'},
    {label: '全部流量', value: 'all'},
    {label: '本地网络', value: 'local'},
    {label: '远程网络', value: 'remote'},
]

const categoryFields = new Set<IPFilterField>(['all', 'local', 'remote'])

export default function IpFilterPage() {
    const [rules, setRules] = useState<IPFilterRule[]>([])
    const [loading, setLoading] = useState(false)
    const [modalOpen, setModalOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [deletingId, setDeletingId] = useState<string | null>(null)
    const [form] = Form.useForm<AddFormValues>()

    const loadRules = useCallback(async () => {
        setLoading(true)
        try {
            setRules(await listIPFilters())
        } catch (err) {
            message.error(err instanceof Error ? err.message : '加载规则失败')
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        void loadRules()
    }, [loadRules])

    const handleAdd = async (values: AddFormValues) => {
        setSubmitting(true)
        try {
            await addIPFilter(toMutatePayload(values.status, values.field, values.value?.trim()))
            message.success('规则已添加')
            setModalOpen(false)
            form.resetFields()
            await loadRules()
        } catch (err) {
            message.error(err instanceof Error ? err.message : '添加失败')
        } finally {
            setSubmitting(false)
        }
    }

    const handleDelete = async (rule: IPFilterRule) => {
        setDeletingId(rule.id)
        try {
            await deleteIPFilter(fromRuleToMutate(rule))
            message.success('规则已删除')
            await loadRules()
        } catch (err) {
            message.error(err instanceof Error ? err.message : '删除失败')
        } finally {
            setDeletingId(null)
        }
    }

    const columns: ColumnsType<IPFilterRule> = [
        {
            title: '动作',
            dataIndex: 'status',
            key: 'status',
            width: 100,
            render: (status: 0 | 1) => (
                <Tag color={status === 1 ? 'success' : 'error'}>{status === 1 ? '允许' : '拒绝'}</Tag>
            ),
        },
        {
            title: '匹配规则',
            key: 'rule',
            render: (_, record) => ruleDisplayText(record),
        },
        {
            title: '创建时间',
            dataIndex: 'createdAt',
            key: 'createdAt',
            width: 200,
        },
        {
            title: '操作',
            key: 'actions',
            width: 100,
            render: (_, record) => (
                <Popconfirm
                    title="确认删除此规则？"
                    onConfirm={() => void handleDelete(record)}
                    okText="删除"
                    cancelText="取消"
                >
                    <Button
                        type="link"
                        danger
                        icon={<DeleteOutlined/>}
                        loading={deletingId === record.id}
                    >
                        删除
                    </Button>
                </Popconfirm>
            ),
        },
    ]

    return (
        <>
            <Space style={{marginBottom: 16, width: '100%', justifyContent: 'space-between'}}>
                <Typography.Title level={4} style={{margin: 0}}>
                    访问控制规则
                </Typography.Title>
                <Space>
                    <Button icon={<ReloadOutlined/>} onClick={() => void loadRules()}>
                        刷新
                    </Button>
                    <Button type="primary" icon={<PlusOutlined/>} onClick={() => setModalOpen(true)}>
                        添加规则
                    </Button>
                </Space>
            </Space>

            <Table
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={rules}
                locale={{emptyText: '暂无访问控制规则'}}
                pagination={{pageSize: 10, showSizeChanger: true}}
            />

            <Modal
                title="添加访问控制规则"
                open={modalOpen}
                onCancel={() => {
                    setModalOpen(false)
                    form.resetFields()
                }}
                onOk={() => form.submit()}
                confirmLoading={submitting}
                destroyOnHidden
            >
                <Form
                    form={form}
                    layout="vertical"
                    initialValues={{status: 0, field: 'ip'}}
                    onFinish={(values) => void handleAdd(values)}
                >
                    <Form.Item name="status" label="动作" rules={[{required: true}]}>
                        <Select
                            options={[
                                {label: '允许 (allow)', value: 1},
                                {label: '拒绝 (block)', value: 0},
                            ]}
                        />
                    </Form.Item>
                    <Form.Item name="field" label="匹配维度" rules={[{required: true}]}>
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
                                    label="匹配值"
                                    rules={[{required: true, message: '请输入匹配值'}]}
                                >
                                    <Input placeholder="例如：203.0.113.10 或 Shenzhen"/>
                                </Form.Item>
                            )
                        }}
                    </Form.Item>
                </Form>
            </Modal>
        </>
    )
}
