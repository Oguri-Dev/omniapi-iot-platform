import api from './api'

export interface EndpointInstance {
  instance_id: string
  endpoint_id: string
  label: string
  method: string
  path: string
  target_block: string
  params?: Record<string, string>
  enabled: boolean
  interval_ms?: number // Intervalo específico para este endpoint (en ms)
}

export interface PollingConfig {
  id?: string
  provider: string
  site_id: string
  site_code: string
  site_name: string
  tenant_id: string
  tenant_code: string
  service_id: string
  endpoints: EndpointInstance[]
  interval_ms: number
  auto_start: boolean
  status: string
  output?: OutputConfig // Configuración de salida MQTT
  created_at?: string
  updated_at?: string
}

export interface WorkerStatus {
  instance_id: string
  endpoint_id: string
  label: string
  status: string
  interval_ms: number
  last_poll_at?: string
  last_success_at?: string
  last_error_at?: string
  last_error?: string
  total_polls: number
  total_success: number
  total_errors: number
  avg_latency_ms: number
  consecutive_errors: number
}

export interface EngineStatus {
  status: string
  active_workers: number
  total_configs: number
  workers: Record<string, WorkerStatus>
  started_at?: string
}

export interface StartPollingRequest {
  provider: string
  site_id: string
  site_code: string
  site_name: string
  tenant_id: string
  tenant_code: string
  service_id: string
  endpoints: EndpointInstance[]
  interval_ms?: number
  auto_start?: boolean
  output?: OutputConfig // Configuración de salida MQTT
}

// Configuración de salida MQTT
export interface OutputConfig {
  broker_id: string
  topic_template: string
  enabled: boolean
}

export interface StopPollingRequest {
  config_id?: string
  site_id?: string
  provider?: string
  instance_id?: string
}

const pollingService = {
  /**
   * Inicia polling para una configuración
   */
  startPolling: async (request: StartPollingRequest) => {
    const response = await api.post('/api/polling/start', request)
    return response.data
  },

  /**
   * Detiene polling según criterios
   */
  stopPolling: async (request: StopPollingRequest) => {
    const response = await api.post('/api/polling/stop', request)
    return response.data
  },

  /**
   * Obtiene el estado del engine
   */
  getStatus: async (): Promise<{ success: boolean; data: EngineStatus }> => {
    const response = await api.get('/api/polling/status')
    return response.data
  },

  /**
   * Lista todas las configuraciones activas
   */
  listConfigs: async (): Promise<{ success: boolean; data: PollingConfig[]; count: number }> => {
    const response = await api.get('/api/polling/configs')
    return response.data
  },

  /**
   * Obtiene el estado de una configuración específica
   */
  getConfig: async (
    configId: string
  ): Promise<{
    success: boolean
    data: { config: PollingConfig; workers: WorkerStatus[] }
  }> => {
    const response = await api.get(`/api/polling/config?id=${configId}`)
    return response.data
  },
}

export default pollingService
