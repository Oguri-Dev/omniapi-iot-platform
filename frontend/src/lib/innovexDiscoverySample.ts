import type { DiscoveryResult } from '../types/discovery'

export interface InnovexDiscoveryContext {
  siteName: string
  siteCode: string
  tenantCode?: string
  monitorId: string
  meditionSample?: string
}

const nowIso = () => new Date().toISOString()

export const buildInnovexDiscoverySample = (context: InnovexDiscoveryContext): DiscoveryResult => {
  const generatedAt = nowIso()

  return {
    provider: 'innovex',
    siteId: context.monitorId,
    siteName: context.siteName,
    tenantCode: context.tenantCode,
    generatedAt,
    headersUsed: {
      Authorization: 'Bearer <token-simulado>',
      'Content-Type': 'application/json',
    },
    summary: {
      timeseriesCount: 62,
      metricsAvailable: 9,
      range: 'Últimas 48h · Respuesta Innovex Dataweb',
    },
    groups: [
      {
        id: 'monitors',
        title: 'Catálogo de Monitores',
        description:
          'Recursos para obtener la lista de centros disponibles y sus sensores asociados.',
        endpoints: [
          {
            label: 'All Monitors',
            method: 'GET',
            path: '/api_dataweb/all_monitors/?active=all',
            description:
              'Devuelve todos los monitores asociados al cliente, incluyendo lat/lon y monitor_key.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Monitores registrados',
              metrics: [
                { label: 'Monitores activos', value: '14', accent: 'success' },
                { label: 'Monitor destacado', value: context.siteName },
                { label: 'Monitor Key', value: context.monitorId, accent: 'info' },
              ],
            },
          },
          {
            label: 'Monitor Detail',
            method: 'GET',
            path: `/api_dataweb/monitor_detail/?monitor_id=${context.monitorId}`,
            description:
              'Lista loggers y sensores del monitor, incluyendo jaulas, módulos y profundidad.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: `Sensores registrados ${context.siteName}`,
              highlights: [
                'Muestra logger y sensores con jaula/silo asociado',
                'Incluye depth_detail.quantity y measurement',
              ],
            },
          },
        ],
      },
      {
        id: 'latest',
        title: 'Últimas Lecturas',
        description:
          'Lecturas frescas de sensores por monitor y tipo de medición (oxygen, flow, weather, etc.).',
        endpoints: [
          {
            label: 'Monitor Sensor Last Data',
            method: 'GET',
            path: `/api_dataweb/monitor_sensor_last_data/?id=${context.monitorId}&medition=${
              context.meditionSample || 'oxygen'
            }`,
            description:
              'Retorna la última medición por sensor con temperatura, saturación y salinidad.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Oxygen snapshot',
              metrics: [
                { label: 'Oxygen', value: '7.4 mg/L', accent: 'info' },
                { label: 'Temperature', value: '9.8 °C' },
                { label: 'Depth', value: '12 m' },
              ],
              highlights: ['Incluye timestamp Unix y detalle de jaula/modulo'],
            },
          },
          {
            label: 'Get Last Data (sensor)',
            method: 'GET',
            path: '/api_dataweb/get_last_data/?monitor_id={monitor_id}&sensor_id={sensor_id}',
            description:
              'Trae la última medición puntual de un sensor específico, ideal para alertas.',
            lastSync: generatedAt,
            availability: 'partial',
            dataset: {
              title: 'Campos devueltos',
              table: {
                columns: ['Campo', 'Descripción'],
                rows: [
                  { Campo: 'timestamp', Descripción: 'UnixTime UTC' },
                  { Campo: 'oxygen', Descripción: 'mg/L' },
                  { Campo: 'temperature', Descripción: '°C' },
                  { Campo: 'salinity', Descripción: 'ppt' },
                  { Campo: 'depth', Descripción: 'm' },
                ],
              },
            },
          },
        ],
      },
      {
        id: 'ranges',
        title: 'Series Históricas',
        description:
          'Consultas para rangos de tiempo limitados (hasta 30 días) por monitor/sensor.',
        endpoints: [
          {
            label: 'Get Data Range',
            method: 'GET',
            path: '/api_dataweb/get_data_range/?monitor_id={monitor_id}&sensor_id={sensor_id}&unixtime_since=...&unixtime_until=...',
            description: 'Devuelve lecturas en un rango definido, útil para dashboards diarios.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Ejemplo de ventana',
              metrics: [
                { label: 'Desde', value: '03/10/2017 00:00' },
                { label: 'Hasta', value: '04/10/2017 00:00' },
                { label: 'Intervalos', value: '5 min' },
              ],
            },
          },
          {
            label: 'Monitor Sensor Time Data',
            method: 'GET',
            path: '/api_dataweb/monitor_sensor_time_data/?monitor_id={monitor_id}&medition=oxygen&unixtime_since=...&unixtime_until=...',
            description:
              'Rango con agrupación por sensor para un tipo de medición completo (oxygen, flow, weather).',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Campos oxygen',
              highlights: [
                'Entrega arrays de oxygen, saturation, temperature, salinity y depth',
                'Incluye metadata de logger y jaula',
              ],
            },
          },
        ],
      },
      {
        id: 'errors',
        title: 'Códigos de error Innovex',
        description:
          'Listado de respuestas comunes para manejar errores (Access Denied, Method Not Allowed, etc.).',
        endpoints: [
          {
            label: 'Error Responses',
            method: 'GET',
            path: 'documentado en manual (tabla de errores)',
            description:
              'Respuestas JSON: Unauthorized sensor information, Miss information, Internal error, Not found, Method Not Allowed, Access Denied, Time out.',
            lastSync: generatedAt,
            availability: 'ready',
            dataset: {
              title: 'Errores frecuentes',
              highlights: [
                "{'response': 'Unauthorized sensor information'}",
                "{'response': 'Access Denied'}",
                "{'response': 'time out'}",
              ],
            },
          },
        ],
      },
    ],
  }
}
