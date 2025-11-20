import type { ScaleAQDiscoveryResult } from '../types/discovery'

export interface ScaleAQDiscoveryContext {
  siteName: string
  siteCode: string
  siteId: string
  tenantCode?: string
  scaleVersion: string
  acceptHeader: string
}

const nowIso = () => new Date().toISOString()

export const buildScaleAQDiscoverySample = (
  context: ScaleAQDiscoveryContext
): ScaleAQDiscoveryResult => {
  const generatedAt = nowIso()

  return {
    siteId: context.siteId,
    siteName: context.siteName,
    tenantCode: context.tenantCode,
    generatedAt,
    headersUsed: {
      scaleVersion: context.scaleVersion,
      accept: context.acceptHeader,
    },
    summary: {
      timeseriesCount: 84,
      metricsAvailable: 15,
      range: 'Últimas 72h · Mirroring archivo demo',
    },
    groups: [
      {
        id: 'meta',
        title: 'Catálogo & Meta',
        description:
          'Información estática entregada por /meta/company y /meta/sites para validar IDs de ScaleAQ.',
        endpoints: [
          {
            label: 'Company Info',
            method: 'GET',
            path: '/meta/company?include=all',
            description: 'Nombre legal, país, contactos y total de centros asociados.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Resumen compañía',
              summary: 'Información consolidada tal como aparece en el archivo de ejemplo.',
              metrics: [
                { label: 'Razón social', value: 'Omni Salmon Holding SpA', accent: 'info' },
                { label: 'Centros activos', value: '18', accent: 'success' },
                { label: 'Regiones', value: 'Los Lagos · Aysén' },
              ],
            },
          },
          {
            label: 'Site Info',
            method: 'GET',
            path: '/meta/sites/{siteId}?include=all',
            description:
              'Ficha completa por centro, incluyendo coordenadas, silos y unidades de cultivo.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: `Centro ${context.siteName}`,
              table: {
                columns: ['Campo', 'Valor'],
                rows: [
                  { Campo: 'ScaleAQ Site ID', Valor: context.siteId },
                  { Campo: 'Código interno', Valor: context.siteCode },
                  { Campo: 'Ubicación', Valor: 'Seno de Reloncaví, Chile' },
                  { Campo: 'Jaulas', Valor: '12' },
                  { Campo: 'Estado sanitario', Valor: 'Sin alertas en demo' },
                ],
              },
            },
          },
        ],
      },
      {
        id: 'timeseries',
        title: 'Time Series',
        description:
          'Endpoints POST /time-series/retrieve y /time-series/retrieve/data-types utilizados en el discovery.',
        endpoints: [
          {
            label: 'Retrieve',
            method: 'POST',
            path: '/time-series/retrieve',
            description: 'Entrega lecturas crudas por canal (oxígeno, alimentación, biomasa).',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Lecturas disponibles',
              metrics: [
                { label: 'Oxygen_ppm', value: '7.8 avg', accent: 'info' },
                { label: 'FeedRateKgH', value: '145 kg/h', accent: 'success' },
                { label: 'BiomassKg', value: '1.2M kg', accent: 'info' },
              ],
              highlights: [
                'Ventana cubierta: 2025-01-15 a 2025-01-18',
                'Resoluciones: 15s, 60s y 5min según canal',
              ],
            },
          },
          {
            label: 'Available Data Types',
            method: 'POST',
            path: '/time-series/retrieve/data-types',
            description: 'Lista los canales habilitados junto al tipo de dato y unidades.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Canales publicados',
              table: {
                columns: ['Canal', 'Unidad', 'Frecuencia'],
                rows: [
                  { Canal: 'oxygen_ppm', Unidad: 'mg/L', Frecuencia: '15 s' },
                  { Canal: 'feeding_rate', Unidad: 'kg/h', Frecuencia: '1 min' },
                  { Canal: 'biomass', Unidad: 'kg', Frecuencia: '5 min' },
                ],
              },
            },
          },
          {
            label: 'Units Aggregate',
            method: 'POST',
            path: '/time-series/retrieve/units/aggregate',
            description: 'Consolida métricas por unidad de cultivo para dashboards de producción.',
            lastSync: generatedAt,
            availability: 'partial',
            dataset: {
              title: 'Unidades agregadas',
              summary: 'Incluye ConsumoTotalKg, TemperaturaPromedio y Biomasa estimada por jaula.',
              highlights: [
                'Unidad CL-REL-01: Consumo 2.4 t / día',
                'Unidad CL-REL-08: Oxígeno estable (7.6 mg/L)',
              ],
            },
          },
          {
            label: 'Silos Aggregate',
            method: 'POST',
            path: '/time-series/retrieve/silos/aggregate',
            description: 'Datos del demo para stock de alimento y tasas de entrega.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Stock de silos',
              table: {
                columns: ['Silo', 'Stock (kg)', 'Consumo 24h'],
                rows: [
                  { Silo: 'S1', 'Stock (kg)': '12.4 t', 'Consumo 24h': '2.1 t' },
                  { Silo: 'S2', 'Stock (kg)': '9.8 t', 'Consumo 24h': '1.7 t' },
                ],
              },
            },
          },
        ],
      },
      {
        id: 'feeding',
        title: 'Feeding Dashboard',
        description: 'Endpoints utilizados en el archivo demo para comedores digitales y unidades.',
        endpoints: [
          {
            label: 'Units Consumption',
            method: 'GET',
            path: '/feeding-dashboard/units?siteId={siteId}',
            description: 'Detalle por unidad de consumo, mortalidad y alarmas.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'KPIs unidades',
              metrics: [
                { label: 'Consumo promedio', value: '145 kg/h' },
                { label: 'Alertas activas', value: '0', accent: 'success' },
                { label: 'Mortalidad 24h', value: '0.08%', accent: 'warning' },
              ],
            },
          },
          {
            label: 'Feeding Timeline',
            method: 'GET',
            path: '/feeding-dashboard/timeline?siteId={siteId}',
            description:
              'Serie consolidada utilizada en el PowerPoint demo para mostrar ritmo de alimentación.',
            lastSync: generatedAt,
            availability: 'partial',
            dataset: {
              title: 'Timeline de alimento',
              summary: 'Slots de 10 minutos con tasa de alimentación y temperatura superficial.',
            },
          },
        ],
      },
      {
        id: 'analytics',
        title: 'Analytics & KPIs',
        description:
          'Conjuntos derivados similares a los del archivo de ejemplo (órdenes, biomasa, oxígeno).',
        endpoints: [
          {
            label: 'Operational KPIs',
            method: 'GET',
            path: '/analytics/kpis?siteId={siteId}',
            description:
              'Agrupa indicadores diarios listados en el archivo compartido (FCR, biomass gain).',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'KPIs diarios',
              metrics: [
                { label: 'FCR', value: '1.32' },
                { label: 'Ganancia biomasa', value: '+0.8% / día' },
                { label: 'Sobrevivencia', value: '98.4%' },
              ],
            },
          },
          {
            label: 'Data Export',
            method: 'POST',
            path: '/analytics/export',
            description: 'Entrega dataset consolidado (CSV) con los campos del archivo demo.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Campos disponibles',
              highlights: [
                'timestamp · siteId · unitId · depth · value',
                'feed_type · ration_plan · oxygen_ppm · biomass_kg',
              ],
            },
          },
        ],
      },
    ],
  }
}
