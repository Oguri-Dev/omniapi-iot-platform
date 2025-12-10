import React, { useState, useEffect, useCallback } from 'react'
import brokerService, {
  type BrokerConfig,
  type BrokerStatus,
  type TopicTemplate,
} from '../services/broker.service'
import '../styles/Brokers.css'

const Brokers: React.FC = () => {
  const [brokers, setBrokers] = useState<BrokerStatus[]>([])
  const [templates, setTemplates] = useState<TopicTemplate[]>([])
  const [variables, setVariables] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingBroker, setEditingBroker] = useState<BrokerStatus | null>(null)
  const [testingBrokerId, setTestingBrokerId] = useState<string | null>(null)
  const [connectedCount, setConnectedCount] = useState(0)

  // Form data
  const createInitialFormData = (): BrokerConfig => ({
    name: '',
    broker_url: 'tcp://localhost:1883',
    client_id: `omniapi-${Date.now()}`,
    username: '',
    password: '',
    qos: 1,
    retained: false,
    clean_session: true,
    keep_alive: 60,
    enabled: true,
  })

  const [formData, setFormData] = useState<BrokerConfig>(createInitialFormData())

  const loadData = useCallback(async (showLoading = false) => {
    try {
      if (showLoading) setLoading(true)

      // Cargar brokers
      const brokersResponse = await brokerService.listBrokers()
      if (brokersResponse.success) {
        setBrokers(brokersResponse.data.brokers || [])
        setConnectedCount(brokersResponse.data.connected_count || 0)
      }

      // Cargar templates
      const templatesResponse = await brokerService.getTopicTemplates()
      if (templatesResponse.success) {
        setTemplates(templatesResponse.data.templates || [])
        setVariables(templatesResponse.data.variables || [])
      }
    } catch (error) {
      console.error('Error cargando datos:', error)
      setBrokers([])
    } finally {
      if (showLoading) setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadData(true) // Mostrar loading solo en la carga inicial
    // Auto-refresh cada 10 segundos (sin loading)
    const interval = setInterval(() => loadData(false), 10000)
    return () => clearInterval(interval)
  }, [loadData])

  const handleOpenModal = (broker?: BrokerStatus) => {
    if (broker) {
      setEditingBroker(broker)
      setFormData({
        id: broker.id,
        name: broker.name,
        broker_url: broker.url,
        client_id: `omniapi-${broker.id}`,
        username: '',
        password: '',
        qos: 1,
        retained: false,
        clean_session: true,
        keep_alive: 60,
        enabled: broker.enabled,
      })
    } else {
      setEditingBroker(null)
      setFormData(createInitialFormData())
    }
    setShowModal(true)
  }

  const handleCloseModal = () => {
    setShowModal(false)
    setEditingBroker(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      if (editingBroker) {
        await brokerService.updateBroker(editingBroker.id, formData)
      } else {
        await brokerService.addBroker(formData)
      }

      handleCloseModal()
      loadData()
    } catch (error: any) {
      alert(error.response?.data?.message || 'Error al guardar broker')
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('¿Está seguro de eliminar este broker?')) return

    try {
      await brokerService.removeBroker(id)
      loadData()
    } catch (error: any) {
      alert(error.response?.data?.message || 'Error al eliminar broker')
    }
  }

  const handleTestConnection = async (broker: BrokerStatus) => {
    setTestingBrokerId(broker.id)

    try {
      const testConfig: BrokerConfig = {
        name: broker.name,
        broker_url: broker.url,
        client_id: `omniapi-test-${Date.now()}`,
        qos: 1,
        retained: false,
        clean_session: true,
        keep_alive: 30,
        enabled: true,
      }

      const response = await brokerService.testConnection(testConfig)
      if (response.success) {
        alert('✅ Conexión exitosa al broker')
      } else {
        alert(`❌ Error: ${response.message}`)
      }
    } catch (error: any) {
      alert(`❌ Error: ${error.response?.data?.message || 'Error de conexión'}`)
    } finally {
      setTestingBrokerId(null)
    }
  }

  const handleTestNewConnection = async () => {
    setTestingBrokerId('new')

    try {
      const response = await brokerService.testConnection(formData)
      if (response.success) {
        alert('✅ Conexión exitosa al broker')
      } else {
        alert(`❌ Error: ${response.message}`)
      }
    } catch (error: any) {
      alert(`❌ Error: ${error.response?.data?.message || 'Error de conexión'}`)
    } finally {
      setTestingBrokerId(null)
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString()
  }

  const getStatusBadge = (broker: BrokerStatus) => {
    if (!broker.enabled) {
      return <span className="status-badge disabled">Deshabilitado</span>
    }
    if (broker.connected) {
      return <span className="status-badge connected">Conectado</span>
    }
    return <span className="status-badge disconnected">Desconectado</span>
  }

  if (loading && brokers.length === 0) {
    return (
      <div className="brokers-page">
        <div className="loading">Cargando configuración de brokers...</div>
      </div>
    )
  }

  return (
    <div className="brokers-page">
      <div className="page-header">
        <div className="header-left">
          <h1>🔌 Brokers MQTT</h1>
          <p className="subtitle">Configuración de brokers para envío de datos en tiempo real</p>
        </div>
        <div className="header-right">
          <div className="stats-summary">
            <div className="stat">
              <span className="stat-value">{brokers.length}</span>
              <span className="stat-label">Total</span>
            </div>
            <div className="stat">
              <span className="stat-value connected">{connectedCount}</span>
              <span className="stat-label">Conectados</span>
            </div>
          </div>
          <button className="btn-primary" onClick={() => handleOpenModal()}>
            + Nuevo Broker
          </button>
        </div>
      </div>

      {/* Lista de Brokers */}
      <div className="brokers-grid">
        {brokers.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">📡</div>
            <h3>No hay brokers configurados</h3>
            <p>Configura un broker MQTT para enviar los datos de polling</p>
            <button className="btn-primary" onClick={() => handleOpenModal()}>
              Configurar Broker
            </button>
          </div>
        ) : (
          brokers.map((broker) => (
            <div key={broker.id} className={`broker-card ${broker.connected ? 'connected' : ''}`}>
              <div className="broker-header">
                <div className="broker-info">
                  <h3>{broker.name}</h3>
                  <span className="broker-url">{broker.url}</span>
                </div>
                {getStatusBadge(broker)}
              </div>

              {broker.stats && (
                <div className="broker-stats">
                  <div className="stat-row">
                    <span className="stat-name">Publicados:</span>
                    <span className="stat-value">{broker.stats.total_published}</span>
                  </div>
                  <div className="stat-row">
                    <span className="stat-name">Exitosos:</span>
                    <span className="stat-value success">{broker.stats.total_success}</span>
                  </div>
                  <div className="stat-row">
                    <span className="stat-name">Errores:</span>
                    <span className="stat-value error">{broker.stats.total_errors}</span>
                  </div>
                  {broker.stats.last_publish_at && (
                    <div className="stat-row">
                      <span className="stat-name">Último envío:</span>
                      <span className="stat-value">{formatDate(broker.stats.last_publish_at)}</span>
                    </div>
                  )}
                  {broker.stats.connected_since && (
                    <div className="stat-row">
                      <span className="stat-name">Conectado desde:</span>
                      <span className="stat-value">{formatDate(broker.stats.connected_since)}</span>
                    </div>
                  )}
                  {broker.stats.last_error && (
                    <div className="stat-row error-row">
                      <span className="stat-name">Último error:</span>
                      <span className="stat-value error">{broker.stats.last_error}</span>
                    </div>
                  )}
                </div>
              )}

              <div className="broker-actions">
                <button
                  className="btn-icon"
                  onClick={() => handleTestConnection(broker)}
                  disabled={testingBrokerId === broker.id}
                  title="Probar conexión"
                >
                  {testingBrokerId === broker.id ? '⏳' : '🔗'}
                </button>
                <button className="btn-icon" onClick={() => handleOpenModal(broker)} title="Editar">
                  ✏️
                </button>
                <button
                  className="btn-icon danger"
                  onClick={() => handleDelete(broker.id)}
                  title="Eliminar"
                >
                  🗑️
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Templates de Topics */}
      <div className="templates-section">
        <h2>📝 Templates de Topics</h2>
        <p className="section-description">
          Usa estos templates para configurar los topics de publicación. Las variables disponibles
          son: {variables.join(', ')}
        </p>
        <div className="templates-grid">
          {templates.map((template) => (
            <div key={template.id} className="template-card">
              <h4>{template.name}</h4>
              <code className="template-pattern">{template.pattern}</code>
              <p className="template-description">{template.description}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Modal de Crear/Editar */}
      {showModal && (
        <div className="modal-overlay" onClick={handleCloseModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>{editingBroker ? 'Editar Broker' : 'Nuevo Broker MQTT'}</h2>
              <button className="btn-close" onClick={handleCloseModal}>
                ×
              </button>
            </div>

            <form onSubmit={handleSubmit}>
              <div className="form-section">
                <h3>Información General</h3>

                <div className="form-group">
                  <label>Nombre *</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="Mi Broker MQTT"
                    required
                  />
                </div>

                <div className="form-group">
                  <label>URL del Broker *</label>
                  <input
                    type="text"
                    value={formData.broker_url}
                    onChange={(e) => setFormData({ ...formData, broker_url: e.target.value })}
                    placeholder="tcp://localhost:1883 o ssl://broker.example.com:8883"
                    required
                  />
                  <small>Usa tcp:// para conexiones no seguras o ssl:// para TLS</small>
                </div>

                <div className="form-group">
                  <label>Client ID</label>
                  <input
                    type="text"
                    value={formData.client_id}
                    onChange={(e) => setFormData({ ...formData, client_id: e.target.value })}
                    placeholder="omniapi-client"
                  />
                  <small>Identificador único para este cliente MQTT</small>
                </div>
              </div>

              <div className="form-section">
                <h3>Autenticación</h3>

                <div className="form-row">
                  <div className="form-group">
                    <label>Usuario</label>
                    <input
                      type="text"
                      value={formData.username || ''}
                      onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                      placeholder="usuario"
                    />
                  </div>

                  <div className="form-group">
                    <label>Contraseña</label>
                    <input
                      type="password"
                      value={formData.password || ''}
                      onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                      placeholder="••••••••"
                    />
                  </div>
                </div>
              </div>

              <div className="form-section">
                <h3>Configuración MQTT</h3>

                <div className="form-row">
                  <div className="form-group">
                    <label>QoS (Quality of Service)</label>
                    <select
                      value={formData.qos}
                      onChange={(e) => setFormData({ ...formData, qos: parseInt(e.target.value) })}
                    >
                      <option value={0}>0 - At most once (Fire and forget)</option>
                      <option value={1}>1 - At least once (Acknowledged)</option>
                      <option value={2}>2 - Exactly once (Assured)</option>
                    </select>
                  </div>

                  <div className="form-group">
                    <label>Keep Alive (segundos)</label>
                    <input
                      type="number"
                      value={formData.keep_alive}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          keep_alive: parseInt(e.target.value) || 60,
                        })
                      }
                      min={10}
                      max={600}
                    />
                  </div>
                </div>

                <div className="form-row checkboxes">
                  <div className="form-group checkbox">
                    <label>
                      <input
                        type="checkbox"
                        checked={formData.retained}
                        onChange={(e) => setFormData({ ...formData, retained: e.target.checked })}
                      />
                      Mensajes Retained
                    </label>
                    <small>Los mensajes se guardan en el broker</small>
                  </div>

                  <div className="form-group checkbox">
                    <label>
                      <input
                        type="checkbox"
                        checked={formData.clean_session}
                        onChange={(e) =>
                          setFormData({ ...formData, clean_session: e.target.checked })
                        }
                      />
                      Clean Session
                    </label>
                    <small>Iniciar sin estado previo</small>
                  </div>

                  <div className="form-group checkbox">
                    <label>
                      <input
                        type="checkbox"
                        checked={formData.enabled}
                        onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                      />
                      Habilitado
                    </label>
                    <small>Conectar al guardar</small>
                  </div>
                </div>
              </div>

              <div className="modal-footer">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={handleTestNewConnection}
                  disabled={testingBrokerId === 'new'}
                >
                  {testingBrokerId === 'new' ? '⏳ Probando...' : '🔗 Probar Conexión'}
                </button>
                <div className="footer-right">
                  <button type="button" className="btn-secondary" onClick={handleCloseModal}>
                    Cancelar
                  </button>
                  <button type="submit" className="btn-primary">
                    {editingBroker ? 'Actualizar' : 'Crear Broker'}
                  </button>
                </div>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Brokers
