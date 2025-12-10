import api from './api'

// Tipos para Broker
export interface BrokerConfig {
  id?: string
  name: string
  broker_url: string
  client_id: string
  username?: string
  password?: string
  qos: number
  retained: boolean
  clean_session: boolean
  keep_alive: number
  enabled: boolean
  created_at?: string
  updated_at?: string
}

export interface BrokerStatus {
  id: string
  name: string
  url: string
  enabled: boolean
  connected: boolean
  stats?: {
    total_published: number
    total_success: number
    total_errors: number
    last_publish_at?: string
    last_error_at?: string
    last_error?: string
    connected_since?: string
    reconnect_count: number
  }
}

export interface BrokersStatusResponse {
  success: boolean
  data: {
    total_brokers: number
    connected_count: number
    brokers: BrokerStatus[]
  }
  timestamp: number
}

export interface TopicTemplate {
  id: string
  name: string
  pattern: string
  description: string
}

export interface TopicTemplatesResponse {
  success: boolean
  data: {
    templates: TopicTemplate[]
    variables: string[]
  }
  timestamp: number
}

// Output config para polling
export interface OutputConfig {
  broker_id: string
  topic_template: string
  enabled: boolean
}

const brokerService = {
  /**
   * Lista todos los brokers configurados
   */
  listBrokers: async (): Promise<BrokersStatusResponse> => {
    const response = await api.get('/api/brokers')
    return response.data
  },

  /**
   * Agrega un nuevo broker
   */
  addBroker: async (config: BrokerConfig) => {
    const response = await api.post('/api/brokers/add', config)
    return response.data
  },

  /**
   * Actualiza un broker existente
   */
  updateBroker: async (id: string, config: BrokerConfig) => {
    const response = await api.put(`/api/brokers/${id}`, config)
    return response.data
  },

  /**
   * Elimina un broker
   */
  removeBroker: async (id: string) => {
    const response = await api.delete(`/api/brokers/${id}`)
    return response.data
  },

  /**
   * Prueba la conexión a un broker
   */
  testConnection: async (config: BrokerConfig) => {
    const response = await api.post('/api/brokers/test', config)
    return response.data
  },

  /**
   * Obtiene los templates de topics disponibles
   */
  getTopicTemplates: async (): Promise<TopicTemplatesResponse> => {
    const response = await api.get('/api/brokers/templates')
    return response.data
  },
}

export default brokerService
