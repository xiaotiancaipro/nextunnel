import {memo, useCallback, useEffect, useMemo, useState} from 'react'
import {Button, Drawer, Empty, Flex, Form, Input, message, Popconfirm, Select, Space, Table, Tag} from 'antd'
import {
    AimOutlined,
    CheckCircleOutlined,
    DeleteOutlined,
    EnvironmentOutlined,
    FilterOutlined,
    GlobalOutlined,
    PlusOutlined,
    ReloadOutlined,
    StopOutlined,
} from '@ant-design/icons'
import type {ColumnsType, TablePaginationConfig} from 'antd/es/table'
import {
    Background,
    BackgroundVariant,
    Controls,
    type Edge,
    Handle,
    MarkerType,
    type Node,
    type NodeProps,
    Position,
    ReactFlow,
    ReactFlowProvider,
    useEdgesState,
    useNodesState,
    useReactFlow,
} from '@xyflow/react'
import {formatTimestamp, PageCard, PageHeader, type TFunction} from '@nextunnel/web-shared'
import {addIPFilter, deleteIPFilter, fromRuleToMutate, listIPFilters, toMutatePayload} from '../api'
import {ruleDisplayText, useI18n} from '../i18n'
import type {IPFilterField, IPFilterRule} from '../types'

const categoryFields = new Set<IPFilterField>(['all', 'local', 'remote'])
const DRAWER_WIDTH = 480
const CARD_GAP = 64
const NODE_WIDTH = 200
const NODE_HEIGHT = 188
const ENTRY_WIDTH = 180
const ENTRY_HEIGHT = 132
const ORIGIN_Y = 24
const PRIORITY_ORDER = ['ip', 'city', 'region', 'country', 'network', 'all'] as const

const EntryFlowNodeMemo = memo(EntryFlowNode)
const RuleFlowNodeMemo = memo(RuleFlowNode)

const nodeTypes = {
    entry: EntryFlowNodeMemo,
    rule: RuleFlowNodeMemo,
}

type PriorityKey = (typeof PRIORITY_ORDER)[number]

type EntryNodeData = { label: string; hint: string }

type RuleNodeData = {
    priorityLabel: string
    text: string
    status: 0 | 1
    key: PriorityKey
    allowLabel: string
    blockLabel: string
}

interface AddFormValues {
    status: 0 | 1
    field: IPFilterField
    value?: string
}

function rulePriorityKey(rule: IPFilterRule): PriorityKey {
    if (rule.field === 'ip') return 'ip'
    if (rule.field === 'city') return 'city'
    if (rule.field === 'region') return 'region'
    if (rule.field === 'country') return 'country'
    if (rule.field === 'category') {
        if (rule.value === 'ALL') return 'all'
        return 'network'
    }
    return 'all'
}

function priorityIcon(key: PriorityKey) {
    switch (key) {
        case 'ip':
            return <AimOutlined/>
        case 'city':
        case 'region':
        case 'country':
            return <EnvironmentOutlined/>
        case 'network':
            return <GlobalOutlined/>
        default:
            return <FilterOutlined/>
    }
}

function sortSamePriority(a: IPFilterRule, b: IPFilterRule): number {
    // Same specificity: allow (1) before block (0)
    if (a.status !== b.status) {
        return b.status - a.status
    }
    return a.id.localeCompare(b.id)
}

function EntryFlowNode({data}: NodeProps<Node<EntryNodeData, 'entry'>>) {
    return (
        <div className="flow-node flow-node--entry">
            <Handle type="source" position={Position.Bottom} className="flow-node__handle"/>
            <div className="flow-node__icon flow-node__icon--blue">
                <GlobalOutlined/>
            </div>
            <div className="flow-node__title">{data.label}</div>
            <div className="flow-node__hint">{data.hint}</div>
        </div>
    )
}

function RuleFlowNode({data}: NodeProps<Node<RuleNodeData, 'rule'>>) {
    const isAllow = data.status === 1
    return (
        <div className={`flow-node flow-node--rule${isAllow ? '' : ' flow-node--deny'}`}>
            <Handle type="target" position={Position.Top} className="flow-node__handle"/>
            <div className={`flow-node__icon${isAllow ? ' flow-node__icon--teal' : ' flow-node__icon--rose'}`}>
                {priorityIcon(data.key)}
            </div>
            <div className="flow-node__title">{data.priorityLabel}</div>
            <div className={`flow-node__action${isAllow ? ' flow-node__action--allow' : ' flow-node__action--block'}`}>
                {isAllow ? (
                    <CheckCircleOutlined/>
                ) : (
                    <StopOutlined/>
                )}
                <span>{isAllow ? data.allowLabel : data.blockLabel}</span>
            </div>
            <div className="flow-node__rule-text">{data.text}</div>
            <Handle type="source" position={Position.Bottom} className="flow-node__handle"/>
        </div>
    )
}

function layerCenterX(count: number): number {
    const width = count * NODE_WIDTH + Math.max(count - 1, 0) * CARD_GAP
    return width / 2
}

