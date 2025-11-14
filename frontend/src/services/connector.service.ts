import api from './api'

export interface ExternalService {
  id: string
  name: string
  type: string // mqtt, rest, websocket, etc.
  url: string
  auth_type?: string // none, basic, token, oauth
  username?: string
  password?: string
  token?: string
  status: string // active, inactive, error
  created_at: string
  updated_at: string
  metadata?: Record<string, any>
}

export interface CreateServiceDTO {
  name: string
  type: string
  url: string
  auth_type?: string
  username?: string
  password?: string
  token?: string
  metadata?: Record<string, any>
}

class ConnectorService {
  async getAll(): Promise<ExternalService[]> {
    const response = await api.get<{ success: boolean; data: ExternalService[] }>('/api/services')
    return response.data.data
  }

  async getById(id: string): Promise<ExternalService> {
    const response = await api.get<{ success: boolean; data: ExternalService }>(
      `/api/services/${id}`
    )
    return response.data.data
  }

  async create(service: CreateServiceDTO): Promise<ExternalService> {
    const response = await api.post<{ success: boolean; data: ExternalService }>(
      '/api/services',
      service
    )
    return response.data.data
  }

  async update(id: string, service: Partial<CreateServiceDTO>): Promise<ExternalService> {
    const response = await api.put<{ success: boolean; data: ExternalService }>(
      `/api/services/${id}`,
      service
    )
    return response.data.data
  }

  async delete(id: string): Promise<void> {
    await api.delete(`/api/services/${id}`)
  }

  async testConnection(id: string): Promise<{ success: boolean; message: string }> {
    const response = await api.post<{ success: boolean; message: string }>(
      `/api/services/${id}/test`
    )
    return response.data
  }
}

export default new ConnectorService()
