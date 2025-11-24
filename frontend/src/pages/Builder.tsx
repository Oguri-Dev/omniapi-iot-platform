import React, { useEffect, useMemo, useState } from 'react'
import siteService, { type Site } from '../services/site.service'
import externalServiceService, { type ExternalService } from '../services/externalService.service'
import discoveryService, {
  type DiscoveryProvider,
  type DiscoveryResult,
  type DiscoveryCache,
} from '../services/discovery.service'
import builderService, {
  type BuilderRunResponse,
  type BuilderEndpointPayload,
} from '../services/builder.service'
import EndpointBuilder, { type EndpointSelectionState } from '../components/EndpointBuilder'
import { discoveryEndpointCatalog } from '../lib/discoveryEndpointCatalog'
import type { BuilderEndpointMeta } from '../types/builder'
import '../styles/Services.css'

const DISCOVERY_PROVIDERS: Array<{ id: DiscoveryProvider; label: string; description: string }> = [
  {
    id: 'scaleaq',
    label: 'ScaleAQ',
    description: 'Discovery basado en Time Series y Feeding del archivo demo.',
  },
  {
    id: 'innovex',
    label: 'Innovex',
    description: 'Discovery de Innovex Dataweb (monitores, lecturas y alertas).',
  },
]

const BuilderPage: React.FC = () => {
  const [sites, setSites] = useState<Site[]>([])
  const [connectors, setConnectors] = useState<ExternalService[]>([])
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null)
  const [discoveryProvider, setDiscoveryProvider] = useState<DiscoveryProvider>('scaleaq')
  const [discoveryCache, setDiscoveryCache] = useState<DiscoveryCache>({})
  const [discoveryLoading, setDiscoveryLoading] = useState(false)
  const [discoveryError, setDiscoveryError] = useState<string | null>(null)
  const [builderSelections, setBuilderSelections] = useState<
    Record<string, EndpointSelectionState>
  >({})
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle')
  const [statusMessage, setStatusMessage] = useState<string | null>(null)
  const [history, setHistory] = useState<BuilderRunResponse[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)

  useEffect(() => {
    loadSites()
    loadConnectors()
  }, [])

  useEffect(() => {
    if (selectedSiteId) {
      loadHistory(selectedSiteId, discoveryProvider)
    }
  }, [selectedSiteId, discoveryProvider])

  useEffect(() => {
    setBuilderSelections({})
  }, [selectedSiteId, discoveryProvider])

  const loadSites = async () => {
    try {
      const response = await siteService.getAll()
      if (response.success) {
        setSites(response.data)
      } else {
        setSites([])
      }
    } catch (error) {
      console.error('Error cargando sitios', error)
      setSites([])
    }
  }

  const loadConnectors = async () => {
    try {
      const response = await externalServiceService.getAll()
      if (response.success) {
        setConnectors(response.data)
      } else {
        setConnectors([])
      }
    } catch (error) {
      console.error('Error cargando conectores', error)
      setConnectors([])
    }
  }

  const loadHistory = async (siteKey: string, provider: DiscoveryProvider) => {
    try {
      setHistoryLoading(true)
      const response = await builderService.listRuns({ site_id: siteKey, provider, limit: 5 })
      if (response.success) {
        setHistory(response.data || [])
      } else {
        setHistory([])
      }
    } catch (error) {
      console.error('Error cargando historial', error)
      setHistory([])
    } finally {
      setHistoryLoading(false)
    }
  }

  const getSiteKey = (site: Site) => site.id || (site as any)._id || site.code

  const siteTree = useMemo(() => {
    if (!sites.length) return []

    return sites.map((site) => {
      const siteKey = getSiteKey(site)
      const related = connectors.filter(
        (conn) => conn.site_id === siteKey || conn.site_id === (site as any)._id
      )
      return { site, siteKey, connectors: related }
    })
  }, [sites, connectors])

  useEffect(() => {
    if (!selectedSiteId && siteTree.length) {
      const preferred = siteTree.find((entry) => entry.connectors.length > 0) ?? siteTree[0]
      if (preferred) {
        setSelectedSiteId(preferred.siteKey)
      }
    }
  }, [siteTree, selectedSiteId])

  const selectedSiteNode = useMemo(() => {
    return siteTree.find((entry) => entry.siteKey === selectedSiteId) || null
  }, [siteTree, selectedSiteId])

  const selectedSite = selectedSiteNode?.site ?? null

  const providerInfo = DISCOVERY_PROVIDERS.find((item) => item.id === discoveryProvider)

  const providerConnectors = useMemo(() => {
    if (!selectedSiteNode) return []
    return selectedSiteNode.connectors.filter((conn) => conn.service_type === discoveryProvider)
  }, [selectedSiteNode, discoveryProvider])

  const builderCatalog = useMemo<BuilderEndpointMeta[]>(() => {
    return discoveryEndpointCatalog[discoveryProvider] || []
  }, [discoveryProvider])

  const builderSelectedEndpoints = useMemo(() => {
    return builderCatalog.filter((endpoint) => builderSelections[endpoint.id]?.enabled)
  }, [builderCatalog, builderSelections])

  const builderSelectedCount = builderSelectedEndpoints.length

  const makeDiscoveryKey = (siteKey: string, provider: DiscoveryProvider) =>
    `${siteKey}:${provider}`

  const activeDiscoveryKey = selectedSiteId
    ? makeDiscoveryKey(selectedSiteId, discoveryProvider)
    : null
  const activeDiscovery: DiscoveryResult | null = activeDiscoveryKey
    ? discoveryCache[activeDiscoveryKey] ?? null
    : null

  const handleSelectSite = (siteKey: string) => {
    setSelectedSiteId(siteKey)
  }

  const handleRunDiscovery = async () => {
    if (!selectedSiteNode) return
    if (providerConnectors.length === 0) {
      setDiscoveryError('Necesitas al menos un conector activo para este proveedor.')
      return
    }

    setDiscoveryError(null)
    setDiscoveryLoading(true)

    try {
      let result: DiscoveryResult
      if (discoveryProvider === 'innovex') {
        result = await discoveryService.runInnovexDiscovery(
          selectedSiteNode.site,
          providerConnectors
        )
      } else {
        result = await discoveryService.runScaleAQDiscovery(
          selectedSiteNode.site,
          providerConnectors
        )
      }

      const cacheKey = makeDiscoveryKey(selectedSiteNode.siteKey, discoveryProvider)
      setDiscoveryCache((prev) => ({
        ...prev,
        [cacheKey]: result,
      }))
      setStatusMessage('Discovery actualizado correctamente.')
    } catch (error) {
      console.error('Discovery error', error)
      setDiscoveryError('No se pudo ejecutar el discovery. Revisa el conector y vuelve a intentar.')
    } finally {
      setDiscoveryLoading(false)
    }
  }

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

  const handleSaveBuilder = async () => {
    if (!selectedSiteId || !selectedSite || builderSelectedCount === 0) {
      setStatusMessage('Selecciona un centro y al menos un endpoint para guardar.')
      return
    }

    const endpointsPayload: BuilderEndpointPayload[] = builderSelectedEndpoints.map((endpoint) => ({
      endpoint_id: endpoint.id,
      label: endpoint.label,
      method: endpoint.method,
      path: endpoint.path,
      target_block: endpoint.targetBlock,
      params: builderSelections[endpoint.id]?.params,
    }))

    const payload = {
      provider: discoveryProvider,
      site: {
        id: selectedSiteId,
        code: selectedSite.code,
        name: selectedSite.name,
        tenant_id: selectedSite.tenant_id,
        tenant_code: selectedSite.tenant_code,
      },
      endpoints: endpointsPayload,
      discovery_summary: activeDiscovery
        ? {
            summary: activeDiscovery.summary,
            generatedAt: activeDiscovery.generatedAt,
            groups: activeDiscovery.groups.map((group) => ({
              id: group.id,
              title: group.title,
              endpoints: group.endpoints.length,
            })),
          }
        : undefined,
    }

    try {
      setSaveStatus('saving')
      setStatusMessage(null)
      const response = await builderService.saveRun(payload)
      if (response.success) {
        setSaveStatus('success')
        setStatusMessage('Builder guardado en MongoDB correctamente.')
        loadHistory(selectedSiteId, discoveryProvider)
      } else {
        setSaveStatus('error')
        setStatusMessage(response.message || 'Error guardando el builder.')
      }
    } catch (error) {
      console.error('Error guardando builder', error)
      setSaveStatus('error')
      setStatusMessage('Hubo un error guardando en MongoDB.')
    } finally {
      setTimeout(() => setSaveStatus('idle'), 1500)
    }
  }

  return (
    <div className="services-page builder-page">
      <header className="services-header">
        <div>
          <h1>Discovery Builder</h1>
          <p>
            Organiza los endpoints seleccionados para cada centro y guarda la plantilla en MongoDB.
          </p>
        </div>
        <div className="services-actions">
          <button
            className="btn-secondary"
            onClick={handleRunDiscovery}
            disabled={!selectedSiteNode || discoveryLoading}
          >
            {discoveryLoading ? 'Corriendo discovery...' : 'Ejecutar discovery'}
          </button>
          <button
            className="btn-primary"
            onClick={handleSaveBuilder}
            disabled={!selectedSiteNode || builderSelectedCount === 0 || saveStatus === 'saving'}
          >
            {saveStatus === 'saving' ? 'Guardando...' : 'Guardar selección'}
          </button>
        </div>
      </header>

      {statusMessage && <div className="services-alert success">{statusMessage}</div>}
      {discoveryError && <div className="services-alert error">{discoveryError}</div>}

      <section className="services-layout">
        <aside className="services-sidebar">
          <div className="sidebar-card">
            <h3>Centros</h3>
            <ul className="tree-list">
              {siteTree.map((entry) => (
                <li key={entry.siteKey}>
                  <button
                    type="button"
                    className={`tree-item ${selectedSiteId === entry.siteKey ? 'is-active' : ''}`}
                    onClick={() => handleSelectSite(entry.siteKey)}
                  >
                    <span>{entry.site.name}</span>
                    <small>{entry.site.code}</small>
                  </button>
                </li>
              ))}
            </ul>
          </div>

          <div className="sidebar-card">
            <h3>Proveedor</h3>
            <div className="provider-tabs">
              {DISCOVERY_PROVIDERS.map((provider) => (
                <button
                  key={provider.id}
                  type="button"
                  className={provider.id === discoveryProvider ? 'is-active' : ''}
                  onClick={() => setDiscoveryProvider(provider.id)}
                >
                  <strong>{provider.label}</strong>
                  <span>{provider.description}</span>
                </button>
              ))}
            </div>
          </div>

          <div className="sidebar-card">
            <h3>Conectores activos</h3>
            {providerConnectors.length === 0 ? (
              <p className="empty-state">
                Agrega un conector {providerInfo?.label} al centro seleccionado.
              </p>
            ) : (
              <ul className="connector-list">
                {providerConnectors.map((conn) => (
                  <li key={conn._id}>
                    <strong>{conn.name}</strong>
                    <span>{conn.base_url}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="sidebar-card">
            <h3>Últimos guardados</h3>
            {historyLoading ? (
              <p>Cargando historial...</p>
            ) : history.length === 0 ? (
              <p className="empty-state">Sin registros para este centro.</p>
            ) : (
              <ul className="history-list">
                {history.map((entry) => (
                  <li key={entry.id}>
                    <p>{new Date(entry.created_at || '').toLocaleString()}</p>
                    <span>{entry.endpoints.length} endpoints</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </aside>

        <div className="services-content">
          <section className="discovery-panel">
            <header>
              <h2>Discovery {providerInfo?.label}</h2>
              <p>
                Usa el resultado para validar los endpoints marcados. Última ejecución:{' '}
                {activeDiscovery ? new Date(activeDiscovery.generatedAt).toLocaleString() : 'n/a'}
              </p>
            </header>
            <div className="discovery-preview">
              <pre>
                {activeDiscovery
                  ? JSON.stringify(activeDiscovery.summary, null, 2)
                  : '// Ejecuta el discovery para ver el resumen'}
              </pre>
            </div>
          </section>

          <section className="builder-panel">
            <header>
              <h2>Endpoint Builder</h2>
              <p>
                {builderSelectedCount} endpoints seleccionados para {selectedSite?.name || '—'} ({' '}
                {providerInfo?.label}).
              </p>
            </header>

            <EndpointBuilder
              provider={discoveryProvider}
              site={selectedSite}
              endpoints={builderCatalog}
              selections={builderSelections}
              onToggle={handleBuilderToggle}
              onParamChange={handleBuilderParamChange}
            />
          </section>
        </div>
      </section>
    </div>
  )
}

export default BuilderPage
