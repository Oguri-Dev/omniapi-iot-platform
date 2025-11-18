import api from './api'

// Interfaces
export interface SiteLocation {
  latitude: number
  longitude: number
  region?: string
  commune?: string
  water_body?: string
}

export interface Site {
  id?: string
  tenant_id: string
  tenant_code: string
  code: string
  name: string
  location?: SiteLocation
  fecha_apertura: string // ISO date string
  numero_jaulas: number
  cepa: string
  tipo_alimentacion: string
  biomasa_promedio: number
  cantidad_inicial_peces: number
  cantidad_actual_peces: number
  porcentaje_mortalidad?: number // Calculado automáticamente
  status: string
  metadata?: Record<string, any>
  created_at?: string
  updated_at?: string
  created_by?: string
}

export interface CreateSiteDTO {
  tenant_id: string
  code: string
  name: string
  location?: SiteLocation
  fecha_apertura: string
  numero_jaulas: number
  cepa: string
  tipo_alimentacion: string
  biomasa_promedio: number
  cantidad_inicial_peces: number
  cantidad_actual_peces: number
  status?: string
  metadata?: Record<string, any>
}

export interface UpdateSiteDTO {
  name: string
  location?: SiteLocation
  fecha_apertura: string
  numero_jaulas: number
  cepa: string
  tipo_alimentacion: string
  biomasa_promedio: number
  cantidad_inicial_peces: number
  cantidad_actual_peces: number
  status: string
  metadata?: Record<string, any>
}

const siteService = {
  /**
   * Obtener todos los centros de cultivo
   * @param params Parámetros opcionales de filtrado
   */
  getAll: async (params?: {
    tenant_id?: string
    tenant_code?: string
    status?: string
    search?: string
  }) => {
    const queryParams = new URLSearchParams()

    if (params?.tenant_id) queryParams.append('tenant_id', params.tenant_id)
    if (params?.tenant_code) queryParams.append('tenant_code', params.tenant_code)
    if (params?.status) queryParams.append('status', params.status)
    if (params?.search) queryParams.append('search', params.search)

    const url = `/api/sites${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
    const response = await api.get(url)
    return response.data
  },

  /**
   * Obtener un centro de cultivo por ID
   * @param id ID del centro de cultivo
   */
  getById: async (id: string) => {
    const response = await api.get(`/api/sites/get?id=${id}`)
    return response.data
  },

  /**
   * Crear un nuevo centro de cultivo
   * @param site Datos del centro de cultivo
   */
  create: async (site: CreateSiteDTO) => {
    const response = await api.post('/api/sites/create', site)
    return response.data
  },

  /**
   * Actualizar un centro de cultivo existente
   * @param id ID del centro de cultivo
   * @param site Datos actualizados
   */
  update: async (id: string, site: UpdateSiteDTO) => {
    const response = await api.put(`/api/sites/update?id=${id}`, site)
    return response.data
  },

  /**
   * Eliminar un centro de cultivo
   * @param id ID del centro de cultivo
   */
  delete: async (id: string) => {
    const response = await api.delete(`/api/sites/delete?id=${id}`)
    return response.data
  },

  /**
   * Obtener centros de cultivo por tenant
   * @param tenantId ID del tenant
   */
  getByTenant: async (tenantId: string) => {
    return siteService.getAll({ tenant_id: tenantId })
  },
}

export default siteService
