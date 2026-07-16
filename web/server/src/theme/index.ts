import type { ThemeConfig } from 'antd'

export const BRAND_BLUE = '#0870e9'
export const BRAND_BLUE_RGB = '8, 112, 233'

export const consoleTheme: ThemeConfig = {
  token: {
    colorPrimary: BRAND_BLUE,
    colorBgLayout: '#f8fafc',
    colorBgContainer: '#ffffff',
    colorBorder: '#e5e7eb',
    colorBorderSecondary: '#e5e7eb',
    colorText: 'rgba(15, 23, 42, 0.88)',
    colorTextSecondary: 'rgba(15, 23, 42, 0.45)',
    colorTextHeading: '#171717',
    borderRadius: 8,
    borderRadiusLG: 12,
    borderRadiusSM: 6,
    fontSize: 14,
    controlHeight: 32,
    controlHeightSM: 32,
  },
  components: {
    Button: {
      borderRadius: 8,
      borderRadiusLG: 8,
      borderRadiusSM: 6,
      controlHeight: 32,
      controlHeightSM: 32,
      paddingInline: 14,
      paddingInlineSM: 14,
    },
    Card: {
      borderRadiusLG: 12,
      paddingLG: 18,
    },
    Input: {
      borderRadius: 8,
      controlHeight: 32,
      paddingInline: 11,
    },
    Select: {
      borderRadius: 8,
      controlHeight: 32,
    },
    InputNumber: {
      borderRadius: 8,
      controlHeight: 32,
    },
    Modal: {
      borderRadiusLG: 12,
      paddingContentHorizontal: 24,
      titleFontSize: 16,
      titleColor: '#171717',
    },
    Table: {
      borderRadius: 12,
      borderRadiusLG: 12,
      headerBg: '#fafafa',
      cellPaddingBlockSM: 10,
      cellPaddingInlineSM: 12,
    },
    Menu: {
      itemBorderRadius: 8,
      subMenuItemBorderRadius: 8,
      itemHeight: 36,
      iconSize: 16,
      itemColor: 'rgba(15, 23, 42, 0.88)',
      itemHoverColor: '#171717',
      itemSelectedColor: '#171717',
      itemHoverBg: `rgba(${BRAND_BLUE_RGB}, 0.06)`,
      itemSelectedBg: `rgba(${BRAND_BLUE_RGB}, 0.1)`,
    },
    Layout: {
      headerBg: '#ffffff',
      siderBg: '#ffffff',
      bodyBg: '#f8fafc',
    },
    Form: {
      labelFontSize: 14,
      itemMarginBottom: 16,
    },
    Tag: {
      borderRadiusSM: 6,
    },
    Switch: {
      colorPrimary: BRAND_BLUE,
    },
  },
}
