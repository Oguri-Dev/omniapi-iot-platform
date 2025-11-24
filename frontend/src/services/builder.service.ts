import api from './api'
import type { DiscoveryProvider } from './discovery.service'
import type { BuilderEndpointMeta } from '../types/builder'

export interface BuilderSitePayload {
  id?: string
  code?: string
  name?: string
  tenant_id?: string
  tenant_code?: string
}

export interface BuilderEndpointPayload {
  endpoint_id: string
  label: string
  method: string
  path: string
  target_block: BuilderEndpointMeta['targetBlock']
  params?: Record<string, string>
}

export interface SaveBuilderRunPayload {
  provider: DiscoveryProvider
  site: BuilderSitePayload
  endpoints: BuilderEndpointPayload[]
  discovery_summary?: Record<string, any>
  notes?: string
}

export interface BuilderRunResponse {
  id: string
  provider: DiscoveryProvider
  site: BuilderSitePayload
  endpoints: BuilderEndpointPayload[]
  discovery_summary?: Record<string, any>
  notes?: string
  created_at: string
  updated_at: string
  created_by?: string
}

const builderService = {
  saveRun: async (payload: SaveBuilderRunPayload) => {
    const response = await api.post('/api/discovery/runs', payload)
    return response.data
  },

  listRuns: async (params?: { site_id?: string; provider?: DiscoveryProvider; limit?: number }) => {
    const response = await api.get('/api/discovery/runs', { params })
    return response.data
  },
}

export default builderService
