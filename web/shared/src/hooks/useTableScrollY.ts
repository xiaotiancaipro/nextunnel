import {useEffect, useRef, useState} from 'react'

/** Fallback when table header is not mounted yet. */
const HEADER_FALLBACK = 39
const MIN_BODY = 120

/**
 * Measure the table body container and return Ant Design Table `scroll.y`.
 * Pagination should live outside this container (card footer).
 */
export default function useTableScrollY() {
    const containerRef = useRef<HTMLDivElement>(null)
    const [scrollY, setScrollY] = useState(MIN_BODY)

    useEffect(() => {
        const el = containerRef.current
        if (!el) {
            return
        }

        const update = () => {
            const thead = el.querySelector<HTMLElement>('.ant-table-thead')
            const theadH = thead ? thead.getBoundingClientRect().height : HEADER_FALLBACK
            const next = Math.max(MIN_BODY, Math.floor(el.clientHeight - theadH))
            setScrollY((prev) => (prev === next ? prev : next))
        }

        update()
        const observer = new ResizeObserver(update)
        observer.observe(el)
        return () => observer.disconnect()
    }, [])

    return {containerRef, scrollY}
}