function buildFlowGraph(
    layers: { key: PriorityKey; label: string; items: IPFilterRule[] }[],
    t: TFunction,
): { nodes: Node[]; edges: Edge[] } {
    if (layers.length === 0) {
        return {nodes: [], edges: []}
    }

    const maxLayerWidth = Math.max(...layers.map((layer) => layerCenterX(layer.items.length) * 2), ENTRY_WIDTH)
    const canvasCenterX = 40 + maxLayerWidth / 2

    const nodes: Node[] = [
        {
            id: 'entry',
            type: 'entry',
            position: {x: canvasCenterX - ENTRY_WIDTH / 2, y: ORIGIN_Y},
            data: {
                label: t('accessControl.flow.stepRequest'),
                hint: t('accessControl.flow.stepRequestHint'),
            },
            draggable: false,
            selectable: false,
            style: {width: ENTRY_WIDTH, height: ENTRY_HEIGHT},
        },
    ]

    const edges: Edge[] = []
    const edgeStyle = {stroke: '#94a3b8', strokeWidth: 1.5, strokeDasharray: '6 4'}
    const marker = {type: MarkerType.ArrowClosed, width: 14, height: 14, color: '#94a3b8'}
    const layerNodeIds: string[][] = []

    layers.forEach((layer, layerIndex) => {
        const count = layer.items.length
        const layerWidth = count * NODE_WIDTH + Math.max(count - 1, 0) * CARD_GAP
        const startX = canvasCenterX - layerWidth / 2
        const y = ORIGIN_Y + ENTRY_HEIGHT + CARD_GAP + layerIndex * (NODE_HEIGHT + CARD_GAP)
        const ids: string[] = []

        layer.items.forEach((rule, ruleIndex) => {
            const id = `rule-${rule.id}`
            ids.push(id)
            nodes.push({
                id,
                type: 'rule',
                position: {x: startX + ruleIndex * (NODE_WIDTH + CARD_GAP), y},
                data: {
                    priorityLabel: layer.label,
                    text: ruleDisplayText(t, rule),
                    status: rule.status,
                    key: layer.key,
                    allowLabel: t('common.allow'),
                    blockLabel: t('common.block'),
                },
                draggable: false,
                selectable: false,
                style: {width: NODE_WIDTH, height: NODE_HEIGHT},
            })
        })

        layerNodeIds.push(ids)
    })

    // Entry → first priority layer
    for (const targetId of layerNodeIds[0]) {
        edges.push({
            id: `entry-${targetId}`,
            source: 'entry',
            target: targetId,
            type: 'default',
            style: edgeStyle,
            markerEnd: marker,
        })
    }

    // Priority chain: IP → city → region → country → network → all
    for (let i = 0; i < layerNodeIds.length - 1; i++) {
        const fromIds = layerNodeIds[i]
        const toIds = layerNodeIds[i + 1]
        for (const sourceId of fromIds) {
            for (const targetId of toIds) {
                edges.push({
                    id: `${sourceId}-${targetId}`,
                    source: sourceId,
                    target: targetId,
                    type: 'default',
                    style: edgeStyle,
                    markerEnd: marker,
                })
            }
        }
    }

    return {nodes, edges}
}

function FilterFlowCanvas({rules}: { rules: IPFilterRule[] }) {
    const {t} = useI18n()
    const {fitView} = useReactFlow()

    const activeLayers = useMemo(() => {
        const buckets: Record<PriorityKey, IPFilterRule[]> = {
            ip: [],
            city: [],
            region: [],
            country: [],
            network: [],
            all: [],
        }
        for (const rule of rules) {
            buckets[rulePriorityKey(rule)].push(rule)
        }

        const labels: Record<PriorityKey, string> = {
            ip: t('accessControl.flow.priorityIp'),
            city: t('accessControl.flow.priorityCity'),
            region: t('accessControl.flow.priorityRegion'),
            country: t('accessControl.flow.priorityCountry'),
            network: t('accessControl.flow.priorityNetwork'),
            all: t('accessControl.flow.priorityAll'),
        }

        return PRIORITY_ORDER
            .filter((key) => buckets[key].length > 0)
            .map((key) => ({
                key,
                label: labels[key],
                items: [...buckets[key]].sort(sortSamePriority),
            }))
    }, [rules, t])

    const graph = useMemo(() => buildFlowGraph(activeLayers, t), [activeLayers, t])
    const [nodes, setNodes, onNodesChange] = useNodesState(graph.nodes)
    const [edges, setEdges, onEdgesChange] = useEdgesState(graph.edges)

    useEffect(() => {
        setNodes(graph.nodes)
        setEdges(graph.edges)
        const timer = window.setTimeout(() => {
            void fitView({padding: 0.18, duration: 200})
        }, 40)
        return () => window.clearTimeout(timer)
    }, [graph, setNodes, setEdges, fitView])

    if (activeLayers.length === 0) {
        return (
            <div className="access-flow-canvas access-flow-canvas--empty">
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('accessControl.empty')}/>
            </div>
        )
    }

    return (
        <div className="access-flow-canvas">
            <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                nodeTypes={nodeTypes}
                fitView
                fitViewOptions={{padding: 0.18}}
                minZoom={0.45}
                maxZoom={1.4}
                nodesDraggable={false}
                nodesConnectable={false}
                elementsSelectable={false}
                panOnScroll
                zoomOnScroll
                proOptions={{hideAttribution: true}}
            >
                <Background
                    id="access-flow-dots"
                    variant={BackgroundVariant.Dots}
                    gap={18}
                    size={1.4}
                    color="#c5cdd8"
                />
                <Controls showInteractive={false} position="bottom-right"/>
            </ReactFlow>
        </div>
    )
}

