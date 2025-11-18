import React, { useState, useEffect } from 'react'
import tenantService, { type Tenant } from '../services/tenant.service'
import '../styles/Tenants.css'

const Tenants: React.FC = () => {
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null)

  // Cargar tenants
  const loadTenants = async () => {
    try {
      setLoading(true)
      setError('')
      const response = await tenantService.getAll({ search: searchTerm })
      if (response.success) {
        setTenants(response.data || [])
      } else {
        setError(response.message || 'Error cargando empresas')
      }
    } catch (err: any) {
      setError(err.response?.data?.message || 'Error cargando empresas')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadTenants()
  }, [])

  // Buscar al escribir (con debounce)
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchTerm !== undefined) {
        loadTenants()
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [searchTerm])

  // Abrir modal para crear
  const handleCreate = () => {
    setEditingTenant(null)
    setShowModal(true)
  }

  // Abrir modal para editar
  const handleEdit = (tenant: Tenant) => {
    setEditingTenant(tenant)
    setShowModal(true)
  }

  // Eliminar tenant
  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`¿Estás seguro de eliminar la empresa "${name}"?`)) {
      return
    }

    try {
      const response = await tenantService.delete(id)
      if (response.success) {
        loadTenants()
      } else {
        alert(response.message || 'Error eliminando empresa')
      }
    } catch (err: any) {
      alert(err.response?.data?.message || 'Error eliminando empresa')
    }
  }

  // Cerrar modal y recargar
  const handleModalClose = (reload: boolean) => {
    setShowModal(false)
    setEditingTenant(null)
    if (reload) {
      loadTenants()
    }
  }

  return (
    <div className="tenants-page">
      <div className="page-header">
        <div>
          <h1>🏢 Empresas Salmoneras</h1>
          <p>Administra las empresas que usan la plataforma</p>
        </div>
        <button className="btn-primary" onClick={handleCreate}>
          + Nueva Empresa
        </button>
      </div>

      <div className="search-bar">
        <input
          type="text"
          placeholder="Buscar por nombre o código..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="search-input"
        />
      </div>

      {error && (
        <div className="error-message">
          <span>⚠️</span> {error}
        </div>
      )}

      {loading ? (
        <div className="loading-state">
          <div className="spinner"></div>
          <p>Cargando empresas...</p>
        </div>
      ) : tenants.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">🏢</div>
          <h3>No hay empresas registradas</h3>
          <p>Comienza creando tu primera empresa salmonera</p>
          <button className="btn-primary" onClick={handleCreate}>
            + Crear Primera Empresa
          </button>
        </div>
      ) : (
        <div className="tenants-grid">
          {tenants.map((tenant) => (
            <div key={tenant.id} className="tenant-card">
              <div className="tenant-header">
                <div>
                  <h3>{tenant.name}</h3>
                  <p className="tenant-code">{tenant.code}</p>
                </div>
                <span className={`status-badge status-${tenant.status || 'active'}`}>
                  {tenant.status === 'active' ? '🟢 Activo' : '🔴 Inactivo'}
                </span>
              </div>

              <div className="tenant-info">
                {tenant.contact?.email && (
                  <div className="info-row">
                    <span className="info-label">📧 Email:</span>
                    <span className="info-value">{tenant.contact.email}</span>
                  </div>
                )}
                {tenant.contact?.phone && (
                  <div className="info-row">
                    <span className="info-label">📞 Teléfono:</span>
                    <span className="info-value">{tenant.contact.phone}</span>
                  </div>
                )}
                {tenant.address?.city && (
                  <div className="info-row">
                    <span className="info-label">📍 Ubicación:</span>
                    <span className="info-value">
                      {tenant.address.city}
                      {tenant.address.region && `, ${tenant.address.region}`}
                    </span>
                  </div>
                )}
              </div>

              <div className="tenant-actions">
                <button className="btn-secondary btn-sm" onClick={() => handleEdit(tenant)}>
                  ✏️ Editar
                </button>
                <button
                  className="btn-danger btn-sm"
                  onClick={() => handleDelete(tenant.id!, tenant.name)}
                >
                  🗑️ Eliminar
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showModal && <TenantModal tenant={editingTenant} onClose={handleModalClose} />}
    </div>
  )
}

// Modal para crear/editar tenant
interface TenantModalProps {
  tenant: Tenant | null
  onClose: (reload: boolean) => void
}

const TenantModal: React.FC<TenantModalProps> = ({ tenant, onClose }) => {
  const [formData, setFormData] = useState({
    code: tenant?.code || '',
    name: tenant?.name || '',
    type: tenant?.type || 'salmon_company',
    email: tenant?.contact?.email || '',
    phone: tenant?.contact?.phone || '',
    contactName: tenant?.contact?.contactName || '',
    country: tenant?.address?.country || 'Chile',
    region: tenant?.address?.region || '',
    city: tenant?.address?.city || '',
    street: tenant?.address?.street || '',
    zipCode: tenant?.address?.zipCode || '',
    status: tenant?.status || 'active',
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
      const tenantData: any = {
        code: formData.code,
        name: formData.name,
        type: formData.type,
        status: formData.status,
      }

      // Agregar contact si hay datos
      if (formData.email || formData.phone || formData.contactName) {
        tenantData.contact = {
          email: formData.email || undefined,
          phone: formData.phone || undefined,
          contactName: formData.contactName || undefined,
        }
      }

      // Agregar address si hay datos
      if (formData.country || formData.region || formData.city) {
        tenantData.address = {
          country: formData.country || undefined,
          region: formData.region || undefined,
          city: formData.city || undefined,
          street: formData.street || undefined,
          zipCode: formData.zipCode || undefined,
        }
      }

      let response
      if (tenant?.id) {
        // Actualizar
        response = await tenantService.update(tenant.id, tenantData)
      } else {
        // Crear
        response = await tenantService.create(tenantData)
      }

      if (response.success) {
        onClose(true)
      } else {
        setError(response.message || 'Error guardando empresa')
      }
    } catch (err: any) {
      setError(err.response?.data?.message || 'Error guardando empresa')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={() => onClose(false)}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{tenant ? '✏️ Editar Empresa' : '➕ Nueva Empresa'}</h2>
          <button className="modal-close" onClick={() => onClose(false)}>
            ✕
          </button>
        </div>

        {error && (
          <div className="error-message">
            <span>⚠️</span> {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="tenant-form">
          <div className="form-section">
            <h3>Información General</h3>

            <div className="form-group">
              <label htmlFor="name">Nombre *</label>
              <input
                type="text"
                id="name"
                name="name"
                value={formData.name}
                onChange={handleChange}
                required
                placeholder="Ej: Mowi Chile"
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
                disabled={!!tenant}
                placeholder="Ej: mowi-chile"
              />
              <small>Se usará en minúsculas sin espacios</small>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="type">Tipo</label>
                <select id="type" name="type" value={formData.type} onChange={handleChange}>
                  <option value="salmon_company">Empresa Salmonera</option>
                  <option value="aquaculture">Acuicultura General</option>
                  <option value="supplier">Proveedor</option>
                  <option value="other">Otro</option>
                </select>
              </div>

              <div className="form-group">
                <label htmlFor="status">Estado</label>
                <select id="status" name="status" value={formData.status} onChange={handleChange}>
                  <option value="active">Activo</option>
                  <option value="inactive">Inactivo</option>
                  <option value="suspended">Suspendido</option>
                </select>
              </div>
            </div>
          </div>

          <div className="form-section">
            <h3>Contacto</h3>

            <div className="form-group">
              <label htmlFor="contactName">Nombre de Contacto</label>
              <input
                type="text"
                id="contactName"
                name="contactName"
                value={formData.contactName}
                onChange={handleChange}
                placeholder="Ej: Juan Pérez"
              />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="email">Email</label>
                <input
                  type="email"
                  id="email"
                  name="email"
                  value={formData.email}
                  onChange={handleChange}
                  placeholder="contacto@empresa.com"
                />
              </div>

              <div className="form-group">
                <label htmlFor="phone">Teléfono</label>
                <input
                  type="tel"
                  id="phone"
                  name="phone"
                  value={formData.phone}
                  onChange={handleChange}
                  placeholder="+56 9 1234 5678"
                />
              </div>
            </div>
          </div>

          <div className="form-section">
            <h3>Ubicación</h3>

            <div className="form-row">
              <div className="form-group">
                <label htmlFor="country">País</label>
                <input
                  type="text"
                  id="country"
                  name="country"
                  value={formData.country}
                  onChange={handleChange}
                  placeholder="Chile"
                />
              </div>

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
            </div>

            <div className="form-group">
              <label htmlFor="city">Ciudad</label>
              <input
                type="text"
                id="city"
                name="city"
                value={formData.city}
                onChange={handleChange}
                placeholder="Puerto Montt"
              />
            </div>
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
                <>💾 {tenant ? 'Actualizar' : 'Crear'} Empresa</>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default Tenants
