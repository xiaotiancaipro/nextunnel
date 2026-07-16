import { GlobalOutlined } from '@ant-design/icons'
import { Select } from 'antd'
import { useI18n, type Locale } from '../i18n'

export default function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n()

  return (
    <Select
      size="small"
      value={locale}
      onChange={(value: Locale) => setLocale(value)}
      options={[
        { value: 'zh', label: t('language.zh') },
        { value: 'en', label: t('language.en') },
      ]}
      suffixIcon={<GlobalOutlined />}
      aria-label={t('language.label')}
      style={{ width: 108 }}
      popupMatchSelectWidth={false}
    />
  )
}
