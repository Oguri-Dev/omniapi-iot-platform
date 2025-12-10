import api from './api'
import type { ExternalService } from './externalService.service'
import type { Site } from './site.service'
import type { DiscoveryResult, DiscoveryProvider, DiscoveryCache } from '../types/discovery'
import { buildScaleAQDiscoverySample } from '../lib/scaleaqDiscoverySample'
import { buildInnovexDiscoverySample } from '../lib/innovexDiscoverySample'

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

// Configuración para elegir entre discovery real o simulado
const USE_REAL_DISCOVERY = true

const discoveryService = {
  /**
   * Ejecuta el discovery de ScaleAQ para un centro específico.
   * Usa el backend para hacer llamadas reales a la API.
   */
  runScaleAQDiscovery: async (
    site: Site,
    connectors: ExternalService[] = []
  ): Promise<DiscoveryResult> => {
    const scaleConnector = connectors.find((conn) => conn.service_type === 'scaleaq')

    if (USE_REAL_DISCOVERY && scaleConnector) {
      try {
        const response = await api.post('/api/discovery/run', {
          provider: 'scaleaq',
          service_id: scaleConnector._id || scaleConnector.id,
          site_id: site.id || (site as any)._id,
        })

        if (response.data.success && response.data.data) {
          return transformBackendResponse(response.data.data, 'scaleaq')
        }
      } catch (error) {
        console.error('Error en discovery real ScaleAQ, usando fallback:', error)
      }
    }

    // Fallback a datos simulados
    const siteIdFromConnector =
      (scaleConnector?.config?.scaleaq_site_id as string) || site.id || site.code

    await sleep(600)

    return buildScaleAQDiscoverySample({
      siteName: site.name,
      siteCode: site.code,
      siteId: siteIdFromConnector,
      tenantCode: site.tenant_code,
      scaleVersion: (scaleConnector?.config?.scale_version as string) || '2025-01-01',
      acceptHeader: (scaleConnector?.config?.accept_header as string) || 'application/json',
    })
  },

  /**
   * Ejecuta el discovery de Innovex para un centro específico.
   * Usa el backend para hacer llamadas reales a la API.
   */
  runInnovexDiscovery: async (
    site: Site,
    connectors: ExternalService[] = []
  ): Promise<DiscoveryResult> => {
    const innovexConnector = connectors.find((conn) => conn.service_type === 'innovex')
    const monitorId =
      (innovexConnector?.config?.monitor_id as string) || site.code || site.id || 'monitor-unknown'

    if (USE_REAL_DISCOVERY && innovexConnector) {
      try {
        const response = await api.post('/api/discovery/run', {
          provider: 'innovex',
          service_id: innovexConnector._id || innovexConnector.id,
          site_id: site.id || (site as any)._id,
          monitor_id: monitorId,
        })

        if (response.data.success && response.data.data) {
          return transformBackendResponse(response.data.data, 'innovex')
        }
      } catch (error) {
        console.error('Error en discovery real Innovex, usando fallback:', error)
      }
    }

    // Fallback a datos simulados
    const meditionSample = (innovexConnector?.config?.medition as string) || 'oxygen'

    await sleep(600)

    return buildInnovexDiscoverySample({
      siteName: site.name,
      siteCode: site.code,
      tenantCode: site.tenant_code,
      monitorId,
      meditionSample,
    })
  },
}

/**
 * Transforma la respuesta del backend al formato esperado por el frontend
 */
function transformBackendResponse(data: any, provider: DiscoveryProvider): DiscoveryResult {
  return {
    provider,
    siteId: data.site_id,
    siteName: data.site_name,
    tenantCode: data.tenant_code,
    generatedAt: data.generated_at,
    headersUsed: data.headers_used || {},
    summary: {
      timeseriesCount: data.summary?.total_endpoints || 0,
      metricsAvailable: data.summary?.available_endpoints || 0,
      range: `Discovery real · ${new Date(data.generated_at).toLocaleString()}`,
    },
    groups: (data.groups || []).map((group: any) => ({
      id: group.id,
      title: group.title,
      description: group.description,
      endpoints: (group.endpoints || []).map((ep: any) => ({
        label: ep.label,
        method: ep.method,
        path: ep.path,
        description: ep.description,
        lastSync: ep.last_sync,
        availability: ep.availability || 'ready',
        fullUrl: ep.full_url,
        statusCode: ep.status_code,
        latencyMs: ep.latency_ms,
        dataset: ep.data
          ? {
              title: `Respuesta real (${ep.status_code || 'N/A'})`,
              highlights: [
                `Latencia: ${ep.latency_ms || 0}ms`,
                ep.error ? `Error: ${ep.error}` : 'Respuesta exitosa',
              ],
              rawData: ep.data,
            }
          : undefined,
      })),
    })),
  }
}

export type { DiscoveryResult, DiscoveryProvider, DiscoveryCache }
export default discoveryService
