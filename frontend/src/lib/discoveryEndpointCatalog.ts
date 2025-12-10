import type { DiscoveryProvider } from '../services/discovery.service'
import type { BuilderEndpointMeta, BuilderEndpointSampleContext } from '../types/builder'

const fallbackSiteId = (context: BuilderEndpointSampleContext, fallback = 'CL-REL-01') =>
  context.site?.id || fallback

const parseListParam = (value?: string, fallback: string[] = []) =>
  value
    ?.split(',')
    .map((chunk) => chunk.trim())
    .filter(Boolean) || fallback

const buildTimestampSeries = (count = 6, stepMinutes = 15, baseValue = 10, amplitude = 2) => {
  const now = Date.now()
  return Array.from({ length: count }).map((_, index) => {
    const timestamp = new Date(now - (count - index - 1) * stepMinutes * 60 * 1000).toISOString()
    const value = Number((baseValue + Math.sin(index / 2) * amplitude).toFixed(2))
    return { timestamp, value }
  })
}

const buildChannelSeries = (channels: string[]) =>
  channels.reduce<Record<string, { timestamp: string; value: number }[]>>((acc, channel, idx) => {
    acc[channel] = buildTimestampSeries(6, 15, 8 + idx * 2, 1.3)
    return acc
  }, {})

