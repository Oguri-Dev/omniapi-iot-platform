import api from './api'

export interface ExternalService {
  _id: string
  site_id: string
  site_code: string
  tenant_id: string
  tenant_code: string
  code: string
  name: string
  service_type: 'scaleaq' | 'innovex' | 'apikey' | 'custom'
  base_url: string
  credentials?: ServiceCredentials
  config?: Record<string, any>
  status: 'active' | 'inactive' | 'error'
  last_auth?: string
  last_error?: string
  metadata?: Record<string, any>
  created_at: string
  updated_at: string
  created_by?: string
}

export interface ServiceCredentials {
  username?: string
  password?: string
  client_id?: string
  client_secret?: string
  api_key?: string
  custom_headers?: Record<string, string>
}

export interface CreateExternalServiceDTO {
  site_id: string
  name: string
  service_type: 'scaleaq' | 'innovex' | 'apikey' | 'custom'
  base_url: string
  credentials?: ServiceCredentials
  config?: Record<string, any>
  code?: string
  status?: string
  metadata?: Record<string, any>
}

export interface UpdateExternalServiceDTO {
  name?: string
  base_url?: string
  credentials?: ServiceCredentials
  config?: Record<string, any>
  status?: string
  metadata?: Record<string, any>
}

export interface TestConnectionResponse {
  success: boolean
  message: string
  data?: {
    token_type: string
    expires_in: number
    expires_at: string
    token_length: number
  }
  error?: string
}

const externalServiceService = {
  getAll: async (params?: {
    site_id?: string
    tenant_id?: string
    service_type?: string
    status?: string
    search?: string
  }) => {
    const response = await api.get('/external-services', { params })
    return response.data
  },

  getById: async (id: string) => {
    const response = await api.get(`/external-services/get?id=${id}`)
    return response.data
  },

  getBySite: async (siteId: string) => {
    const response = await api.get('/external-services', {
      params: { site_id: siteId },
    })
    return response.data
  },

  create: async (data: CreateExternalServiceDTO) => {
    const response = await api.post('/external-services/create', data)
    return response.data
  },

  update: async (id: string, data: UpdateExternalServiceDTO) => {
    const response = await api.post(`/external-services/update?id=${id}`, data)
    return response.data
  },

  delete: async (id: string) => {
    const response = await api.delete(`/external-services/delete?id=${id}`)
    return response.data
  },

  testConnection: async (id: string): Promise<TestConnectionResponse> => {
    const response = await api.post(`/external-services/test?id=${id}`)
    return response.data
  },
}

export default externalServiceService
