import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { App as AntdApp, ConfigProvider } from 'antd'
import enUS from 'antd/locale/en_US'
import zhCN from 'antd/locale/zh_CN'
import { consoleTheme } from '../theme'

export type Locale = 'zh' | 'en'

const STORAGE_KEY = 'nextunnel-locale'

const antdLocales = { zh: zhCN, en: enUS }

function detectLocale(): Locale {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'zh' || stored === 'en') {
    return stored
  }
  const browserLang = navigator.language.toLowerCase()
  return browserLang.startsWith('zh') ? 'zh' : 'en'
}

type Params = Record<string, string | number>

function lookup(obj: Record<string, unknown>, path: string): string | undefined {
  const value = path.split('.').reduce<unknown>((current, key) => {
    if (current && typeof current === 'object') {
      return (current as Record<string, unknown>)[key]
    }
    return undefined
  }, obj)
  return typeof value === 'string' ? value : undefined
}

function interpolate(template: string, params?: Params): string {
  if (!params) return template
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) => String(params[key] ?? ''))
}

export type TFunction = (key: string, params?: Params) => string

interface I18nContextValue {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: TFunction
}

const I18nContext = createContext<I18nContextValue | null>(null)

export interface I18nProviderProps {
  children: ReactNode
  locales: Record<Locale, Record<string, unknown>>
  documentTitles?: Partial<Record<Locale, string>>
}

export function I18nProvider({ children, locales, documentTitles }: I18nProviderProps) {
  const [locale, setLocaleState] = useState<Locale>(detectLocale)

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next)
    localStorage.setItem(STORAGE_KEY, next)
  }, [])

  const t = useCallback<TFunction>(
    (key, params) => {
      const text = lookup(locales[locale], key) ?? key
      return interpolate(text, params)
    },
    [locale, locales],
  )

  useEffect(() => {
    document.documentElement.lang = locale === 'zh' ? 'zh-CN' : 'en'
    const title = documentTitles?.[locale]
    if (title) {
      document.title = title
    }
  }, [documentTitles, locale])

  const value = useMemo(() => ({ locale, setLocale, t }), [locale, setLocale, t])

  return (
    <I18nContext.Provider value={value}>
      <ConfigProvider locale={antdLocales[locale]} theme={consoleTheme}>
        <AntdApp>{children}</AntdApp>
      </ConfigProvider>
    </I18nContext.Provider>
  )
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) {
    throw new Error('useI18n must be used within I18nProvider')
  }
  return ctx
}
