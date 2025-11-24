import type { DiscoveryProvider } from '../services/discovery.service'

export type BuilderViewMode = 'discovery' | 'builder'

export interface EndpointParamDescriptor {
  name: string
  label: string
  required?: boolean
  placeholder?: string
  helperText?: string
}

export interface BuilderEndpointMeta {
  id: string
  provider: DiscoveryProvider
  label: string
  method: string
  path: string
  description: string
  category: string
  targetBlock: 'snapshots' | 'timeseries' | 'kpis' | 'assets' | 'alerts'
  sampleResponseHint?: string
  defaultParams?: Record<string, string>
  params?: EndpointParamDescriptor[]
}

export interface BuilderEndpointSelection {
  endpointId: string
  label: string
  method: string
  path: string
  targetBlock: BuilderEndpointMeta['targetBlock']
  params: Record<string, string>
}

export interface BuilderPayloadPreview {
  provider: DiscoveryProvider
  site: {
    id: string
    name?: string
    code?: string
    tenantCode?: string
  }
  generatedAt: string
  endpoints: BuilderEndpointSelection[]
}
