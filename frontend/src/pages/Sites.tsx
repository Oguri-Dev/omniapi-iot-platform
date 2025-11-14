import React, { useState, useEffect } from 'react'
import siteService, { type Site, type CreateSiteDTO } from '../services/site.service'
import tenantService, { type Tenant } from '../services/tenant.service'
import '../styles/Sites.css'

const Sites: React.FC = () => {
  const [sites, setSites] = useState<Site[]>([])
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedTenant, setSelectedTenant] = useState<string>('')
  const [showModal, setShowModal] = useState(false)
  const [editingSite, setEditingSite] = useState<Site | null>(null)

  // Cargar tenants y sites
  const loadData = async () => {
    try {
      setLoading(true)
      setError('')

      const [tenantsRes, sitesRes] = await Promise.all([
        tenantService.getAll({ status: 'active' }),
        siteService.getAll({
          search: searchTerm,
          tenant_code: selectedTenant || undefined,
        }),
      ])

      if (tenantsRes.success) {
        setTenants(tenantsRes.data || [])
      }

      if (sitesRes.success) {
        setSites(sitesRes.data || [])
      } else {
        setError(sitesRes.message || 'Error cargando centros')
      }
    } catch (err: any) {
      setError(err.response?.data?.message || 'Error cargando datos')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  // Buscar con debounce
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchTerm !== undefined || selectedTenant !== undefined) {
        loadData()
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [searchTerm, selectedTenant])

  const handleCreate = () => {
    setEditingSite(null)
    setShowModal(true)
  }

  const handleEdit = (site: Site) => {
    setEditingSite(site)
    setShowModal(true)
  }

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`¿Estás seguro de eliminar el centro "${name}"?`)) {
      return
    }

    try {
      const response = await siteService.delete(id)
      if (response.success) {
        loadData()
      } else {
        alert(response.message || 'Error eliminando centro')
      }
    } catch (err: any) {
      alert(err.response?.data?.message || 'Error eliminando centro')
    }
  }

  const handleModalClose = (reload: boolean) => {
    setShowModal(false)
    setEditingSite(null)
    if (reload) {
      loadData()
    }
  }

  // Agrupar sites por tenant
  const sitesByTenant = sites.reduce((acc, site) => {
    const tenantCode = site.tenant_code
    if (!acc[tenantCode]) {
      acc[tenantCode] = []
    }
    acc[tenantCode].push(site)
    return acc
  }, {} as Record<string, Site[]>)

  return (
    <div className="sites-page">
      <div className="page-header">
        <div>
          <h1>🏭 Centros de Cultivo</h1>
          <p>Administra los centros de producción acuícola</p>
        </div>
        <button className="btn-primary" onClick={handleCreate}>
          + Nuevo Centro
        </button>
      </div>

      <div className="filters-bar">
        <input
          type="text"
          placeholder="Buscar por nombre, código o cepa..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="search-input"
        />

        <select
          value={selectedTenant}
          onChange={(e) => setSelectedTenant(e.target.value)}
          className="filter-select"
        >
          <option value="">Todas las empresas</option>
          {tenants.map((tenant) => (
            <option key={tenant.id} value={tenant.code}>
              {tenant.name}
            </option>
          ))}
        </select>
      </div>

      {error && (
        <div className="error-message">
          <span>⚠️</span> {error}
        </div>
      )}

      {loading ? (
        <div className="loading-state">
          <div className="spinner"></div>
          <p>Cargando centros de cultivo...</p>
        </div>
      ) : sites.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">🏭</div>
          <h3>No hay centros de cultivo</h3>
          <p>Comienza creando tu primer centro de producción</p>
          <button className="btn-primary" onClick={handleCreate}>
            + Crear Primer Centro
          </button>
        </div>
      ) : (
        <div className="sites-container">
          {Object.entries(sitesByTenant).map(([tenantCode, tenantSites]) => {
            const tenant = tenants.find((t) => t.code === tenantCode)

            return (
              <div key={tenantCode} className="tenant-group">
                <h2 className="tenant-group-title">
                  🏢 {tenant?.name || tenantCode}
                  <span className="site-count">
                    {tenantSites.length} centro{tenantSites.length !== 1 ? 's' : ''}
                  </span>
                </h2>

                <div className="sites-grid">
                  {tenantSites.map((site) => (
                    <div key={site.id} className="site-card">
                      <div className="site-header">
                        <div>
                          <h3>{site.name}</h3>
                          <p className="site-code">{site.code}</p>
                        </div>
                        <span className={`status-badge status-${site.status}`}>
                          {site.status === 'active'
                            ? '🟢 Activo'
                            : site.status === 'maintenance'
                            ? '🟡 Mantención'
                            : '🔴 Inactivo'}
                        </span>
                      </div>

                      <div className="site-stats">
                        <div className="stat-item">
                          <span className="stat-label">🐟 Cepa</span>
                          <span className="stat-value">{site.cepa}</span>
                        </div>
                        <div className="stat-item">
                          <span className="stat-label">🎯 Jaulas</span>
                          <span className="stat-value">{site.numero_jaulas}</span>
                        </div>
                        <div className="stat-item">
                          <span className="stat-label">⚖️ Biomasa</span>
                          <span className="stat-value">{site.biomasa_promedio.toFixed(1)} ton</span>
                        </div>
                        <div className="stat-item">
                          <span className="stat-label">💀 Mortalidad</span>
                          <span
                            className={`stat-value ${
                              site.porcentaje_mortalidad! > 5 ? 'high-mortality' : ''
                            }`}
                          >
                            {site.porcentaje_mortalidad?.toFixed(2)}%
                          </span>
                        </div>
                      </div>

                      <div className="site-details">
                        <div className="detail-row">
                          <span className="detail-label">🍽️ Alimentación:</span>
                          <span className="detail-value">{site.tipo_alimentacion}</span>
                        </div>
                        <div className="detail-row">
                          <span className="detail-label">📊 Peces:</span>
                          <span className="detail-value">
                            {site.cantidad_actual_peces.toLocaleString()} /{' '}
                            {site.cantidad_inicial_peces.toLocaleString()}
                          </span>
                        </div>
                        {site.location && (
                          <div className="detail-row">
                            <span className="detail-label">📍 Ubicación:</span>
                            <span className="detail-value">
                              {site.location.commune || 'Sin comuna'}
                              {site.location.water_body && ` - ${site.location.water_body}`}
                            </span>
                          </div>
                        )}
                        <div className="detail-row">
                          <span className="detail-label">📅 Apertura:</span>
                          <span className="detail-value">
                            {new Date(site.fecha_apertura).toLocaleDateString('es-CL')}
                          </span>
                        </div>
                      </div>

                      {site.location && (
                        <div className="coordinates">
                          🗺️ {site.location.latitude.toFixed(5)},{' '}
                          {site.location.longitude.toFixed(5)}
                        </div>
                      )}

                      <div className="site-actions">
                        <button className="btn-secondary btn-sm" onClick={() => handleEdit(site)}>
                          ✏️ Editar
                        </button>
                        <button
                          className="btn-danger btn-sm"
                          onClick={() => handleDelete(site.id!, site.name)}
                        >
                          🗑️ Eliminar
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {showModal && <SiteModal site={editingSite} tenants={tenants} onClose={handleModalClose} />}
    </div>
  )
}

// Modal para crear/editar site
interface SiteModalProps {
  site: Site | null
  tenants: Tenant[]
  onClose: (reload: boolean) => void
}

const SiteModal: React.FC<SiteModalProps> = ({ site, tenants, onClose }) => {
  const [formData, setFormData] = useState({
    tenant_id: site?.tenant_id || '',
    code: site?.code || '',
    name: site?.name || '',
    latitude: site?.location?.latitude?.toString() || '',
    longitude: site?.location?.longitude?.toString() || '',
    region: site?.location?.region || '',
    commune: site?.location?.commune || '',
    water_body: site?.location?.water_body || '',
    fecha_apertura: site?.fecha_apertura?.split('T')[0] || new Date().toISOString().split('T')[0],
    numero_jaulas: site?.numero_jaulas?.toString() || '',
    cepa: site?.cepa || 'Atlantic Salmon',
    tipo_alimentacion: site?.tipo_alimentacion || 'monorracion',
    biomasa_promedio: site?.biomasa_promedio?.toString() || '',
    cantidad_inicial_peces: site?.cantidad_inicial_peces?.toString() || '',
    cantidad_actual_peces: site?.cantidad_actual_peces?.toString() || '',
    status: site?.status || 'active',
  })

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    })
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const siteData: any = {
        name: formData.name,
        numero_jaulas: parseInt(formData.numero_jaulas),
        cepa: formData.cepa,
        tipo_alimentacion: formData.tipo_alimentacion,
        biomasa_promedio: parseFloat(formData.biomasa_promedio),
        cantidad_inicial_peces: parseInt(formData.cantidad_inicial_peces),
        cantidad_actual_peces: parseInt(formData.cantidad_actual_peces),
        fecha_apertura: new Date(formData.fecha_apertura).toISOString(),
        status: formData.status,
      }

      // Agregar location si hay coordenadas
      if (formData.latitude && formData.longitude) {
        siteData.location = {
          latitude: parseFloat(formData.latitude),
          longitude: parseFloat(formData.longitude),
          region: formData.region || undefined,
          commune: formData.commune || undefined,
          water_body: formData.water_body || undefined,
        }
      }

      let response
      if (site?.id) {
        // Actualizar
        response = await siteService.update(site.id, siteData)
      } else {
        // Crear (incluir campos adicionales)
        const createData: CreateSiteDTO = {
          ...siteData,
          tenant_id: formData.tenant_id,
          code: formData.code,
        }
        response = await siteService.create(createData)
      }

      if (response.success) {
        onClose(true)
      } else {
        setError(response.message || 'Error guardando centro')
      }
    } catch (err: any) {
      setError(err.response?.data?.message || 'Error guardando centro')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={() => onClose(false)}>
      <div className="modal-content modal-large" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{site ? '✏️ Editar Centro' : '➕ Nuevo Centro de Cultivo'}</h2>
          <button className="modal-close" onClick={() => onClose(false)}>
            ✕
          </button>
        </div>

        {error && (
          <div className="error-message">
            <span>⚠️</span> {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="site-form">
          <div className="form-section">
            <h3>Información General</h3>

            {!site && (
              <div className="form-group">
                <label htmlFor="tenant_id">Empresa *</label>
                <select
                  id="tenant_id"
                  name="tenant_id"
                  value={formData.tenant_id}
                  onChange={handleChange}
                  required
                >
                  <option value="">Seleccionar empresa...</option>
                  {tenants.map((tenant) => (
                    <option key={tenant.id} value={tenant.id}>
                      {tenant.name}
                    </option>
                  ))}
                </select>
              </div>
            )}

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="name">Nombre del Centro *</label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  required
                  placeholder="Ej: Centro Reloncaví"
                />
              </div>

              <div className="form-group">
                <label htmlFor="code">Código (único) *</label>
                <input
                  type="text"
                  id="code"
                  name="code"
                  value={formData.code}
                  onChange={handleChange}
                  required
                  disabled={!!site}
                  placeholder="Ej: mowi-reloncavi-1"
                />
                {!site && <small>Se normalizará automáticamente</small>}
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="fecha_apertura">Fecha de Apertura *</label>
                <input
                  type="date"
                  id="fecha_apertura"
                  name="fecha_apertura"
                  value={formData.fecha_apertura}
                  onChange={handleChange}
                  required
                />
              </div>

              <div className="form-group">
                <label htmlFor="status">Estado</label>
                <select id="status" name="status" value={formData.status} onChange={handleChange}>
                  <option value="active">Activo</option>
                  <option value="inactive">Inactivo</option>
                  <option value="maintenance">En Mantención</option>
                </select>
              </div>
            </div>
          </div>

          <div className="form-section">
            <h3>Ubicación</h3>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="latitude">Latitud</label>
                <input
                  type="number"
                  step="any"
                  id="latitude"
                  name="latitude"
                  value={formData.latitude}
                  onChange={handleChange}
                  placeholder="-41.4693"
                />
              </div>

              <div className="form-group">
                <label htmlFor="longitude">Longitud</label>
                <input
                  type="number"
                  step="any"
                  id="longitude"
                  name="longitude"
                  value={formData.longitude}
                  onChange={handleChange}
                  placeholder="-72.9318"
                />
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="region">Región</label>
                <input
                  type="text"
                  id="region"
                  name="region"
                  value={formData.region}
                  onChange={handleChange}
                  placeholder="Los Lagos"
                />
              </div>

              <div className="form-group">
                <label htmlFor="commune">Comuna</label>
                <input
                  type="text"
                  id="commune"
                  name="commune"
                  value={formData.commune}
                  onChange={handleChange}
                  placeholder="Puerto Montt"
                />
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="water_body">Cuerpo de Agua</label>
              <input
                type="text"
                id="water_body"
                name="water_body"
                value={formData.water_body}
                onChange={handleChange}
                placeholder="Seno Reloncaví"
              />
            </div>
          </div>

          <div className="form-section">
            <h3>Datos de Producción</h3>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="cepa">Cepa (Especie) *</label>
                <select
                  id="cepa"
                  name="cepa"
                  value={formData.cepa}
                  onChange={handleChange}
                  required
                >
                  <option value="Atlantic Salmon">Atlantic Salmon (Salmón del Atlántico)</option>
                  <option value="Coho Salmon">Coho Salmon (Salmón Coho)</option>
                  <option value="Rainbow Trout">Rainbow Trout (Trucha Arcoíris)</option>
                  <option value="Chinook Salmon">Chinook Salmon (Salmón Chinook)</option>
                </select>
              </div>

              <div className="form-group">
                <label htmlFor="tipo_alimentacion">Tipo de Alimentación *</label>
                <select
                  id="tipo_alimentacion"
                  name="tipo_alimentacion"
                  value={formData.tipo_alimentacion}
                  onChange={handleChange}
                  required
                >
                  <option value="monorracion">Monoración</option>
                  <option value="ciclico">Cíclico</option>
                  <option value="mixto">Mixto</option>
                </select>
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="numero_jaulas">Número de Jaulas *</label>
                <input
                  type="number"
                  id="numero_jaulas"
                  name="numero_jaulas"
                  value={formData.numero_jaulas}
                  onChange={handleChange}
                  required
                  min="1"
                  placeholder="10"
                />
              </div>

              <div className="form-group">
                <label htmlFor="biomasa_promedio">Biomasa Promedio (ton) *</label>
                <input
                  type="number"
                  step="0.1"
                  id="biomasa_promedio"
                  name="biomasa_promedio"
                  value={formData.biomasa_promedio}
                  onChange={handleChange}
                  required
                  min="0"
                  placeholder="150.5"
                />
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="cantidad_inicial_peces">Cantidad Inicial de Peces *</label>
                <input
                  type="number"
                  id="cantidad_inicial_peces"
                  name="cantidad_inicial_peces"
                  value={formData.cantidad_inicial_peces}
                  onChange={handleChange}
                  required
                  min="0"
                  placeholder="500000"
                />
              </div>

              <div className="form-group">
                <label htmlFor="cantidad_actual_peces">Cantidad Actual de Peces *</label>
                <input
                  type="number"
                  id="cantidad_actual_peces"
                  name="cantidad_actual_peces"
                  value={formData.cantidad_actual_peces}
                  onChange={handleChange}
                  required
                  min="0"
                  placeholder="475000"
                />
              </div>
            </div>

            {formData.cantidad_inicial_peces && formData.cantidad_actual_peces && (
              <div className="mortality-indicator">
                <strong>Mortalidad Calculada:</strong>{' '}
                {(
                  ((parseInt(formData.cantidad_inicial_peces) -
                    parseInt(formData.cantidad_actual_peces)) /
                    parseInt(formData.cantidad_inicial_peces)) *
                  100
                ).toFixed(2)}
                %
              </div>
            )}
          </div>

          <div className="modal-actions">
            <button
              type="button"
              className="btn-secondary"
              onClick={() => onClose(false)}
              disabled={loading}
            >
              Cancelar
            </button>
            <button type="submit" className="btn-primary" disabled={loading}>
              {loading ? (
                <>
                  <span className="spinner-sm"></span> Guardando...
                </>
              ) : (
                <>💾 {site ? 'Actualizar' : 'Crear'} Centro</>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default Sites
