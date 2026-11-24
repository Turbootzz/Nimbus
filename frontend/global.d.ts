import type messages from './messages/en.json'

// Type-checked message keys: t('a.b') fails to compile when en.json lacks it
declare module 'next-intl' {
  interface AppConfig {
    Messages: typeof messages
  }
}