function FilterFlowChart({rules}: { rules: IPFilterRule[] }) {
    return (
        <ReactFlowProvider>
            <FilterFlowCanvas rules={rules}/>
        </ReactFlowProvider>
    )
}

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
            {label: t('accessControl.field.ip'), value: 'ip' as const},
            {label: t('accessControl.field.country'), value: 'country' as const},
            {label: t('accessControl.field.region'), value: 'region' as const},
            {label: t('accessControl.field.city'), value: 'city' as const},
            {label: t('accessControl.field.all'), value: 'all' as const},
            {label: t('accessControl.field.local'), value: 'local' as const},
            {label: t('accessControl.field.remote'), value: 'remote' as const},
        ],
        [t],
    )

    const loadRules = useCallback(async () => {
        setLoading(true)
        try {
            setRules(await listIPFilters())
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('accessControl.loadFailed'))
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
            message.success(t('accessControl.addSuccess'))
            closeDrawer()
            await loadRules()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('accessControl.addFailed'))
        } finally {
            setSubmitting(false)
        }
    }

    const handleDelete = async (rule: IPFilterRule) => {
        setDeletingId(rule.id)
        try {
            await deleteIPFilter(fromRuleToMutate(rule))
            message.success(t('accessControl.deleteSuccess'))
            await loadRules()
        } catch (err) {
            message.error(err instanceof Error ? err.message : t('accessControl.deleteFailed'))
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
                width: 30,
                render: (_value, _record, index) => (page - 1) * pageSize + index + 1,
            },
            {
                title: t('accessControl.columnAction'),
                dataIndex: 'status',
                key: 'status',
                width: 50,
                render: (status: 0 | 1) => (
                    <Tag bordered={false} color={status === 1 ? 'success' : 'error'}>
                        {status === 1 ? t('common.allow') : t('common.block')}
                    </Tag>
                ),
            },
            {
                title: t('accessControl.columnRule'),
                key: 'rule',
                width: 150,
                ellipsis: true,
                render: (_, record) => <span className="console-rule-text">{ruleDisplayText(t, record)}</span>,
            },
            {
                title: t('common.createdAt'),
                dataIndex: 'createdAt',
                key: 'createdAt',
                width: 150,
                render: (value: string) => (
                    <span className="console-id-text">{formatTimestamp(value)}</span>
                ),
            },
            {
                title: t('common.actions'),
                key: 'actions',
                width: 150,
                fixed: 'right',
                render: (_, record) => (
                    <Popconfirm
                        title={t('accessControl.deleteConfirmTitle')}
                        description={t('accessControl.deleteConfirmDesc')}
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
        <div className="console-page access-control-page">
            <PageHeader description={t('accessControl.description')}/>

            <Flex className="console-page-actions" justify="flex-start">
                <Space wrap>
                    <Button type="primary" icon={<PlusOutlined/>} onClick={() => setDrawerOpen(true)}>
                        {t('accessControl.addRule')}
                    </Button>
                    <Button icon={<ReloadOutlined/>} onClick={() => void loadRules()}>
                        {t('common.refresh')}
                    </Button>
                </Space>
            </Flex>

            <div className="access-control-layout">
                <PageCard className="access-control-layout__table">
                    <Table
                        rowKey="id"
                        size="small"
                        loading={loading}
                        columns={columns}
                        dataSource={rules}
                        tableLayout="fixed"
                        scroll={{x: 560}}
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
                                    <Empty
                                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                                        description={t('accessControl.empty')}
                                    />
                                </div>
                            ),
                        }}
                    />
                </PageCard>

                <PageCard className="access-control-layout__flow" title={t('accessControl.flow.title')}>
                    <FilterFlowChart rules={rules}/>
                </PageCard>
            </div>

            <Drawer
                className="console-drawer"
                title={t('accessControl.modalTitle')}
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
                    <Form.Item name="status" label={t('accessControl.actionLabel')} rules={[{required: true}]}>
                        <Select
                            options={[
                                {label: t('accessControl.allowOption'), value: 1},
                                {label: t('accessControl.blockOption'), value: 0},
                            ]}
                        />
                    </Form.Item>
                    <Form.Item name="field" label={t('accessControl.fieldLabel')} rules={[{required: true}]}>
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
                                    label={t('accessControl.valueLabel')}
                                    rules={[{required: true, message: t('accessControl.valueRequired')}]}
                                >
                                    <Input placeholder={t('accessControl.valuePlaceholder')}/>
                                </Form.Item>
                            )
                        }}
                    </Form.Item>
                </Form>
            </Drawer>
        </div>
    )
}
