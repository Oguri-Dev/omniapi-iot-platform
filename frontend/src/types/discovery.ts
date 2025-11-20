export type DiscoveryAvailability = 'ready' | 'partial' | 'error'

export interface DiscoveryMetric {
  label: string
  value: string
  accent?: 'info' | 'success' | 'warning'
}

export interface DiscoveryTableRow {
  [key: string]: string | number
}

export interface DiscoveryDataset {
  title: string
  summary?: string
  highlights?: string[]
  metrics?: DiscoveryMetric[]
  table?: {
    columns: string[]
    rows: DiscoveryTableRow[]
  }
}

export interface DiscoveryEndpoint {
  label: string
  method: 'GET' | 'POST'
  path: string
  description: string
  lastSync?: string
  availability: DiscoveryAvailability
  dataset?: DiscoveryDataset
}

export interface DiscoveryGroup {
  id: string
  title: string
  description: string
  endpoints: DiscoveryEndpoint[]
}

export interface ScaleAQDiscoveryResult {
  siteId: string
  siteName: string
  tenantCode?: string
  generatedAt: string
  headersUsed: {
    scaleVersion: string
    accept: string
  }
  summary: {
    timeseriesCount: number
    metricsAvailable: number
    range: string
  }
  groups: DiscoveryGroup[]
}
