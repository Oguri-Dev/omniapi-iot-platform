import React, { useState, useEffect } from 'react'
import connectorService from '../services/connector.service'
import type { ExternalService, CreateServiceDTO } from '../services/connector.service'

const Services: React.FC = () => {
  const [services, setServices] = useState<ExternalService[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingService, setEditingService] = useState<ExternalService | null>(null)
  const [formData, setFormData] = useState<CreateServiceDTO>({
    name: '',
    type: 'rest',
    url: '',
    auth_type: 'none',
  })

  useEffect(() => {
    loadServices()
  }, [])

  const loadServices = async () => {
    try {
      setLoading(true)
      const data = await connectorService.getAll()
      setServices(data)
    } catch (error) {
      console.error('Error loading services:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = () => {
    setEditingService(null)
    setFormData({
      name: '',
      type: 'rest',
      url: '',
      auth_type: 'none',
    })
    setShowModal(true)
  }

  const handleEdit = (service: ExternalService) => {
    setEditingService(service)
    setFormData({
      name: service.name,
      type: service.type,
      url: service.url,
      auth_type: service.auth_type || 'none',
      username: service.username,
      password: service.password,
      token: service.token,
    })
    setShowModal(true)
  }

  const handleDelete = async (id: string) => {
    if (confirm('¿Estás seguro de eliminar este servicio?')) {
      try {
        await connectorService.delete(id)
        loadServices()
      } catch (error) {
        console.error('Error deleting service:', error)
      }
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      if (editingService) {
        await connectorService.update(editingService.id, formData)
      } else {
        await connectorService.create(formData)
      }
      setShowModal(false)
      loadServices()
    } catch (error) {
      console.error('Error saving service:', error)
    }
  }

  const handleTestConnection = async (id: string) => {
    try {
      const result = await connectorService.testConnection(id)
      alert(result.message)
    } catch (error) {
      alert('Error probando conexión')
    }
  }

  if (loading) {
    return <div className="loading">Cargando servicios...</div>
  }

  return (
    <div className="services-page">
      <div className="page-header">
        <h1>Servicios Externos</h1>
        <button onClick={handleCreate} className="btn-primary">
          ➕ Nuevo Servicio
        </button>
      </div>

      <div className="services-grid">
        {services.length === 0 ? (
          <div className="empty-state">
            <p>No hay servicios configurados</p>
            <button onClick={handleCreate} className="btn-primary">
              Crear primer servicio
            </button>
          </div>
        ) : (
          services.map((service) => (
            <div key={service.id} className="service-card">
              <div className="service-header">
                <h3>{service.name}</h3>
                <span className={`status-badge status-${service.status}`}>{service.status}</span>
              </div>
              <div className="service-info">
                <p>
                  <strong>Tipo:</strong> {service.type}
                </p>
                <p>
                  <strong>URL:</strong> {service.url}
                </p>
                <p>
                  <strong>Auth:</strong> {service.auth_type || 'none'}
                </p>
              </div>
              <div className="service-actions">
                <button onClick={() => handleTestConnection(service.id)} className="btn-test">
                  🔍 Probar
                </button>
                <button onClick={() => handleEdit(service)} className="btn-edit">
                  ✏️ Editar
                </button>
                <button onClick={() => handleDelete(service.id)} className="btn-delete">
                  🗑️ Eliminar
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>{editingService ? 'Editar Servicio' : 'Nuevo Servicio'}</h2>
              <button onClick={() => setShowModal(false)} className="modal-close">
                ✕
              </button>
            </div>

            <form onSubmit={handleSubmit} className="service-form">
              <div className="form-group">
                <label>Nombre</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>

              <div className="form-group">
                <label>Tipo</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                  required
                >
                  <option value="rest">REST API</option>
                  <option value="mqtt">MQTT</option>
                  <option value="websocket">WebSocket</option>
                  <option value="graphql">GraphQL</option>
                </select>
              </div>

              <div className="form-group">
                <label>URL</label>
                <input
                  type="url"
                  value={formData.url}
                  onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                  placeholder="https://api.ejemplo.com"
                  required
                />
              </div>

              <div className="form-group">
                <label>Tipo de Autenticación</label>
                <select
                  value={formData.auth_type}
                  onChange={(e) => setFormData({ ...formData, auth_type: e.target.value })}
                >
                  <option value="none">Sin autenticación</option>
                  <option value="basic">Basic Auth</option>
                  <option value="token">Token/Bearer</option>
                  <option value="oauth">OAuth 2.0</option>
                </select>
              </div>

              {formData.auth_type === 'basic' && (
                <>
                  <div className="form-group">
                    <label>Usuario</label>
                    <input
                      type="text"
                      value={formData.username || ''}
                      onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Contraseña</label>
                    <input
                      type="password"
                      value={formData.password || ''}
                      onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    />
                  </div>
                </>
              )}

              {formData.auth_type === 'token' && (
                <div className="form-group">
                  <label>Token</label>
                  <input
                    type="text"
                    value={formData.token || ''}
                    onChange={(e) => setFormData({ ...formData, token: e.target.value })}
                    placeholder="Bearer token..."
                  />
                </div>
              )}

              <div className="form-actions">
                <button type="button" onClick={() => setShowModal(false)} className="btn-secondary">
                  Cancelar
                </button>
                <button type="submit" className="btn-primary">
                  {editingService ? 'Actualizar' : 'Crear'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default Services
