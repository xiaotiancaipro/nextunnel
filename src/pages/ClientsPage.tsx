import {useCallback, useEffect, useState} from 'react'
import {Button, Form, Input, InputNumber, message, Modal, Space, Switch, Table, Tag, Typography,} from 'antd'
import {DownloadOutlined, PlusOutlined, ReloadOutlined} from '@ant-design/icons'
import type {ColumnsType} from 'antd/es/table'
import {createClient, downloadClientCerts, formatPortRange, listClients,} from '../api'
import type {Client} from '../types'

interface CreateFormValues {
    name: string
    limitPorts: boolean
    portStart?: number
    portEnd?: number
}

export default function ClientsPage() {
    const [clients, setClients] = useState<Client[]>([])
    const [loading, setLoading] = useState(false)
    const [modalOpen, setModalOpen] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [downloading, setDownloading] = useState<string | null>(null)
    const [form] = Form.useForm<CreateFormValues>()

    const loadClients = useCallback(async () => {
        setLoading(true)
        try {
            setClients(await listClients())
        } catch (err) {
            message.error(err instanceof Error ? err.message : '加载客户端失败')
        } finally {
            setLoading(false)
        }
    }, [])

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
            message.success(`客户端 ${values.name} 创建成功`)
            setModalOpen(false)
            form.resetFields()
            await loadClients()
        } catch (err) {
            message.error(err instanceof Error ? err.message : '创建失败')
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
            message.success(`已下载 ${name} 的证书`)
        } catch (err) {
            message.error(err instanceof Error ? err.message : '证书生成失败')
        } finally {
            setDownloading(null)
        }
    }

    const columns: ColumnsType<Client> = [
        {
            title: 'ID',
            dataIndex: 'id',
            key: 'id',
            ellipsis: true,
        },
        {
            title: '名称',
            dataIndex: 'name',
            key: 'name',
            render: (name: string) => <Typography.Text strong>{name}</Typography.Text>,
        },
        {
            title: '远程端口范围',
            key: 'ports',
            render: (_, record) => (
                <Tag color={record.portStart > 0 ? 'blue' : 'default'}>
                    {formatPortRange(record.portStart, record.portEnd)}
                </Tag>
            ),
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
            width: 160,
            render: (_, record) => (
                <Button
                    type="link"
                    icon={<DownloadOutlined/>}
                    loading={downloading === record.name}
                    onClick={() => void handleDownloadCerts(record.name)}
                >
                    生成证书
                </Button>
            ),
        },
    ]

    return (
        <>
            <Space style={{marginBottom: 16, width: '100%', justifyContent: 'space-between'}}>
                <Typography.Title level={4} style={{margin: 0}}>
                    客户端管理
                </Typography.Title>
                <Space>
                    <Button icon={<ReloadOutlined/>} onClick={() => void loadClients()}>
                        刷新
                    </Button>
                    <Button type="primary" icon={<PlusOutlined/>} onClick={() => setModalOpen(true)}>
                        创建客户端
                    </Button>
                </Space>
            </Space>

            <Table
                rowKey="id"
                loading={loading}
                columns={columns}
                dataSource={clients}
                pagination={{pageSize: 10, showSizeChanger: true}}
            />

            <Modal
                title="创建客户端"
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
                    initialValues={{limitPorts: false}}
                    onFinish={(values) => void handleCreate(values)}
                >
                    <Form.Item
                        name="name"
                        label="客户端名称"
                        rules={[{required: true, message: '请输入客户端名称'}]}
                    >
                        <Input placeholder="例如：office-pc"/>
                    </Form.Item>
                    <Form.Item name="limitPorts" label="限制远程端口范围" valuePropName="checked">
                        <Switch/>
                    </Form.Item>
                    <Form.Item noStyle shouldUpdate={(prev, next) => prev.limitPorts !== next.limitPorts}>
                        {({getFieldValue}) =>
                            getFieldValue('limitPorts') ? (
                                <Space style={{display: 'flex'}} align="start">
                                    <Form.Item
                                        name="portStart"
                                        label="起始端口"
                                        rules={[{required: true, message: '请输入起始端口'}]}
                                    >
                                        <InputNumber min={1} max={65535} style={{width: 140}}/>
                                    </Form.Item>
                                    <Form.Item
                                        name="portEnd"
                                        label="结束端口"
                                        rules={[{required: true, message: '请输入结束端口'}]}
                                    >
                                        <InputNumber min={1} max={65535} style={{width: 140}}/>
                                    </Form.Item>
                                </Space>
                            ) : null
                        }
                    </Form.Item>
                </Form>
            </Modal>
        </>
    )
}
