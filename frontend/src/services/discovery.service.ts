import type { ExternalService } from './externalService.service'
import type { Site } from './site.service'
import type { ScaleAQDiscoveryResult } from '../types/discovery'
import { buildScaleAQDiscoverySample } from '../lib/scaleaqDiscoverySample'

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

const discoveryService = {
  /**
   * Ejecuta el discovery de ScaleAQ para un centro específico.
   * Por ahora devuelve los datos del archivo de ejemplo compartido.
   */
  runScaleAQDiscovery: async (
    site: Site,
    connectors: ExternalService[] = []
  ): Promise<ScaleAQDiscoveryResult> => {
    const scaleConnector = connectors.find((conn) => conn.service_type === 'scaleaq')
    const siteIdFromConnector =
      (scaleConnector?.config?.scaleaq_site_id as string) || site.id || site.code

    await sleep(600) // Simula latencia de red

    return buildScaleAQDiscoverySample({
      siteName: site.name,
      siteCode: site.code,
      siteId: siteIdFromConnector,
      tenantCode: site.tenant_code,
      scaleVersion: (scaleConnector?.config?.scale_version as string) || '2025-01-01',
      acceptHeader: (scaleConnector?.config?.accept_header as string) || 'application/json',
    })
  },
}

export type { ScaleAQDiscoveryResult }
export default discoveryService
