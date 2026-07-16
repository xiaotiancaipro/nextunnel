import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'

dayjs.extend(utc)

/** Format a UTC ISO timestamp for display in the browser's local timezone. */
export function formatTimestamp(value: string): string {
    const parsed = dayjs.utc(value).local()
    if (!parsed.isValid()) {
        return value
    }
    return parsed.format('YYYY-MM-DD HH:mm:ss')
}