export const discoveryEndpointCatalog: Record<DiscoveryProvider, BuilderEndpointMeta[]> = {
  scaleaq: [
    {
      id: 'scale-meta-company',
      provider: 'scaleaq',
      label: 'Company Info',
      method: 'GET',
      path: '/meta/company?include=all',
      description: 'Ficha de compañía completa utilizada en el discovery demo.',
      category: 'Meta',
      targetBlock: 'assets',
      sampleResponseHint: 'Devuelve datos corporativos y lista de sitios.',
      makeSampleResponse: () => ({
        companyId: 'scale-demo-01',
        name: 'ScaleAQ Benchmark Farms',
        tenantCode: 'REL',
        sites: [
          { id: 'CL-REL-01', name: 'Reloncaví Norte', cages: 8 },
          { id: 'CL-REL-08', name: 'Reloncaví Sur', cages: 6 },
        ],
        updatedAt: new Date().toISOString(),
      }),
    },
    {
      id: 'scale-meta-site',
      provider: 'scaleaq',
      label: 'Site Info',
      method: 'GET',
      path: '/meta/sites/{siteId}?include=all',
      description: 'Información detallada del centro seleccionado.',
      category: 'Meta',
      targetBlock: 'assets',
      sampleResponseHint: 'Incluye jaulas, ubicación, biomass target.',
      params: [{ name: 'siteId', label: 'Site ID', required: true, placeholder: 'CL-REL-01' }],
      makeSampleResponse: (context) => ({
        siteId: fallbackSiteId(context),
        name: context.site?.name || 'Centro Reloncaví',
        location: {
          region: 'Los Lagos',
          lat: -41.922,
          lon: -72.953,
        },
        cages: Array.from({ length: 6 }).map((_, idx) => ({
          id: `C${idx + 1}`,
          biomassTargetKg: 19000 + idx * 800,
          depth: 28 + idx,
        })),
        health: {
          seaLice: 1.2,
          mortality: 0.04,
        },
      }),
    },
    {
      id: 'scale-ts-retrieve',
      provider: 'scaleaq',
      label: 'Time Series Retrieve',
      method: 'POST',
      path: '/time-series/retrieve',
      description:
        'Consulta cruda de series basadas en canales/timeRange. Útil para oxígeno, alimento y biomasa.',
      category: 'Time Series',
      targetBlock: 'timeseries',
      sampleResponseHint: 'Devuelve arrays por canal con timestamps.',
      params: [
        {
          name: 'channels',
          label: 'Channels',
          required: true,
          placeholder: 'oxygen_ppm, feeding_rate',
          helperText: 'Lista separada por coma de channels permitidos.',
        },
        {
          name: 'range',
          label: 'Rango (ej. 48h)',
          required: true,
          placeholder: '48h',
        },
      ],
      makeSampleResponse: (context) => {
        const channels = parseListParam(context.params?.channels, ['oxygen_ppm', 'feeding_rate'])
        return {
          siteId: fallbackSiteId(context),
          range: context.params?.range || '48h',
          channels,
          data: buildChannelSeries(channels),
        }
      },
    },
    {
      id: 'scale-ts-data-types',
      provider: 'scaleaq',
      label: 'Available Data Types',
      method: 'POST',
      path: '/time-series/retrieve/data-types',
      description: 'Catálogo de canales y unidades disponibles para el sitio.',
      category: 'Time Series',
      targetBlock: 'assets',
      sampleResponseHint: 'Retorna labels, units y frecuencias.',
      makeSampleResponse: (context) => ({
        siteId: fallbackSiteId(context),
        dataTypes: [
          { channel: 'oxygen_ppm', unit: 'ppm', frequency: '1m' },
          { channel: 'feeding_rate', unit: 'kg/min', frequency: '10m' },
          { channel: 'biomass_tons', unit: 'tons', frequency: '1h' },
        ],
      }),
    },
    {
      id: 'scale-units-aggregate',
      provider: 'scaleaq',
      label: 'Units Aggregate',
      method: 'POST',
      path: '/time-series/retrieve/units/aggregate',
      description: 'KPIs por unidad de cultivo (consumo, biomasa, oxígeno).',
      category: 'Aggregates',
      targetBlock: 'kpis',
      params: [
        {
          name: 'units',
          label: 'Unidades',
          placeholder: 'CL-REL-01, CL-REL-08',
          helperText: 'Opcional. Por defecto retorna todas las unidades.',
        },
      ],
      makeSampleResponse: (context) => ({
        siteId: fallbackSiteId(context),
        units: (context.params?.units || 'CL-REL-01,CL-REL-08').split(',').map((unit, index) => ({
          unitId: unit.trim(),
          feedingKg: 1200 + index * 150,
          biomassKg: 18000 + index * 900,
          oxygenMin: 7.8 - index * 0.2,
        })),
      }),
    },
    {
      id: 'scale-silos-aggregate',
      provider: 'scaleaq',
      label: 'Silos Aggregate',
      method: 'POST',
      path: '/time-series/retrieve/silos/aggregate',
      description: 'Resumen de stock y consumo de silos.',
      category: 'Aggregates',
      targetBlock: 'kpis',
      params: [
        {
          name: 'silos',
          label: 'IDs de Silos',
          placeholder: 'S1,S2',
        },
      ],
      makeSampleResponse: () => ({
        silos: [
          { siloId: 'S1', stockKg: 8500, feedType: 'EW 3mm', lastRefill: '2025-02-07T10:00:00Z' },
          { siloId: 'S2', stockKg: 6200, feedType: 'EW 5mm', lastRefill: '2025-02-05T14:00:00Z' },
        ],
      }),
    },
    {
      id: 'scale-feeding-units',
      provider: 'scaleaq',
      label: 'Feeding Units',
      method: 'GET',
      path: '/feeding-dashboard/units?siteId={siteId}',
      description: 'Resumen operacional por unidad (consumo, mortalidad, alertas).',
      category: 'Feeding',
      targetBlock: 'snapshots',
      params: [{ name: 'siteId', label: 'Site ID', required: true, placeholder: 'CL-REL-01' }],
      makeSampleResponse: (context) => ({
        siteId: fallbackSiteId(context),
        updatedAt: new Date().toISOString(),
        units: [
          { id: 'U1', feedingKg: 320, mortality: 0.05, alerts: 0 },
          { id: 'U2', feedingKg: 295, mortality: 0.04, alerts: 1 },
        ],
      }),
    },
    {
      id: 'scale-feeding-timeline',
      provider: 'scaleaq',
      label: 'Feeding Timeline',
      method: 'GET',
      path: '/feeding-dashboard/timeline?siteId={siteId}',
      description: 'Serie consolidada de alimento por bloques de 10 minutos.',
      category: 'Feeding',
      targetBlock: 'timeseries',
      params: [{ name: 'siteId', label: 'Site ID', required: true, placeholder: 'CL-REL-01' }],
      makeSampleResponse: (context) => ({
        siteId: fallbackSiteId(context),
        timeline: buildTimestampSeries(12, 10, 25, 4).map(({ timestamp, value }) => ({
          timestamp,
          feedingKg: Number((value * 12).toFixed(1)),
        })),
      }),
    },
    {
      id: 'scale-analytics-kpis',
      provider: 'scaleaq',
      label: 'Analytics KPIs',
      method: 'GET',
      path: '/analytics/kpis?siteId={siteId}',
      description: 'Indicadores operacionales diarios utilizados en el demo.',
      category: 'Analytics',
      targetBlock: 'kpis',
      params: [{ name: 'siteId', label: 'Site ID', required: true, placeholder: 'CL-REL-01' }],
      makeSampleResponse: (context) => ({
        siteId: fallbackSiteId(context),
        reportDate: new Date().toISOString().slice(0, 10),
        kpis: {
          fcr: 1.23,
          avgOxygen: 8.1,
          avgTemp: 12.4,
          biomassDelta: 2.4,
        },
      }),
    },
  ],
  innovex: [
    {
      id: 'innovex-all-monitors',
      provider: 'innovex',
      label: 'All Monitors',
      method: 'GET',
      path: '/api_dataweb/all_monitors/?active=all',
      description: 'Lista monitores asociados al cliente con coordenadas y monitor_key.',
      category: 'Catálogo',
      targetBlock: 'assets',
      makeSampleResponse: () => ({
        count: 2,
        monitors: [
          { id: 1201, name: 'Monitor 1201', lat: -41.9, lon: -72.95, monitor_key: 'REL-AX1' },
          { id: 1202, name: 'Monitor 1202', lat: -41.91, lon: -72.97, monitor_key: 'REL-AX2' },
        ],
      }),
    },
    {
      id: 'innovex-monitor-detail',
      provider: 'innovex',
      label: 'Monitor Detail',
      method: 'GET',
      path: '/api_dataweb/monitor_detail/?monitor_id={monitor_id}',
      description: 'Detalle de sensores, jaulas y profundidad por monitor.',
      category: 'Catálogo',
      targetBlock: 'assets',
      params: [
        {
          name: 'monitor_id',
          label: 'Monitor ID',
          required: true,
          placeholder: '1234',
        },
      ],
      makeSampleResponse: (context) => ({
        monitorId: context.params?.monitor_id || '1234',
        status: 'active',
        depth: 40,
        cages: 6,
        sensors: [
          { id: 'OX-1', medition: 'oxygen', depth: 5 },
          { id: 'TMP-2', medition: 'temperature', depth: 15 },
        ],
      }),
    },
    {
      id: 'innovex-monitor-sensor-last-data',
      provider: 'innovex',
      label: 'Monitor Sensor Last Data',
      method: 'GET',
      path: '/api_dataweb/monitor_sensor_last_data/?id={monitor_id}&medition=oxygen',
      description: 'Última medición por sensor para una medition específica.',
      category: 'Lecturas',
      targetBlock: 'snapshots',
      params: [
        { name: 'monitor_id', label: 'Monitor ID', required: true, placeholder: '1234' },
        {
          name: 'medition',
          label: 'Medition',
          required: true,
          placeholder: 'oxygen',
          helperText: 'oxygen | flow | weather | temperature',
        },
      ],
      makeSampleResponse: (context) => ({
        monitorId: context.params?.monitor_id || '1234',
        medition: context.params?.medition || 'oxygen',
        updatedAt: new Date().toISOString(),
        sensors: [
          { sensor_id: 'S-01', value: 8.3, unit: 'ppm' },
          { sensor_id: 'S-02', value: 8.1, unit: 'ppm' },
        ],
      }),
    },
    {
      id: 'innovex-get-last-data',
      provider: 'innovex',
      label: 'Get Last Data (sensor)',
      method: 'GET',
      path: '/api_dataweb/get_last_data/?monitor_id={monitor_id}&sensor_id={sensor_id}',
      description: 'Dato puntual de un sensor específico, ideal para alertas.',
      category: 'Lecturas',
      targetBlock: 'snapshots',
      params: [
        { name: 'monitor_id', label: 'Monitor ID', required: true },
        { name: 'sensor_id', label: 'Sensor ID', required: true },
      ],
      makeSampleResponse: (context) => ({
        monitorId: context.params?.monitor_id || '1201',
        sensorId: context.params?.sensor_id || 'S-01',
        value: 7.95,
        unit: 'ppm',
        timestamp: new Date().toISOString(),
      }),
    },
    {
      id: 'innovex-get-data-range',
      provider: 'innovex',
      label: 'Get Data Range',
      method: 'GET',
      path: '/api_dataweb/get_data_range/?monitor_id={monitor_id}&sensor_id={sensor_id}&unixtime_since=...&unixtime_until=...',
      description: 'Devuelve serie en un rango definido (máx 30 días).',
      category: 'Históricos',
      targetBlock: 'timeseries',
      params: [
        { name: 'monitor_id', label: 'Monitor ID', required: true },
        { name: 'sensor_id', label: 'Sensor ID', required: true },
        {
          name: 'range',
          label: 'Rango (ej. 2025-01-01/2025-01-03)',
          required: true,
        },
      ],
      makeSampleResponse: (context) => ({
        monitorId: context.params?.monitor_id || '1201',
        sensorId: context.params?.sensor_id || 'S-01',
        range: context.params?.range || '2025-01-01/2025-01-03',
        points: buildTimestampSeries(10, 60, 8, 0.6),
      }),
    },
    {
      id: 'innovex-monitor-sensor-time-data',
      provider: 'innovex',
      label: 'Monitor Sensor Time Data',
      method: 'GET',
      path: '/api_dataweb/monitor_sensor_time_data/?monitor_id={monitor_id}&medition=oxygen&unixtime_since=...&unixtime_until=...',
      description: 'Series agrupadas por sensor para una medition completa.',
      category: 'Históricos',
      targetBlock: 'timeseries',
      params: [
        { name: 'monitor_id', label: 'Monitor ID', required: true },
        { name: 'medition', label: 'Medition', required: true, placeholder: 'oxygen' },
        {
          name: 'range',
          label: 'Rango Unix',
          required: true,
          placeholder: '1700700000-1700786400',
        },
      ],
      makeSampleResponse: (context) => ({
        monitorId: context.params?.monitor_id || '1201',
        medition: context.params?.medition || 'oxygen',
        range: context.params?.range || '1700700000-1700786400',
        sensors: [
          { sensor_id: 'S-01', series: buildTimestampSeries(8, 30, 8, 0.4) },
          { sensor_id: 'S-02', series: buildTimestampSeries(8, 30, 8.4, 0.5) },
        ],
      }),
    },
    {
      id: 'innovex-error-table',
      provider: 'innovex',
      label: 'Error Reference',
      method: 'DOC',
      path: 'Manual Innovex – tabla de errores',
      description: 'Lista de respuestas comunes (Access Denied, Unauthorized sensor...).',
      category: 'Alertas',
      targetBlock: 'alerts',
      makeSampleResponse: () => ({
        errors: [
          { code: 'AccessDenied', hint: 'Revisar API Key y monitor asignado.' },
          { code: 'UnauthorizedSensor', hint: 'Sensor no pertenece al monitor indicado.' },
        ],
      }),
    },
  ],
}

export type ProviderEndpointCatalog = typeof discoveryEndpointCatalog
