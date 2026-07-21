import type { ReactNode } from 'react'
import { Card } from 'antd'
import type { CardProps } from 'antd'

interface PageCardProps extends CardProps {
  children: ReactNode
}

export default function PageCard({ children, className, ...props }: PageCardProps) {
  return (
    <Card size="small" {...props} className={['console-card', className].filter(Boolean).join(' ')}>
      {children}
    </Card>
  )
}
