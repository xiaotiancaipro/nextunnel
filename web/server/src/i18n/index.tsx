import {I18nProvider as SharedI18nProvider, type Locale, type TFunction, useI18n} from '@nextunnel/web-shared'
import type {ReactNode} from 'react'
import en from './locales/en'
import zh from './locales/zh'

const locales = {
    zh: zh as unknown as Record<string, unknown>,
    en: en as unknown as Record<string, unknown>,
}

const documentTitles: Record<Locale, string> = {
    zh: 'nextunnel-server 管理控制台',
    en: 'nextunnel-server Console',
}

export function I18nProvider({children}: { children: ReactNode }) {
    return (
        <SharedI18nProvider locales={locales} documentTitles={documentTitles}>
            {children}
        </SharedI18nProvider>
    )
}

export {useI18n, type Locale, type TFunction}
export {formatPortRange, ruleDisplayText} from './display'
