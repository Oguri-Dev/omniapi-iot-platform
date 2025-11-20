import React, { useState, useEffect } from 'react'
import externalServiceService, {
  type ExternalService,
  type CreateExternalServiceDTO,
  type ServiceCredentials,
} from '../services/externalService.service'
import siteService, { type Site } from '../services/site.service'
import '../styles/ExternalServices.css'

const ExternalServices: React.FC = () => {
  const [services, setServices] = useState<ExternalService[]>([])
  const [sites, setSites] = useState<Site[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingService, setEditingService] = useState<ExternalService | null>(null)
  const [testingServiceId, setTestingServiceId] = useState<string | null>(null)

  // Helper function para obtener el ID del servicio de manera segura
  const getServiceId = (service: ExternalService): string => {
    return service._id || service.id || ''
  }

  // Filtros
  const [selectedSite, setSelectedSite] = useState<string>('')
  const [selectedType, setSelectedType] = useState<string>('')
  const [searchTerm, setSearchTerm] = useState('')

  // Form data
  const defaultScaleAQConfig = {
    scale_version: '2025-01-01',
    accept_header: 'application/json',
    scaleaq_site_id: '',
  }

  const createInitialFormData = (): CreateExternalServiceDTO => ({
    site_id: '',
    name: '',
    service_type: 'scaleaq',
    base_url: '',
    credentials: {},
    config: { ...defaultScaleAQConfig },
  })

  const [formData, setFormData] = useState<CreateExternalServiceDTO>(createInitialFormData())

  useEffect(() => {
    loadData()
  }, [selectedSite, selectedType, searchTerm])

  const loadData = async () => {
    try {
      setLoading(true)

      // Cargar sites
      const sitesResponse = await siteService.getAll()
      if (sitesResponse.success) {
        setSites(sitesResponse.data)
      }

      // Cargar servicios con filtros
      const params: any = {}
      if (selectedSite) params.site_id = selectedSite
      if (selectedType) params.service_type = selectedType
      if (searchTerm) params.search = searchTerm

      const servicesResponse = await externalServiceService.getAll(params)
      if (servicesResponse.success && Array.isArray(servicesResponse.data)) {
        setServices(servicesResponse.data)
      } else {
        console.warn('Respuesta de servicios no válida:', servicesResponse)
        setServices([])
      }
    } catch (error) {
      console.error('Error cargando datos:', error)
      setServices([]) // Asegurar que services sea un array vacío en caso de error
      setSites([])
    } finally {
      setLoading(false)
    }
  }

  const handleOpenModal = (service?: ExternalService) => {
    if (service) {
      setEditingService(service)
      setFormData({
        site_id: service.site_id,
        name: service.name,
        service_type: service.service_type,
        base_url: service.base_url,
        credentials: {},
        config:
          service.service_type === 'scaleaq'
            ? { ...defaultScaleAQConfig, ...(service.config || {}) }
            : service.config || {},
      })
    } else {
      setEditingService(null)
      setFormData(createInitialFormData())
    }
    setShowModal(true)
  }

  const handleCloseModal = () => {
    setShowModal(false)
    setEditingService(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      if (editingService) {
        const serviceId = getServiceId(editingService)

        if (!serviceId) {
          alert('ID inválido para el servicio seleccionado')
          return
        }

        await externalServiceService.update(serviceId, {
          name: formData.name,
          base_url: formData.base_url,
          credentials: formData.credentials,
          config: formData.config,
        })
      } else {
        await externalServiceService.create(formData)
      }

      handleCloseModal()
      loadData()
    } catch (error: any) {
      alert(error.response?.data?.error || 'Error al guardar servicio')
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('¿Está seguro de eliminar este servicio externo?')) return

    try {
      await externalServiceService.delete(id)
      loadData()
    } catch (error: any) {
      alert(error.response?.data?.error || 'Error al eliminar servicio')
    }
  }

  const handleTestConnection = async (id: string) => {
    if (!id) {
      alert('❌ Error: ID del servicio no válido')
      return
    }

    setTestingServiceId(id)

    try {
      const response = await externalServiceService.testConnection(id)

      if (response.success) {
        alert(
          `✅ Conexión exitosa!\n\nTipo: ${response.data?.token_type}\nExpira en: ${response.data?.expires_in}s\nToken length: ${response.data?.token_length} caracteres`
        )
        loadData() // Recargar para actualizar status
      } else {
        alert(`❌ Error: ${response.error}`)
      }
    } catch (error: any) {
      alert(error.response?.data?.error || 'Error probando conexión')
    } finally {
      setTestingServiceId(null)
    }
  }

  const handleCredentialChange = (field: keyof ServiceCredentials, value: string) => {
    setFormData({
      ...formData,
      credentials: {
        ...formData.credentials,
        [field]: value,
      },
    })
  }

  const handleConfigChange = (field: string, value: string) => {
    setFormData({
      ...formData,
      config: {
        ...(formData.config || {}),
        [field]: value,
      },
    })
  }

  const getServiceIcon = (type: string) => {
    const icons: Record<string, string> = {
      scaleaq: '🐟',
      innovex: '🔬',
      apikey: '🔑',
      custom: '⚙️',
    }
    return icons[type] || '🔌'
  }

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: '#28a745',
      inactive: '#6c757d',
      error: '#dc3545',
    }
    return colors[status] || '#6c757d'
  }

  // Agrupar servicios por site
  const groupedServices = (services || []).reduce((acc, service) => {
    const siteKey = service.site_id || 'sin-site'
    if (!acc[siteKey]) {
      acc[siteKey] = []
    }
    acc[siteKey].push(service)
    return acc
  }, {} as Record<string, ExternalService[]>)

  if (loading) {
    return <div className="loading">Cargando servicios externos...</div>
  }

  return (
    <div className="external-services-container">
      <div className="header">
        <h1>🔌 Servicios Externos</h1>
        <button className="btn-primary" onClick={() => handleOpenModal()}>
          + Nuevo Servicio
        </button>
      </div>

      <div className="filters">
        <select
          value={selectedSite}
          onChange={(e) => setSelectedSite(e.target.value)}
          className="filter-select"
        >
          <option value="">Todos los centros</option>
          {sites.map((site) => (
            <option key={site.id} value={site.id}>
              {site.name} ({site.tenant_code})
            </option>
          ))}
        </select>

        <select
          value={selectedType}
          onChange={(e) => setSelectedType(e.target.value)}
          className="filter-select"
        >
          <option value="">Todos los tipos</option>
          <option value="scaleaq">ScaleAQ</option>
          <option value="innovex">Innovex</option>
          <option value="apikey">API Key</option>
          <option value="custom">Custom</option>
        </select>

        <input
          type="text"
          placeholder="🔍 Buscar por nombre o código..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="search-input"
        />
      </div>

      <div className="services-list">
        {Object.keys(groupedServices).length === 0 ? (
          <div className="empty-state">
            <p>No hay servicios externos configurados</p>
            <button className="btn-primary" onClick={() => handleOpenModal()}>
              Crear primer servicio
            </button>
          </div>
        ) : (
          Object.entries(groupedServices).map(([siteId, siteServices]) => {
            const site = sites.find((s) => s.id === siteId)

            return (
              <div key={siteId} className="site-group">
                <h3 className="site-header">
                  🏭 {site?.name || 'Sin Centro Asignado'}
                  {site && <span className="site-tenant">({site.tenant_code})</span>}
                </h3>

                <div className="services-grid">
                  {siteServices.map((service) => (
                    <div key={service._id} className="service-card">
                      <div className="service-header">
                        <div className="service-title">
                          <span className="service-icon">
                            {getServiceIcon(service.service_type)}
                          </span>
                          <h4>{service.name}</h4>
                        </div>
                        <div
                          className="service-status"
                          style={{ backgroundColor: getStatusColor(service.status) }}
                        >
                          {service.status}
                        </div>
                      </div>

                      <div className="service-info">
                        <p>
                          <strong>Tipo:</strong> {service.service_type.toUpperCase()}
                        </p>
                        <p>
                          <strong>Código:</strong> {service.code}
                        </p>
                        <p>
                          <strong>URL:</strong> <span className="url-text">{service.base_url}</span>
                        </p>

                        {service.last_auth && (
                          <p className="last-auth">
                            <strong>Última auth:</strong>{' '}
                            {new Date(service.last_auth).toLocaleString()}
                          </p>
                        )}

                        {service.last_error && (
                          <p className="error-text">
                            <strong>Error:</strong> {service.last_error}
                          </p>
                        )}
                      </div>

                      <div className="service-actions">
                        <button
                          className="btn-test"
                          onClick={() => handleTestConnection(getServiceId(service))}
                          disabled={testingServiceId === getServiceId(service)}
                        >
                          {testingServiceId === getServiceId(service)
                            ? '🔄 Probando...'
                            : '🔌 Probar Conexión'}
                        </button>
                        <button className="btn-edit" onClick={() => handleOpenModal(service)}>
                          ✏️ Editar
                        </button>
                        <button
                          className="btn-delete"
                          onClick={() => handleDelete(getServiceId(service))}
                        >
                          🗑️ Eliminar
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )
          })
        )}
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={handleCloseModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>{editingService ? 'Editar Servicio' : 'Nuevo Servicio Externo'}</h2>
              <button className="modal-close" onClick={handleCloseModal}>
                ×
              </button>
            </div>

            <form onSubmit={handleSubmit}>
              <div className="form-section">
                <h3>📋 Información General</h3>

                <div className="form-group">
                  <label>Centro de Cultivo *</label>
                  <select
                    value={formData.site_id}
                    onChange={(e) => setFormData({ ...formData, site_id: e.target.value })}
                    required
                    disabled={!!editingService}
                  >
                    <option value="">Seleccionar centro...</option>
                    {sites.map((site) => (
                      <option key={site.id} value={site.id}>
                        {site.name} - {site.tenant_code}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label>Nombre del Servicio *</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="Ej: ScaleAQ Producción"
                    required
                  />
                </div>

                <div className="form-group">
                  <label>Tipo de Servicio *</label>
                  <select
                    value={formData.service_type}
                    onChange={(e) => {
                      const newType = e.target.value as CreateExternalServiceDTO['service_type']
                      setFormData({
                        ...formData,
                        service_type: newType,
                        credentials: {},
                        config:
                          newType === 'scaleaq'
                            ? { ...defaultScaleAQConfig, ...(formData.config || {}) }
                            : {},
                      })
                    }}
                    required
                    disabled={!!editingService}
                  >
                    <option value="scaleaq">ScaleAQ (Bearer Token)</option>
                    <option value="innovex">Innovex (OAuth2)</option>
                    <option value="apikey">API Key</option>
                    <option value="custom">Custom</option>
                  </select>
                </div>

                <div className="form-group">
                  <label>URL Base *</label>
                  <input
                    type="url"
                    value={formData.base_url}
                    onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                    placeholder="https://api.ejemplo.com"
                    required
                  />
                </div>
              </div>

              <div className="form-section">
                <h3>🔐 Credenciales</h3>

                {formData.service_type === 'scaleaq' && (
                  <>
                    <div className="form-group">
                      <label>Usuario</label>
                      <input
                        type="text"
                        value={formData.credentials?.username || ''}
                        onChange={(e) => handleCredentialChange('username', e.target.value)}
                        placeholder="usuario@scaleaq.com"
                      />
                    </div>
                    <div className="form-group">
                      <label>Password</label>
                      <input
                        type="password"
                        value={formData.credentials?.password || ''}
                        onChange={(e) => handleCredentialChange('password', e.target.value)}
                        placeholder="••••••••"
                      />
                    </div>
                    <div className="form-group">
                      <label>Scale-Version header</label>
                      <input
                        type="text"
                        value={
                          (formData.config?.scale_version as string) ||
                          defaultScaleAQConfig.scale_version
                        }
                        onChange={(e) => handleConfigChange('scale_version', e.target.value)}
                        placeholder="2025-01-01"
                      />
                    </div>
                    <div className="form-group">
                      <label>Accept header</label>
                      <input
                        type="text"
                        value={
                          (formData.config?.accept_header as string) ||
                          defaultScaleAQConfig.accept_header
                        }
                        onChange={(e) => handleConfigChange('accept_header', e.target.value)}
                        placeholder="application/json"
                      />
                    </div>
                    <div className="form-group">
                      <label>ScaleAQ Site ID</label>
                      <input
                        type="text"
                        value={(formData.config?.scaleaq_site_id as string) || ''}
                        onChange={(e) => handleConfigChange('scaleaq_site_id', e.target.value)}
                        placeholder="Ej: 1521"
                      />
                    </div>
                  </>
                )}

                {formData.service_type === 'innovex' && (
                  <>
                    <div className="form-group">
                      <label>Client ID</label>
                      <input
                        type="text"
                        value={formData.credentials?.client_id || ''}
                        onChange={(e) => handleCredentialChange('client_id', e.target.value)}
                        placeholder="client_id"
                      />
                    </div>
                    <div className="form-group">
                      <label>Client Secret</label>
                      <input
                        type="password"
                        value={formData.credentials?.client_secret || ''}
                        onChange={(e) => handleCredentialChange('client_secret', e.target.value)}
                        placeholder="••••••••"
                      />
                    </div>
                    <div className="form-group">
                      <label>Usuario</label>
                      <input
                        type="text"
                        value={formData.credentials?.username || ''}
                        onChange={(e) => handleCredentialChange('username', e.target.value)}
                        placeholder="OmniFish"
                      />
                    </div>
                    <div className="form-group">
                      <label>Password</label>
                      <input
                        type="password"
                        value={formData.credentials?.password || ''}
                        onChange={(e) => handleCredentialChange('password', e.target.value)}
                        placeholder="••••••••"
                      />
                    </div>
                  </>
                )}

                {formData.service_type === 'apikey' && (
                  <div className="form-group">
                    <label>API Key</label>
                    <input
                      type="password"
                      value={formData.credentials?.api_key || ''}
                      onChange={(e) => handleCredentialChange('api_key', e.target.value)}
                      placeholder="••••••••"
                    />
                  </div>
                )}

                {formData.service_type === 'custom' && (
                  <p className="info-text">
                    Para servicios custom, configure las credenciales después de crear el servicio
                  </p>
                )}
              </div>

              <div className="modal-actions">
                <button type="button" className="btn-secondary" onClick={handleCloseModal}>
                  Cancelar
                </button>
                <button type="submit" className="btn-primary">
                  {editingService ? 'Actualizar' : 'Crear'} Servicio
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

export default ExternalServices
