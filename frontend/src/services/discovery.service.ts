import type { ExternalService } from './externalService.service'
import type { Site } from './site.service'
import type { DiscoveryResult, DiscoveryProvider, DiscoveryCache } from '../types/discovery'
import { buildScaleAQDiscoverySample } from '../lib/scaleaqDiscoverySample'
import { buildInnovexDiscoverySample } from '../lib/innovexDiscoverySample'

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

const discoveryService = {
  /**
   * Ejecuta el discovery de ScaleAQ para un centro específico.
   * Por ahora devuelve los datos del archivo de ejemplo compartido.
   */
  runScaleAQDiscovery: async (
    site: Site,
    connectors: ExternalService[] = []
  ): Promise<DiscoveryResult> => {
    const scaleConnector = connectors.find((conn) => conn.service_type === 'scaleaq')
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

  runInnovexDiscovery: async (
    site: Site,
    connectors: ExternalService[] = []
  ): Promise<DiscoveryResult> => {
    const innovexConnector = connectors.find((conn) => conn.service_type === 'innovex')
    const monitorId =
      (innovexConnector?.config?.monitor_id as string) || site.code || site.id || 'monitor-unknown'
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

export type { DiscoveryResult, DiscoveryProvider, DiscoveryCache }
export default discoveryService
