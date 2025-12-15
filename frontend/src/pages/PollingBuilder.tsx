import React, { useEffect, useMemo, useState, useCallback } from 'react'
import siteService, { type Site } from '../services/site.service'
import externalServiceService, { type ExternalService } from '../services/externalService.service'
import pollingService, {
  type EndpointInstance,
  type PollingConfig,
  type EngineStatus,
  type OutputConfig,
} from '../services/polling.service'
import brokerService, { type BrokerStatus, type TopicTemplate } from '../services/broker.service'
import { discoveryEndpointCatalog } from '../lib/discoveryEndpointCatalog'
import type { BuilderEndpointMeta } from '../types/builder'
import type { DiscoveryProvider } from '../services/discovery.service'
import '../styles/PollingBuilder.css'

// Tipo para una instancia de endpoint en el UI (puede haber múltiples del mismo endpoint)
interface EndpointInstanceUI extends EndpointInstance {
  _uiId: string // ID único para React keys
  _isNew?: boolean // Si es una instancia recién creada
  interval_ms?: number // Intervalo específico en ms
}

const PollingBuilderPage: React.FC = () => {
  // Data
  const [sites, setSites] = useState<Site[]>([])
  const [connectors, setConnectors] = useState<ExternalService[]>([])
  const [pollingConfigs, setPollingConfigs] = useState<PollingConfig[]>([])
  const [engineStatus, setEngineStatus] = useState<EngineStatus | null>(null)

  // Brokers MQTT
  const [brokers, setBrokers] = useState<BrokerStatus[]>([])
  const [topicTemplates, setTopicTemplates] = useState<TopicTemplate[]>([])

  // Selección
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null)
  const [selectedProvider, setSelectedProvider] = useState<DiscoveryProvider | null>(null)

  // Instancias de endpoints configuradas
  const [endpointInstances, setEndpointInstances] = useState<EndpointInstanceUI[]>([])

  // Output Config (MQTT)
  const [outputEnabled, setOutputEnabled] = useState(false)
  const [outputBrokerId, setOutputBrokerId] = useState<string>('')
  const [outputTopicTemplate, setOutputTopicTemplate] = useState<string>('standard')

  // UI State
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [statusMessage, setStatusMessage] = useState<{
    type: 'success' | 'error' | 'info'
    text: string
  } | null>(null)

  // Modal de configuración
  const [configModalOpen, setConfigModalOpen] = useState(false)
  const [configModalEndpoint, setConfigModalEndpoint] = useState<BuilderEndpointMeta | null>(null)
  const [configModalInstance, setConfigModalInstance] = useState<EndpointInstanceUI | null>(null)
  const [configModalParams, setConfigModalParams] = useState<Record<string, string>>({})
  const [configModalIntervalSec, setConfigModalIntervalSec] = useState<number>(2) // Intervalo en segundos

  // Cargar datos iniciales
  useEffect(() => {
    loadInitialData()
  }, [])

  // Cargar instancias cuando cambia site/provider
  useEffect(() => {
    if (selectedSiteId && selectedProvider) {
      loadExistingConfig()
    } else {
      setEndpointInstances([])
      // Reset output config
      setOutputEnabled(false)
      setOutputBrokerId('')
      setOutputTopicTemplate('standard')
    }
  }, [selectedSiteId, selectedProvider])

  const loadInitialData = async () => {
    setLoading(true)
    try {
      const [sitesRes, connectorsRes, configsRes, statusRes, brokersRes, templatesRes] =
        await Promise.all([
          siteService.getAll(),
          externalServiceService.getAll(),
          pollingService.listConfigs(),
          pollingService.getStatus(),
          brokerService.listBrokers(),
          brokerService.getTopicTemplates(),
        ])

      if (sitesRes.success) setSites(sitesRes.data)
      if (connectorsRes.success) setConnectors(connectorsRes.data)
      if (configsRes.success) setPollingConfigs(configsRes.data || [])
      if (statusRes.success) setEngineStatus(statusRes.data)
      if (brokersRes.success) setBrokers(brokersRes.data.brokers || [])
      if (templatesRes.success) setTopicTemplates(templatesRes.data.templates || [])
    } catch (error) {
      console.error('Error cargando datos iniciales', error)
      setStatusMessage({ type: 'error', text: 'Error cargando datos' })
    } finally {
      setLoading(false)
    }
  }

  const loadExistingConfig = async () => {
    if (!selectedSiteId || !selectedProvider) return

    // Buscar si ya existe una config para este site/provider
    const existingConfig = pollingConfigs.find(
      (c) => c.site_id === selectedSiteId && c.provider === selectedProvider
    )

    if (existingConfig && existingConfig.endpoints) {
      // Convertir a formato UI (incluir interval_ms)
      const instances: EndpointInstanceUI[] = existingConfig.endpoints.map((ep, idx) => ({
        ...ep,
        interval_ms: ep.interval_ms || 2000, // Default si no existe
        _uiId: `${ep.instance_id}-${idx}`,
      }))
      setEndpointInstances(instances)

      // Cargar configuración de output si existe
      if (existingConfig.output) {
        setOutputEnabled(existingConfig.output.enabled)
        setOutputBrokerId(existingConfig.output.broker_id || '')
        setOutputTopicTemplate(existingConfig.output.topic_template || 'standard')
      } else {
        setOutputEnabled(false)
        setOutputBrokerId('')
        setOutputTopicTemplate('standard')
      }
    } else {
      setEndpointInstances([])
      setOutputEnabled(false)
      setOutputBrokerId('')
      setOutputTopicTemplate('standard')
    }
  }

  // Site seleccionado
  const selectedSite = useMemo(() => {
    return sites.find((s) => (s.id || (s as any)._id) === selectedSiteId) || null
  }, [sites, selectedSiteId])

  // Proveedores disponibles para el site seleccionado
  const availableProviders = useMemo(() => {
    if (!selectedSiteId) return []

    const siteConnectors = connectors.filter(
      (c) => c.site_id === selectedSiteId || c.site_id === (selectedSite as any)?._id
    )

    const providers: Array<{ id: DiscoveryProvider; label: string; service: ExternalService }> = []

    siteConnectors.forEach((conn) => {
      if (conn.service_type === 'innovex') {
        providers.push({ id: 'innovex', label: 'Innovex Dataweb', service: conn })
      } else if (conn.service_type === 'scaleaq') {
        providers.push({ id: 'scaleaq', label: 'ScaleAQ', service: conn })
      }
    })

    return providers
  }, [selectedSiteId, connectors, selectedSite])

  // Servicio externo seleccionado
  const selectedService = useMemo(() => {
    return availableProviders.find((p) => p.id === selectedProvider)?.service || null
  }, [availableProviders, selectedProvider])

  // Catálogo de endpoints para el provider seleccionado
  const endpointCatalog = useMemo(() => {
    if (!selectedProvider) return []
    return discoveryEndpointCatalog[selectedProvider] || []
  }, [selectedProvider])

  // Generar ID único para instancia
  const generateInstanceId = (endpointId: string) => {
    return `${endpointId}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
  }

  // Agregar endpoint al polling
  const handleAddEndpoint = (endpoint: BuilderEndpointMeta) => {
    // Si tiene parámetros requeridos, abrir modal
    if (endpoint.params && endpoint.params.length > 0) {
      setConfigModalEndpoint(endpoint)
      setConfigModalInstance(null)
      setConfigModalParams({})
      setConfigModalIntervalSec(2) // Default 2 segundos
      setConfigModalOpen(true)
    } else {
      // Agregar directamente
      const newInstance: EndpointInstanceUI = {
        _uiId: generateInstanceId(endpoint.id),
        instance_id: generateInstanceId(endpoint.id),
        endpoint_id: endpoint.id,
        label: endpoint.label,
        method: endpoint.method,
        path: endpoint.path,
        target_block: endpoint.targetBlock,
        params: {},
        enabled: true,
        interval_ms: 2000, // Default 2 segundos
        _isNew: true,
      }
      setEndpointInstances((prev) => [...prev, newInstance])
    }
  }

  // Duplicar instancia de endpoint
  const handleDuplicateInstance = (instance: EndpointInstanceUI) => {
    const endpoint = endpointCatalog.find((e) => e.id === instance.endpoint_id)
    if (endpoint) {
      setConfigModalEndpoint(endpoint)
      setConfigModalInstance(null)
      setConfigModalParams({ ...instance.params })
      setConfigModalIntervalSec(instance.interval_ms ? instance.interval_ms / 1000 : 2)
      setConfigModalOpen(true)
    }
  }

  // Editar configuración de instancia
  const handleEditInstance = (instance: EndpointInstanceUI) => {
    const endpoint = endpointCatalog.find((e) => e.id === instance.endpoint_id)
    if (endpoint) {
      setConfigModalEndpoint(endpoint)
      setConfigModalInstance(instance)
      setConfigModalParams({ ...instance.params })
      setConfigModalIntervalSec(instance.interval_ms ? instance.interval_ms / 1000 : 2)
      setConfigModalOpen(true)
    }
  }

  // Eliminar instancia
  const handleRemoveInstance = (uiId: string) => {
    setEndpointInstances((prev) => prev.filter((i) => i._uiId !== uiId))
  }

  // Toggle enabled/disabled
  const handleToggleEnabled = (uiId: string) => {
    setEndpointInstances((prev) =>
      prev.map((i) => (i._uiId === uiId ? { ...i, enabled: !i.enabled } : i))
    )
  }

  // Confirmar configuración del modal
  const handleConfirmConfig = () => {
    if (!configModalEndpoint) return

    // Convertir segundos a milisegundos, mínimo 1 segundo
    const intervalMs = Math.max(1, configModalIntervalSec) * 1000

    if (configModalInstance) {
      // Editando instancia existente
      setEndpointInstances((prev) =>
        prev.map((i) =>
          i._uiId === configModalInstance._uiId
            ? {
                ...i,
                params: { ...configModalParams },
                label: buildInstanceLabel(configModalEndpoint, configModalParams),
                interval_ms: intervalMs,
              }
            : i
        )
      )
    } else {
      // Nueva instancia
      const newInstance: EndpointInstanceUI = {
        _uiId: generateInstanceId(configModalEndpoint.id),
        instance_id: generateInstanceId(configModalEndpoint.id),
        endpoint_id: configModalEndpoint.id,
        label: buildInstanceLabel(configModalEndpoint, configModalParams),
        method: configModalEndpoint.method,
        path: configModalEndpoint.path,
        target_block: configModalEndpoint.targetBlock,
        params: { ...configModalParams },
        enabled: true,
        interval_ms: intervalMs,
        _isNew: true,
      }
      setEndpointInstances((prev) => [...prev, newInstance])
    }

    setConfigModalOpen(false)
    setConfigModalEndpoint(null)
    setConfigModalInstance(null)
    setConfigModalParams({})
    setConfigModalIntervalSec(2)
  }

  // Construir label descriptivo para la instancia
  const buildInstanceLabel = (endpoint: BuilderEndpointMeta, params: Record<string, string>) => {
    const paramValues = Object.values(params).filter(Boolean)
    if (paramValues.length > 0) {
      return `${endpoint.label} (${paramValues.slice(0, 2).join(', ')})`
    }
    return endpoint.label
  }

  // Guardar y reiniciar polling
  const handleSaveAndRestart = async () => {
    if (!selectedSite || !selectedProvider || !selectedService) {
      setStatusMessage({ type: 'error', text: 'Selecciona un centro y proveedor' })
      return
    }

    setSaving(true)
    setStatusMessage(null)

    try {
      // Primero detener el polling existente para este site/provider
      await pollingService.stopPolling({
        site_id: selectedSiteId!,
        provider: selectedProvider,
      })

      // Si no hay endpoints, solo detener (eliminar configuración)
      if (endpointInstances.length === 0) {
        setStatusMessage({ type: 'success', text: 'Polling detenido - todos los endpoints eliminados' })
        loadInitialData()
        setSaving(false)
        return
      }

      // Preparar endpoints para el request
      const endpoints: EndpointInstance[] = endpointInstances.map((inst) => ({
        instance_id: inst.instance_id,
        endpoint_id: inst.endpoint_id,
        label: inst.label,
        method: inst.method,
        path: inst.path,
        target_block: inst.target_block,
        params: inst.params,
        enabled: inst.enabled,
        interval_ms: inst.interval_ms || 2000, // Incluir intervalo específico
      }))

      // Preparar output config si está habilitado
      const outputConfig: OutputConfig | undefined =
        outputEnabled && outputBrokerId
          ? {
              broker_id: outputBrokerId,
              topic_template: outputTopicTemplate,
              enabled: true,
            }
          : undefined

      // Iniciar nuevo polling
      const response = await pollingService.startPolling({
        provider: selectedProvider,
        site_id: selectedSiteId!,
        site_code: selectedSite.code,
        site_name: selectedSite.name,
        tenant_id: selectedSite.tenant_id || '',
        tenant_code: selectedSite.tenant_code || '',
        service_id: selectedService._id || selectedService.id || '',
        endpoints,
        interval_ms: 2000,
        auto_start: true,
        output: outputConfig,
      })

      if (response.success) {
        setStatusMessage({ type: 'success', text: 'Polling guardado y reiniciado correctamente' })
        // Recargar configs y status
        loadInitialData()
      } else {
        setStatusMessage({
          type: 'error',
          text: response.message || 'Error guardando configuración',
        })
      }
    } catch (error) {
      console.error('Error guardando polling', error)
      setStatusMessage({ type: 'error', text: 'Error al guardar la configuración' })
    } finally {
      setSaving(false)
    }
  }

  // Contar instancias de un endpoint
  const countEndpointInstances = useCallback(
    (endpointId: string) => {
      return endpointInstances.filter((i) => i.endpoint_id === endpointId).length
    },
    [endpointInstances]
  )

  // Agrupar endpoints por categoría
  const endpointsByCategory = useMemo(() => {
    const grouped: Record<string, BuilderEndpointMeta[]> = {}
    endpointCatalog.forEach((ep) => {
      if (!grouped[ep.category]) {
        grouped[ep.category] = []
      }
      grouped[ep.category].push(ep)
    })
    return grouped
  }, [endpointCatalog])

  return (
    <div className="polling-builder-page">
      {/* Header */}
      <header className="polling-builder-header">
        <div>
          <h1>Polling Builder</h1>
          <p>Configura los endpoints a consultar para cada centro y proveedor</p>
        </div>
        <div className="polling-builder-actions">
          {engineStatus && (
            <span className={`engine-status ${engineStatus.status}`}>
              {engineStatus.status === 'running' ? '🟢' : '🔴'} Engine: {engineStatus.status}
              {engineStatus.active_workers > 0 && ` (${engineStatus.active_workers} workers)`}
            </span>
          )}
          <button
            className="btn-primary"
            onClick={handleSaveAndRestart}
            disabled={saving || !selectedSiteId || !selectedProvider}
          >
            {saving ? 'Guardando...' : '💾 Guardar y Reiniciar Polling'}
          </button>
        </div>
      </header>

      {/* Status Message */}
      {statusMessage && (
        <div className={`status-message ${statusMessage.type}`}>{statusMessage.text}</div>
      )}

      {/* Output Config Section - Solo visible cuando hay site y provider seleccionados */}
      {selectedSiteId && selectedProvider && (
        <div className="output-config-section">
          <div className="output-config-header">
            <label className="output-toggle">
              <input
                type="checkbox"
                checked={outputEnabled}
                onChange={(e) => setOutputEnabled(e.target.checked)}
              />
              <span className="toggle-slider"></span>
              <span className="toggle-label">📡 Publicar datos a Broker MQTT</span>
            </label>
          </div>

          {outputEnabled && (
            <div className="output-config-fields">
              <div className="output-field">
                <label>Broker MQTT</label>
                <select value={outputBrokerId} onChange={(e) => setOutputBrokerId(e.target.value)}>
                  <option value="">-- Selecciona un broker --</option>
                  {brokers
                    .filter((b) => b.connected)
                    .map((broker) => (
                      <option key={broker.id} value={broker.id}>
                        {broker.name} ({broker.url}) {broker.connected ? '🟢' : '🔴'}
                      </option>
                    ))}
                  {brokers
                    .filter((b) => !b.connected)
                    .map((broker) => (
                      <option key={broker.id} value={broker.id} disabled>
                        {broker.name} ({broker.url}) 🔴 Desconectado
                      </option>
                    ))}
                </select>
                {brokers.length === 0 && (
                  <small className="field-hint">
                    No hay brokers configurados. <a href="/dashboard/brokers">Agregar broker</a>
                  </small>
                )}
              </div>

              <div className="output-field">
                <label>Template de Topic</label>
                <select
                  value={outputTopicTemplate}
                  onChange={(e) => setOutputTopicTemplate(e.target.value)}
                >
                  {topicTemplates.map((template) => (
                    <option key={template.id} value={template.id}>
                      {template.name} - {template.pattern}
                    </option>
                  ))}
                </select>
                <small className="field-hint">
                  Variables: {'{provider}'}, {'{site}'}, {'{tenant}'}, {'{endpoint}'},{' '}
                  {'{data_type}'}
                </small>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Main Content - 3 Columns */}
      <div className="polling-builder-content">
        {/* Column 1: Sites */}
        <div className="polling-column sites-column">
          <h3>📍 Centros de Cultivo</h3>
          <div className="column-list">
            {loading ? (
              <div className="column-loading">Cargando...</div>
            ) : sites.length === 0 ? (
              <div className="column-empty">No hay centros registrados</div>
            ) : (
              sites.map((site) => {
                const siteId = site.id || (site as any)._id
                const isSelected = selectedSiteId === siteId
                const hasConfig = pollingConfigs.some((c) => c.site_id === siteId)

                return (
                  <button
                    key={siteId}
                    className={`column-item ${isSelected ? 'selected' : ''} ${
                      hasConfig ? 'has-config' : ''
                    }`}
                    onClick={() => {
                      setSelectedSiteId(siteId)
                      setSelectedProvider(null)
                    }}
                  >
                    <span className="item-name">{site.name}</span>
                    <span className="item-code">{site.code}</span>
                    {hasConfig && <span className="item-badge">⚡</span>}
                  </button>
                )
              })
            )}
          </div>
        </div>

        {/* Column 2: Providers */}
        <div className="polling-column providers-column">
          <h3>🔌 Proveedores</h3>
          <div className="column-list">
            {!selectedSiteId ? (
              <div className="column-empty">Selecciona un centro</div>
            ) : availableProviders.length === 0 ? (
              <div className="column-empty">
                No hay proveedores configurados para este centro.
                <br />
                <small>Agrega un servicio externo en "Servicios Externos"</small>
              </div>
            ) : (
              availableProviders.map((provider) => {
                const isSelected = selectedProvider === provider.id
                const hasConfig = pollingConfigs.some(
                  (c) => c.site_id === selectedSiteId && c.provider === provider.id
                )

                return (
                  <button
                    key={provider.id}
                    className={`column-item ${isSelected ? 'selected' : ''} ${
                      hasConfig ? 'has-config' : ''
                    }`}
                    onClick={() => setSelectedProvider(provider.id)}
                  >
                    <span className="item-name">{provider.label}</span>
                    <span className="item-code">{provider.service.name}</span>
                    {hasConfig && <span className="item-badge">⚡</span>}
                  </button>
                )
              })
            )}
          </div>
        </div>

        {/* Column 3: Endpoints */}
        <div className="polling-column endpoints-column">
          <h3>📡 Endpoints</h3>
          <div className="column-content">
            {!selectedProvider ? (
              <div className="column-empty">Selecciona un proveedor</div>
            ) : (
              <>
                {/* Endpoints activos */}
                {endpointInstances.length > 0 && (
                  <div className="endpoints-active-section">
                    <h4>Endpoints Activos ({endpointInstances.length})</h4>
                    <div className="endpoints-active-list">
                      {endpointInstances.map((instance) => {
                        const endpoint = endpointCatalog.find((e) => e.id === instance.endpoint_id)
                        const hasParams = endpoint?.params && endpoint.params.length > 0

                        return (
                          <div
                            key={instance._uiId}
                            className={`endpoint-instance ${
                              instance.enabled ? 'enabled' : 'disabled'
                            }`}
                          >
                            <div className="endpoint-instance-main">
                              <label className="endpoint-checkbox">
                                <input
                                  type="checkbox"
                                  checked={instance.enabled}
                                  onChange={() => handleToggleEnabled(instance._uiId)}
                                />
                                <span className="checkmark"></span>
                              </label>
                              <div className="endpoint-instance-info">
                                <span className="endpoint-label">{instance.label}</span>
                                <span className="endpoint-path">
                                  <code>{instance.method}</code> {instance.path}
                                </span>
                                {instance.params && Object.keys(instance.params).length > 0 && (
                                  <span className="endpoint-params">
                                    {Object.entries(instance.params).map(([k, v]) => (
                                      <span key={k} className="param-tag">
                                        {k}: {v}
                                      </span>
                                    ))}
                                  </span>
                                )}
                                <span className="endpoint-interval">
                                  ⏱️ {instance.interval_ms ? instance.interval_ms / 1000 : 2}s
                                </span>
                              </div>
                            </div>
                            <div className="endpoint-instance-actions">
                              {hasParams && (
                                <>
                                  <button
                                    className="btn-icon"
                                    onClick={() => handleEditInstance(instance)}
                                    title="Editar configuración"
                                  >
                                    ✏️
                                  </button>
                                  <button
                                    className="btn-icon"
                                    onClick={() => handleDuplicateInstance(instance)}
                                    title="Duplicar con nueva configuración"
                                  >
                                    📋
                                  </button>
                                </>
                              )}
                              <button
                                className="btn-icon danger"
                                onClick={() => handleRemoveInstance(instance._uiId)}
                                title="Eliminar"
                              >
                                🗑️
                              </button>
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )}

                {/* Catálogo de endpoints disponibles */}
                <div className="endpoints-catalog-section">
                  <h4>Endpoints Disponibles</h4>
                  {Object.entries(endpointsByCategory).map(([category, endpoints]) => (
                    <div key={category} className="endpoint-category">
                      <h5>{category}</h5>
                      <div className="endpoint-category-list">
                        {endpoints.map((endpoint) => {
                          const instanceCount = countEndpointInstances(endpoint.id)
                          const hasParams = endpoint.params && endpoint.params.length > 0

                          return (
                            <div key={endpoint.id} className="endpoint-catalog-item">
                              <div className="endpoint-catalog-info">
                                <span className="endpoint-label">
                                  {endpoint.label}
                                  {instanceCount > 0 && (
                                    <span className="instance-count">({instanceCount})</span>
                                  )}
                                </span>
                                <span className="endpoint-description">{endpoint.description}</span>
                                <span className="endpoint-path">
                                  <code>{endpoint.method}</code> {endpoint.path}
                                </span>
                                {hasParams && (
                                  <span className="endpoint-has-params">
                                    ⚙️ Requiere: {endpoint.params?.map((p) => p.label).join(', ')}
                                  </span>
                                )}
                              </div>
                              <button
                                className="btn-add-endpoint"
                                onClick={() => handleAddEndpoint(endpoint)}
                                title={hasParams ? 'Agregar con configuración' : 'Agregar endpoint'}
                              >
                                + Agregar
                              </button>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Modal de configuración de parámetros */}
      {configModalOpen && configModalEndpoint && (
        <div className="config-modal-overlay" onClick={() => setConfigModalOpen(false)}>
          <div className="config-modal" onClick={(e) => e.stopPropagation()}>
            <div className="config-modal-header">
              <h3>
                {configModalInstance ? 'Editar' : 'Configurar'}: {configModalEndpoint.label}
              </h3>
              <button className="btn-close" onClick={() => setConfigModalOpen(false)}>
                ✕
              </button>
            </div>
            <div className="config-modal-body">
              <p className="config-modal-description">{configModalEndpoint.description}</p>
              <div className="config-modal-path">
                <code>{configModalEndpoint.method}</code> {configModalEndpoint.path}
              </div>

              <div className="config-modal-params">
                {configModalEndpoint.params?.map((param) => (
                  <div key={param.name} className="config-param">
                    <label>
                      {param.label}
                      {param.required && <span className="required">*</span>}
                    </label>
                    <input
                      type="text"
                      value={configModalParams[param.name] || ''}
                      onChange={(e) =>
                        setConfigModalParams((prev) => ({
                          ...prev,
                          [param.name]: e.target.value,
                        }))
                      }
                      placeholder={param.placeholder}
                    />
                    {param.helperText && <small>{param.helperText}</small>}
                  </div>
                ))}

                {/* Campo de intervalo de polling */}
                <div className="config-param config-interval">
                  <label>
                    Intervalo de Polling <span className="required">*</span>
                  </label>
                  <div className="interval-input-wrapper">
                    <input
                      type="number"
                      min="1"
                      max="3600"
                      value={configModalIntervalSec}
                      onChange={(e) =>
                        setConfigModalIntervalSec(Math.max(1, parseInt(e.target.value) || 1))
                      }
                    />
                    <span className="interval-unit">segundos</span>
                  </div>
                  <small>
                    Tiempo entre consultas (mínimo 1s). Ej: 2 = cada 2 segundos, 60 = cada minuto
                  </small>
                </div>
              </div>
            </div>
            <div className="config-modal-footer">
              <button className="btn-secondary" onClick={() => setConfigModalOpen(false)}>
                Cancelar
              </button>
              <button
                className="btn-primary"
                onClick={handleConfirmConfig}
                disabled={configModalEndpoint.params?.some(
                  (p) => p.required && !configModalParams[p.name]
                )}
              >
                {configModalInstance ? 'Guardar cambios' : 'Agregar endpoint'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default PollingBuilderPage
