import React, { useMemo } from 'react'
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
  const grouped = useMemo(() => {
    return endpoints.reduce<Record<string, BuilderEndpointMeta[]>>((acc, item) => {
      if (!acc[item.category]) {
        acc[item.category] = []
      }
      acc[item.category].push(item)
      return acc
    }, {})
  }, [endpoints])

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

    return {
      provider,
      site: {
        id: (site.id as string) || ((site as any)._id as string) || 'unknown-site',
        name: site.name,
        code: site.code,
        tenantCode: site.tenant_code,
      },
      generatedAt: new Date().toISOString(),
      endpoints: selectedEndpoints,
    }
  }, [provider, site, selectedEndpoints])

  const handleCopy = () => {
    if (!preview) return
    navigator.clipboard.writeText(JSON.stringify(preview, null, 2))
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
                return (
                  <article
                    key={endpoint.id}
                    className={`endpoint-card ${enabled ? 'is-active' : ''}`}
                  >
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
