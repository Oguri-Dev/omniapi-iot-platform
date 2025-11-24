import React, { useState, useEffect, useMemo } from 'react'
import connectorService, {
  type ExternalService as LegacyConnector,
  type CreateServiceDTO,
} from '../services/connector.service'
import externalServiceService, {
  type ExternalService as IntegrationService,
} from '../services/externalService.service'
import siteService, { type Site } from '../services/site.service'
import discoveryService, {
  type DiscoveryResult,
  type DiscoveryProvider,
  type DiscoveryCache,
} from '../services/discovery.service'
import EndpointBuilder, { type EndpointSelectionState } from '../components/EndpointBuilder'
import { discoveryEndpointCatalog } from '../lib/discoveryEndpointCatalog'
import type { BuilderViewMode } from '../types/builder'
import '../styles/Services.css'

const DISCOVERY_PROVIDERS: Array<{
  id: DiscoveryProvider
  label: string
  description: string
}> = [
  {
    id: 'scaleaq',
    label: 'ScaleAQ',
    description: 'Discovery basado en Time Series y Feeding del archivo demo.',
  },
  {
    id: 'innovex',
    label: 'Innovex',
    description: 'Discovery de Innovex Dataweb (all_monitors, detail, lecturas y errores).',
  },
]

const Services: React.FC = () => {
  const [services, setServices] = useState<LegacyConnector[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingService, setEditingService] = useState<LegacyConnector | null>(null)
  const [formData, setFormData] = useState<CreateServiceDTO>({
    name: '',
    type: 'rest',
    url: '',
    auth_type: 'none',
  })
  const [sites, setSites] = useState<Site[]>([])
  const [integrationServices, setIntegrationServices] = useState<IntegrationService[]>([])
  const [treeLoading, setTreeLoading] = useState(true)
  const [expandedSites, setExpandedSites] = useState<Record<string, boolean>>({})
  const [expandedConnectors, setExpandedConnectors] = useState<Record<string, boolean>>({})
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null)
  const [discoveryCache, setDiscoveryCache] = useState<DiscoveryCache>({})
  const [discoveryLoading, setDiscoveryLoading] = useState(false)
  const [discoveryError, setDiscoveryError] = useState<string | null>(null)
  const [discoveryProvider, setDiscoveryProvider] = useState<DiscoveryProvider>('scaleaq')
  // UI toggle: permite alternar entre la vista clásica de discovery y el nuevo builder
  const [viewMode, setViewMode] = useState<BuilderViewMode>('discovery')
  // Estado local con los endpoints marcados + parámetros personalizados
  const [builderSelections, setBuilderSelections] = useState<
    Record<string, EndpointSelectionState>
  >({})

  const SCALEAQ_DATA_GROUPS = useMemo(
    () => [
      {
        title: 'Meta',
        description: 'Información estática del centro y la compañía',
        items: [
          { label: 'Company Info', endpoint: 'GET /meta/company?include=all' },
          { label: 'Site Info', endpoint: 'GET /meta/sites/{siteId}?include=all' },
        ],
      },
      {
        title: 'Time Series',
        description: 'Datos crudos capturados por ScaleAQ',
        items: [
          { label: 'Retrieve', endpoint: 'POST /time-series/retrieve' },
          { label: 'Available Data Types', endpoint: 'POST /time-series/retrieve/data-types' },
        ],
      },
      {
        title: 'Aggregates',
        description: 'Resúmenes por unidad y silos (feeding, biomasa, oxígeno)',
        items: [
          { label: 'Units Aggregate', endpoint: 'POST /time-series/retrieve/units/aggregate' },
          { label: 'Silos Aggregate', endpoint: 'POST /time-series/retrieve/silos/aggregate' },
        ],
      },
    ],
    []
  )

  const INNOVEX_DATA_GROUPS = useMemo(
    () => [
      {
        title: 'Catálogo',
        description: 'Monitores y sensores registrados en Innovex Dataweb.',
        items: [
          { label: 'All Monitors', endpoint: 'GET /api_dataweb/all_monitors/?active=all' },
          { label: 'Monitor Detail', endpoint: 'GET /api_dataweb/monitor_detail/?monitor_id=' },
        ],
      },
      {
        title: 'Lecturas',
        description: 'Últimos datos por sensor y medición.',
        items: [
          {
            label: 'Monitor Sensor Last Data',
            endpoint: 'GET /api_dataweb/monitor_sensor_last_data/?id={monitor_id}&medition=oxygen',
          },
          { label: 'Get Last Data', endpoint: 'GET /api_dataweb/get_last_data/?sensor_id=' },
        ],
      },
      {
        title: 'Históricos',
        description: 'Consultas por rango de fechas (hasta 30 días).',
        items: [
          { label: 'Get Data Range', endpoint: 'GET /api_dataweb/get_data_range/' },
          {
            label: 'Monitor Sensor Time Data',
            endpoint: 'GET /api_dataweb/monitor_sensor_time_data/',
          },
        ],
      },
    ],
    []
  )

  useEffect(() => {
    loadServices()
    loadConnectorTree()
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

  const loadConnectorTree = async () => {
    try {
      setTreeLoading(true)
      const [sitesResponse, integrationsResponse] = await Promise.all([
        siteService.getAll(),
        externalServiceService.getAll(),
      ])

      if (sitesResponse.success) {
        setSites(sitesResponse.data)
      } else {
        setSites([])
      }

      if (integrationsResponse.success) {
        setIntegrationServices(integrationsResponse.data)
      } else {
        setIntegrationServices([])
      }
    } catch (error) {
      console.error('Error loading connector tree:', error)
      setSites([])
      setIntegrationServices([])
    } finally {
      setTreeLoading(false)
    }
  }

  const getSiteKey = (site: Site) => site.id || (site as any)._id || ''

  const siteTree = useMemo(() => {
    if (!sites || !integrationServices) return []

    return sites.map((site) => {
      const siteKey = getSiteKey(site)
      const relatedConnectors = integrationServices.filter(
        (service) => service.site_id === siteKey || service.site_id === (site as any)._id
      )

      return {
        site,
        siteKey,
        connectors: relatedConnectors,
      }
    })
  }, [sites, integrationServices])

  useEffect(() => {
    if (!selectedSiteId && siteTree.length > 0) {
      const preferred = siteTree.find((entry) => entry.connectors.length > 0) ?? siteTree[0]
      if (preferred) {
        setSelectedSiteId(preferred.siteKey)
        setExpandedSites((prev) => ({ ...prev, [preferred.siteKey]: true }))
      }
    }
  }, [siteTree, selectedSiteId])

  const selectedSiteNode = useMemo(() => {
    return siteTree.find((entry) => entry.siteKey === selectedSiteId) || null
  }, [siteTree, selectedSiteId])

  const makeDiscoveryKey = (siteKey: string, provider: DiscoveryProvider) =>
    `${siteKey}:${provider}`

  const activeDiscoveryKey = selectedSiteId
    ? makeDiscoveryKey(selectedSiteId, discoveryProvider)
    : null

  const activeDiscovery: DiscoveryResult | null = activeDiscoveryKey
    ? discoveryCache[activeDiscoveryKey] ?? null
    : null

  const providerInfo = useMemo(
    () => DISCOVERY_PROVIDERS.find((item) => item.id === discoveryProvider),
    [discoveryProvider]
  )

  const providerConnectors = useMemo(() => {
    if (!selectedSiteNode) return []
    return selectedSiteNode.connectors.filter((conn) => conn.service_type === discoveryProvider)
  }, [selectedSiteNode, discoveryProvider])

  const builderCatalog = useMemo(() => {
    return discoveryEndpointCatalog[discoveryProvider] || []
  }, [discoveryProvider])

  const builderSelectedCount = useMemo(() => {
    return Object.values(builderSelections).filter((item) => item.enabled).length
  }, [builderSelections])

  // Al hacer toggle agregamos/removemos el endpoint seleccionado en el builder
  const handleBuilderToggle = (endpointId: string, enabled: boolean) => {
    setBuilderSelections((prev) => {
      if (!enabled) {
        const { [endpointId]: _, ...rest } = prev
        return rest
      }
      return {
        ...prev,
        [endpointId]: {
          enabled: true,
          params: prev[endpointId]?.params || {},
        },
      }
    })
  }

  // Persistimos los parámetros escritos por el usuario para cada endpoint
  const handleBuilderParamChange = (endpointId: string, paramName: string, value: string) => {
    setBuilderSelections((prev) => ({
      ...prev,
      [endpointId]: {
        enabled: true,
        params: {
          ...(prev[endpointId]?.params || {}),
          [paramName]: value,
        },
      },
    }))
  }

  const handleClearBuilder = () => {
    setBuilderSelections({})
  }

  const toggleSite = (siteKey: string) => {
    setExpandedSites((prev) => ({
      ...prev,
      [siteKey]: !prev[siteKey],
    }))
  }

  const toggleConnector = (connectorKey: string) => {
    setExpandedConnectors((prev) => ({
      ...prev,
      [connectorKey]: !prev[connectorKey],
    }))
  }

  const handleFocusSite = (siteKey: string) => {
    setSelectedSiteId(siteKey)
    setExpandedSites((prev) => ({
      ...prev,
      [siteKey]: true,
    }))
  }

  const handleSelectProvider = (provider: DiscoveryProvider) => {
    setDiscoveryProvider(provider)
    setDiscoveryError(null)
    setBuilderSelections({})
  }

  const handleRunDiscovery = async (provider: DiscoveryProvider) => {
    if (!selectedSiteId) return
    const node = siteTree.find((entry) => entry.siteKey === selectedSiteId)
    if (!node) return

    const connectorsForProvider = node.connectors.filter(
      (connector) => connector.service_type === provider
    )

    if (connectorsForProvider.length === 0) {
      setDiscoveryError('Agrega un conector activo de este proveedor para ejecutar el discovery.')
      return
    }

    setDiscoveryError(null)
    setDiscoveryLoading(true)

    try {
      let result: DiscoveryResult
      if (provider === 'innovex') {
        result = await discoveryService.runInnovexDiscovery(node.site, connectorsForProvider)
      } else {
        result = await discoveryService.runScaleAQDiscovery(node.site, connectorsForProvider)
      }

      const cacheKey = makeDiscoveryKey(selectedSiteId, provider)
      setDiscoveryCache((prev: DiscoveryCache) => ({
        ...prev,
        [cacheKey]: result,
      }))
    } catch (error) {
      console.error('Error running discovery:', error)
      setDiscoveryError(
        provider === 'innovex'
          ? 'No se pudo ejecutar el discovery Innovex. Verifica monitor_id y credenciales.'
          : 'No se pudo ejecutar el discovery ScaleAQ. Revisa el conector o vuelve a intentar.'
      )
    } finally {
      setDiscoveryLoading(false)
    }
  }

  useEffect(() => {
    setBuilderSelections({})
  }, [selectedSiteId])

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

  const handleEdit = (service: LegacyConnector) => {
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
      <section className="connector-tree">
        <div className="connector-tree__header">
          <div>
            <h2>🌐 Conectores por Centro</h2>
            <p className="connector-tree__subtitle">
              Visualiza los datos disponibles agrupados por centro de cultivo y tipo de integración.
            </p>
          </div>
          <button className="btn-secondary" onClick={loadConnectorTree} disabled={treeLoading}>
            {treeLoading ? 'Actualizando...' : '↻ Actualizar tree'}
          </button>
        </div>

        <div className="sites-strip">
          <div className="sites-strip__header">
            <div>
              <h3>Centros conectados</h3>
              <p>Selecciona un centro para ejecutar el discovery y revisar sus endpoints.</p>
            </div>
            <span className="sites-strip__count">{siteTree.length} centros</span>
          </div>

          <div className="sites-strip__list">
            {siteTree.map(({ site, siteKey, connectors }) => {
              const providerCounts = connectors.reduce<Record<string, number>>((acc, connector) => {
                acc[connector.service_type] = (acc[connector.service_type] || 0) + 1
                return acc
              }, {})

              const providerBadges = [
                { label: 'ScaleAQ', count: providerCounts.scaleaq || 0 },
                { label: 'Innovex', count: providerCounts.innovex || 0 },
              ].filter((badge) => badge.count > 0)

              return (
                <button
                  key={siteKey}
                  className={`site-pill ${selectedSiteId === siteKey ? 'is-active' : ''}`}
                  onClick={() => handleFocusSite(siteKey)}
                >
                  <div className="site-pill__title">{site.name}</div>
                  <div className="site-pill__meta">
                    {site.tenant_code} · {site.code}
                  </div>
                  <div className="site-pill__stats">
                    <span>{connectors.length} conectores</span>
                    {providerBadges.length > 0 ? (
                      providerBadges.map((badge) => (
                        <span key={badge.label}>
                          {badge.count} {badge.label}
                        </span>
                      ))
                    ) : (
                      <span>Sin proveedores</span>
                    )}
                  </div>
                </button>
              )
            })}
          </div>
        </div>

        <div className="discovery-panel">
          <div className="discovery-panel__header">
            <div>
              <h3>Discovery {providerInfo?.label}</h3>
              <p>{providerInfo?.description}</p>
              {selectedSiteNode && (
                <small>
                  Centro seleccionado: <strong>{selectedSiteNode.site.name}</strong> · ID{' '}
                  {selectedSiteNode.site.id || selectedSiteNode.site.code}
                </small>
              )}
              <div className="discovery-provider-tabs">
                {DISCOVERY_PROVIDERS.map((option) => (
                  <button
                    key={option.id}
                    className={`provider-tab ${discoveryProvider === option.id ? 'is-active' : ''}`}
                    onClick={() => handleSelectProvider(option.id)}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
              <div className="discovery-view-tabs">
                {(
                  [
                    { id: 'discovery', label: 'Vista discovery' },
                    { id: 'builder', label: 'Menu builder' },
                  ] as Array<{ id: BuilderViewMode; label: string }>
                ).map((view) => (
                  <button
                    key={view.id}
                    className={`view-tab ${viewMode === view.id ? 'is-active' : ''}`}
                    onClick={() => setViewMode(view.id)}
                  >
                    {view.label}
                  </button>
                ))}
              </div>
            </div>
            {viewMode === 'discovery' ? (
              <button
                className="btn-primary"
                onClick={() => handleRunDiscovery(discoveryProvider)}
                disabled={!selectedSiteId || discoveryLoading || providerConnectors.length === 0}
              >
                {discoveryLoading ? 'Descubriendo...' : '▶ Ejecutar discovery'}
              </button>
            ) : (
              <button
                className="btn-secondary"
                onClick={handleClearBuilder}
                disabled={builderSelectedCount === 0}
              >
                Limpiar selección
              </button>
            )}
          </div>

          {providerConnectors.length === 0 && selectedSiteNode && !discoveryLoading && (
            <div className="discovery-panel__hint">
              No hay conectores {providerInfo?.label} configurados para este centro. Agrega uno en
              “Servicios Externos” para habilitar el discovery.
            </div>
          )}

          {viewMode === 'discovery' && discoveryError && (
            <div className="discovery-panel__error">{discoveryError}</div>
          )}

          {viewMode === 'discovery' ? (
            discoveryLoading ? (
              <div className="tree-loading">Consultando endpoints...</div>
            ) : activeDiscovery ? (
              <div className="discovery-output">
                <div className="discovery-summary">
                  <div>
                    <p className="muted">Última ejecución</p>
                    <strong>{new Date(activeDiscovery.generatedAt).toLocaleString()}</strong>
                  </div>
                  <div>
                    <p className="muted">Proveedor</p>
                    <strong>
                      {activeDiscovery.provider === 'innovex' ? 'Innovex' : 'ScaleAQ'}
                    </strong>
                  </div>
                  <div>
                    <p className="muted">Ventana cubierta</p>
                    <strong>{activeDiscovery.summary.range}</strong>
                  </div>
                  <div>
                    <p className="muted">Series detectadas</p>
                    <strong>{activeDiscovery.summary.timeseriesCount}</strong>
                  </div>
                  <div>
                    <p className="muted">KPIs</p>
                    <strong>{activeDiscovery.summary.metricsAvailable}</strong>
                  </div>
                </div>

                {activeDiscovery.headersUsed && (
                  <div className="discovery-headers">
                    {Object.entries(activeDiscovery.headersUsed).map(([header, value]) => (
                      <span key={header}>
                        {header}: <code>{value}</code>
                      </span>
                    ))}
                  </div>
                )}

                <div className="discovery-groups">
                  {activeDiscovery.groups.map((group) => (
                    <div key={group.id} className="discovery-group">
                      <div className="discovery-group__header">
                        <div>
                          <h4>{group.title}</h4>
                          <p>{group.description}</p>
                        </div>
                        <span>{group.endpoints.length} endpoints</span>
                      </div>
                      <div className="discovery-group__body">
                        {group.endpoints.map((endpoint) => {
                          const dataset = endpoint.dataset
                          return (
                            <div key={endpoint.path} className="discovery-endpoint">
                              <div className="discovery-endpoint__meta">
                                <span
                                  className={`method-badge method-${endpoint.method.toLowerCase()}`}
                                >
                                  {endpoint.method}
                                </span>
                                <code>{endpoint.path}</code>
                                <span
                                  className={`availability availability-${endpoint.availability}`}
                                >
                                  {endpoint.availability === 'ready'
                                    ? 'Disponible'
                                    : endpoint.availability === 'partial'
                                    ? 'Parcial'
                                    : 'Error'}
                                </span>
                              </div>
                              <p>{endpoint.description}</p>
                              {dataset && (
                                <div className="discovery-dataset">
                                  <div className="discovery-dataset__title">{dataset.title}</div>
                                  {dataset.summary && (
                                    <p className="discovery-dataset__summary">{dataset.summary}</p>
                                  )}
                                  {dataset.metrics && (
                                    <div className="discovery-dataset__metrics">
                                      {dataset.metrics.map((metric) => (
                                        <div
                                          key={metric.label}
                                          className={`dataset-metric ${
                                            metric.accent ? `is-${metric.accent}` : ''
                                          }`}
                                        >
                                          <span>{metric.label}</span>
                                          <strong>{metric.value}</strong>
                                        </div>
                                      ))}
                                    </div>
                                  )}
                                  {dataset.highlights && (
                                    <ul className="discovery-dataset__highlights">
                                      {dataset.highlights.map((highlight) => (
                                        <li key={highlight}>{highlight}</li>
                                      ))}
                                    </ul>
                                  )}
                                  {dataset.table &&
                                    (() => {
                                      const table = dataset.table
                                      return (
                                        <div className="discovery-dataset__table">
                                          <div className="table-header">
                                            {table.columns.map((column) => (
                                              <span key={column}>{column}</span>
                                            ))}
                                          </div>
                                          {table.rows.map((row, index) => (
                                            <div key={index} className="table-row">
                                              {table.columns.map((column) => (
                                                <span key={`${column}-${index}`}>
                                                  {row[column] as string}
                                                </span>
                                              ))}
                                            </div>
                                          ))}
                                        </div>
                                      )
                                    })()}
                                </div>
                              )}
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="tree-empty">
                Selecciona un centro y ejecuta el discovery para desplegar los endpoints del archivo
                demo.
              </div>
            )
          ) : selectedSiteNode ? (
            <EndpointBuilder
              provider={discoveryProvider}
              site={selectedSiteNode.site}
              endpoints={builderCatalog}
              selections={builderSelections}
              onToggle={handleBuilderToggle}
              onParamChange={handleBuilderParamChange}
            />
          ) : (
            <div className="tree-empty">
              Selecciona un centro para comenzar a construir el payload de endpoints.
            </div>
          )}
        </div>

        {treeLoading ? (
          <div className="tree-loading">Construyendo árbol de conectores...</div>
        ) : siteTree.length === 0 ? (
          <div className="tree-empty">No hay centros con conectores configurados.</div>
        ) : (
          <div className="tree-list">
            {siteTree.map(({ site, siteKey, connectors }) => (
              <div
                key={siteKey}
                className={`tree-site ${selectedSiteId === siteKey ? 'tree-site--active' : ''}`}
              >
                <button className="tree-site__toggle" onClick={() => toggleSite(siteKey)}>
                  <span className="tree-arrow">{expandedSites[siteKey] ? '▾' : '▸'}</span>
                  <div>
                    <strong>{site.name}</strong>
                    <small>
                      {site.tenant_code} · Código interno: {site.code}
                    </small>
                  </div>
                </button>

                {expandedSites[siteKey] && (
                  <div className="tree-site__content">
                    {connectors.length === 0 ? (
                      <p className="tree-placeholder">
                        Este centro aún no tiene servicios vinculados.
                      </p>
                    ) : (
                      connectors.map((connector) => {
                        const connectorKey = `${siteKey}-${connector._id || connector.id}`
                        const isScaleAQ = connector.service_type === 'scaleaq'
                        const isInnovex = connector.service_type === 'innovex'
                        const scaleaqSiteId =
                          (connector.config?.scaleaq_site_id as string) || 'No definido'
                        const scaleVersion =
                          (connector.config?.scale_version as string) || '2025-01-01'
                        const acceptHeader =
                          (connector.config?.accept_header as string) || 'application/json'
                        const monitorId =
                          (connector.config?.monitor_id as string) || 'No configurado'
                        const preferredMedition = (connector.config?.medition as string) || 'oxygen'

                        return (
                          <div key={connectorKey} className="tree-connector">
                            <button
                              className="tree-connector__toggle"
                              onClick={() => toggleConnector(connectorKey)}
                            >
                              <span className="tree-arrow">
                                {expandedConnectors[connectorKey] ? '▾' : '▸'}
                              </span>
                              <div>
                                <strong>{connector.name}</strong>
                                <small>
                                  {connector.service_type.toUpperCase()} · Estado {connector.status}
                                </small>
                              </div>
                            </button>

                            {expandedConnectors[connectorKey] && (
                              <div className="tree-connector__details">
                                <p>
                                  <strong>Base URL:</strong> {connector.base_url}
                                </p>
                                {isScaleAQ && (
                                  <div className="scaleaq-meta">
                                    <span>
                                      <strong>ScaleAQ Site ID:</strong> {scaleaqSiteId}
                                    </span>
                                    <span>
                                      <strong>Headers:</strong> Scale-Version {scaleVersion} ·
                                      Accept {acceptHeader}
                                    </span>
                                  </div>
                                )}
                                {isInnovex && (
                                  <div className="innovex-meta">
                                    <span>
                                      <strong>Monitor ID:</strong> {monitorId}
                                    </span>
                                    <span>
                                      <strong>Medición favorita:</strong> {preferredMedition}
                                    </span>
                                  </div>
                                )}

                                {isScaleAQ ? (
                                  <div className="scaleaq-groups">
                                    {SCALEAQ_DATA_GROUPS.map((group) => (
                                      <div key={group.title} className="scaleaq-group">
                                        <h5>{group.title}</h5>
                                        <p>{group.description}</p>
                                        <ul>
                                          {group.items.map((item) => (
                                            <li key={item.endpoint}>
                                              <span>{item.label}</span>
                                              <code>{item.endpoint}</code>
                                            </li>
                                          ))}
                                        </ul>
                                      </div>
                                    ))}
                                  </div>
                                ) : isInnovex ? (
                                  <div className="scaleaq-groups">
                                    {INNOVEX_DATA_GROUPS.map((group) => (
                                      <div key={group.title} className="scaleaq-group">
                                        <h5>{group.title}</h5>
                                        <p>{group.description}</p>
                                        <ul>
                                          {group.items.map((item) => (
                                            <li key={item.endpoint}>
                                              <span>{item.label}</span>
                                              <code>{item.endpoint}</code>
                                            </li>
                                          ))}
                                        </ul>
                                      </div>
                                    ))}
                                  </div>
                                ) : (
                                  <p className="tree-placeholder">
                                    Este conector aún no expone datos discovery en esta vista.
                                  </p>
                                )}
                              </div>
                            )}
                          </div>
                        )
                      })
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

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
