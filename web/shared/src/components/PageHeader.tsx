import type { ReactNode } from 'react'
import { Flex, Typography } from 'antd'

interface PageHeaderProps {
  description?: string
  extra?: ReactNode
  badge?: ReactNode
}

export default function PageHeader({ description, extra, badge }: PageHeaderProps) {
  return (
    <Flex
      justify="space-between"
      align="flex-start"
      wrap="wrap"
      gap={12}
      className="console-page-toolbar"
    >
      <Flex vertical gap={8} flex={1} style={{ minWidth: 0 }}>
        {description ? (
          <Typography.Paragraph className="console-page-desc">{description}</Typography.Paragraph>
        ) : null}
        {badge ? <div className="console-page-badges">{badge}</div> : null}
      </Flex>
      {extra ? <div>{extra}</div> : null}
    </Flex>
  )
}
