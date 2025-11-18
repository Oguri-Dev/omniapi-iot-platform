import api from './api'

export interface TenantContact {
  email?: string
  phone?: string
  contactName?: string
}

export interface TenantAddress {
  country?: string
  region?: string
  city?: string
  street?: string
  zipCode?: string
}

export interface Tenant {
  id?: string
  code: string
  name: string
  type?: string
  contact?: TenantContact
  address?: TenantAddress
  status?: string
  metadata?: Record<string, any>
  createdAt?: string
  updatedAt?: string
  createdBy?: string
}

export interface CreateTenantDTO {
  code: string
  name: string
  type?: string
  contact?: TenantContact
  address?: TenantAddress
  metadata?: Record<string, any>
}

export interface UpdateTenantDTO {
  name?: string
  type?: string
  contact?: TenantContact
  address?: TenantAddress
  status?: string
  metadata?: Record<string, any>
}

const tenantService = {
  // Obtener todos los tenants
  getAll: async (params?: { status?: string; search?: string }) => {
    const queryParams = new URLSearchParams()
    if (params?.status) queryParams.append('status', params.status)
    if (params?.search) queryParams.append('search', params.search)

    const url = `/api/tenants${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
    const response = await api.get(url)
    return response.data
  },

  // Obtener un tenant por ID
  getById: async (id: string) => {
    const response = await api.get(`/api/tenants/get?id=${id}`)
    return response.data
  },

  // Crear un nuevo tenant
  create: async (tenant: CreateTenantDTO) => {
    const response = await api.post('/api/tenants/create', tenant)
    return response.data
  },

  // Actualizar un tenant
  update: async (id: string, tenant: UpdateTenantDTO) => {
    const response = await api.put(`/api/tenants/update?id=${id}`, tenant)
    return response.data
  },

  // Eliminar un tenant
  delete: async (id: string) => {
    const response = await api.delete(`/api/tenants/delete?id=${id}`)
    return response.data
  },
}

export default tenantService
