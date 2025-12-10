import React, { useMemo, useState } from 'react'
import type { Site } from '../services/site.service'
import type { DiscoveryProvider } from '../services/discovery.service'
import type {
  BuilderEndpointMeta,
  BuilderEndpointSelection,
  BuilderPayloadPreview,
} from '../types/builder'

export interface EndpointSelectionState {
  enabled: boolean
  params: Record<string, string>
}

interface EndpointBuilderProps {
  provider: DiscoveryProvider
  site: Site | null
  endpoints: BuilderEndpointMeta[]
  selections: Record<string, EndpointSelectionState>
  onToggle: (endpointId: string, enabled: boolean) => void
  onParamChange: (endpointId: string, paramName: string, value: string) => void
}

const EndpointBuilder: React.FC<EndpointBuilderProps> = ({
  provider,
  site,
  endpoints,
  selections,
  onToggle,
  onParamChange,
}) => {
  const [sampleResponses, setSampleResponses] = useState<
    Record<string, { status: 'success' | 'error'; payload: string }>
  >({})
  const [testingEndpoint, setTestingEndpoint] = useState<string | null>(null)

  const grouped = useMemo(() => {
    return endpoints.reduce<Record<string, BuilderEndpointMeta[]>>((acc, item) => {
      if (!acc[item.category]) {
        acc[item.category] = []
      }
      acc[item.category].push(item)
      return acc
    }, {})
  }, [endpoints])

  const normalizedSite = useMemo(() => {
    if (!site) return null
    return {
      id: (site.id as string) || ((site as any)._id as string) || 'unknown-site',
      name: site.name,
      code: site.code,
      tenantCode: site.tenant_code,
    }
  }, [site])

  const selectedEndpoints = useMemo(() => {
    return endpoints
      .filter((endpoint) => selections[endpoint.id]?.enabled)
      .map<BuilderEndpointSelection>((endpoint) => ({
        endpointId: endpoint.id,
        label: endpoint.label,
        method: endpoint.method,
        path: endpoint.path,
        targetBlock: endpoint.targetBlock,
        params: selections[endpoint.id]?.params ?? {},
      }))
  }, [endpoints, selections])

  const preview: BuilderPayloadPreview | null = useMemo(() => {
    if (!site || selectedEndpoints.length === 0) return null

    if (!normalizedSite) return null

    return {
      provider,
      site: normalizedSite,
      generatedAt: new Date().toISOString(),
      endpoints: selectedEndpoints,
    }
  }, [provider, normalizedSite, selectedEndpoints])

  const handleCopy = () => {
    if (!preview) return
    navigator.clipboard.writeText(JSON.stringify(preview, null, 2))
  }

  const handleTestEndpoint = async (endpoint: BuilderEndpointMeta) => {
    if (!endpoint.makeSampleResponse) return
    if (!normalizedSite) {
      setSampleResponses((prev) => ({
        ...prev,
        [endpoint.id]: {
          status: 'error',
          payload: 'Selecciona un sitio antes de probar el endpoint.',
        },
      }))
      return
    }

    const selection = selections[endpoint.id]
    setTestingEndpoint(endpoint.id)
    try {
      await new Promise((resolve) => setTimeout(resolve, 350))
      const sample = endpoint.makeSampleResponse({
        provider,
        site: normalizedSite,
        params: selection?.params,
      })
      setSampleResponses((prev) => ({
        ...prev,
        [endpoint.id]: { status: 'success', payload: JSON.stringify(sample, null, 2) },
      }))
    } catch (error) {
      setSampleResponses((prev) => ({
        ...prev,
        [endpoint.id]: {
          status: 'error',
          payload:
            error instanceof Error ? error.message : 'No se pudo generar la respuesta de prueba.',
        },
      }))
    } finally {
      setTestingEndpoint(null)
    }
  }

  return (
    <div className="endpoint-builder">
      <div className="endpoint-builder__catalog">
        {Object.entries(grouped).map(([category, items]) => (
          <section key={category} className="endpoint-builder__group">
            <header>
              <h4>{category}</h4>
              <p>Selecciona los endpoints necesarios para {category.toLowerCase()}.</p>
            </header>
            <div className="endpoint-builder__cards">
              {items.map((endpoint) => {
                const selection = selections[endpoint.id]
                const enabled = Boolean(selection?.enabled)
                const hasMissingRequiredParams = Boolean(
                  enabled &&
                    endpoint.params?.some((param) =>
                      param.required ? !selection?.params?.[param.name]?.trim() : false
                    )
                )
                const sampleResponse = sampleResponses[endpoint.id]
                const showSample = Boolean(enabled && sampleResponse)
                const cardClassNames = [
                  'endpoint-card',
                  enabled ? 'is-active' : '',
                  showSample ? 'endpoint-card--wide' : '',
                ]
                return (
                  <article key={endpoint.id} className={cardClassNames.filter(Boolean).join(' ')}>
                    <div className="endpoint-card__header">
                      <label className="endpoint-card__toggle">
                        <input
                          type="checkbox"
                          checked={enabled}
                          onChange={(event) => onToggle(endpoint.id, event.target.checked)}
                        />
                        <span>{endpoint.label}</span>
                      </label>
                      <span className="endpoint-card__method">{endpoint.method}</span>
                    </div>
                    <p className="endpoint-card__path">{endpoint.path}</p>
                    <p className="endpoint-card__description">{endpoint.description}</p>
                    <p className="endpoint-card__target">Bloque destino: {endpoint.targetBlock}</p>
                    {endpoint.sampleResponseHint && (
                      <p className="endpoint-card__hint">{endpoint.sampleResponseHint}</p>
                    )}
                    {enabled && endpoint.params && (
                      <div className="endpoint-card__params">
                        {endpoint.params.map((param) => (
                          <label key={param.name}>
                            <span>{param.label}</span>
                            <input
                              type="text"
                              placeholder={param.placeholder}
                              value={selection?.params?.[param.name] ?? ''}
                              onChange={(event) =>
                                onParamChange(endpoint.id, param.name, event.target.value)
                              }
                            />
                            {param.helperText && (
                              <small className="helper-text">{param.helperText}</small>
                            )}
                          </label>
                        ))}
                      </div>
                    )}
                    {enabled && endpoint.makeSampleResponse && (
                      <div className="endpoint-card__actions">
                        <button
                          type="button"
                          className="btn-tertiary"
                          disabled={
                            testingEndpoint === endpoint.id ||
                            hasMissingRequiredParams ||
                            !normalizedSite
                          }
                          onClick={() => handleTestEndpoint(endpoint)}
                        >
                          {testingEndpoint === endpoint.id ? 'Probando...' : 'Probar respuesta'}
                        </button>
                        {!normalizedSite && (
                          <small>Selecciona un sitio para habilitar la prueba.</small>
                        )}
                        {hasMissingRequiredParams && (
                          <small>Completa los campos obligatorios para probar.</small>
                        )}
                      </div>
                    )}
                    {showSample && (
                      <div
                        className={`endpoint-card__sample endpoint-card__sample--${sampleResponse.status}`}
                      >
                        <header>
                          <strong>Última respuesta</strong>
                          <span>{sampleResponse.status === 'success' ? 'Simulada' : 'Error'}</span>
                        </header>
                        <pre>{sampleResponse.payload}</pre>
                      </div>
                    )}
                  </article>
                )
              })}
            </div>
          </section>
        ))}
      </div>

      <aside className="endpoint-builder__sidebar">
        <div className="endpoint-builder__summary">
          <h4>Selección actual</h4>
          <p>{selectedEndpoints.length} endpoints listos para construir el payload.</p>
          <ul>
            {selectedEndpoints.map((item) => (
              <li key={item.endpointId}>
                <strong>{item.label}</strong>
                <span>{item.method}</span>
              </li>
            ))}
          </ul>
        </div>

        <div className="endpoint-builder__preview">
          <div className="endpoint-builder__preview-header">
            <div>
              <h4>JSON a enviar</h4>
              <p>Incluye proveedor, sitio y el detalle de endpoints.</p>
            </div>
            <button
              type="button"
              className="btn-secondary"
              onClick={handleCopy}
              disabled={!preview}
            >
              Copiar JSON
            </button>
          </div>
          <pre>
            {preview
              ? JSON.stringify(preview, null, 2)
              : '// Selecciona endpoints para generar el payload'}
          </pre>
        </div>
      </aside>
    </div>
  )
}

export default EndpointBuilder
