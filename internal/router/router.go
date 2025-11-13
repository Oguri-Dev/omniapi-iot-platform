package router

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"
	"omniapi/internal/metrics"
	"omniapi/internal/queue/requester"
	"omniapi/internal/queue/status"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Router es el componente principal que coordina el routing de eventos
type Router struct {
	resolver  *Resolver
	throttler *Throttler
	stats     *RouterStats
	eventChan chan *connectors.CanonicalEvent
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex

	// Callbacks para enviar eventos a clientes
	onSendEvent func(clientID string, event *connectors.CanonicalEvent) error
}

// NewRouter crea una nueva instancia del router
func NewRouter() *Router {
	return &Router{
		resolver:  NewResolver(),
		throttler: NewThrottler(),
		stats: &RouterStats{
			EventsByKind:    make(map[string]int64),
			ClientsByTenant: make(map[string]int),
		},
		eventChan: make(chan *connectors.CanonicalEvent, 1000),
		stopChan:  make(chan struct{}),
	}
}

// Start inicia el router
func (r *Router) Start(ctx context.Context) error {
	r.wg.Add(1)
	go r.eventLoop(ctx)
	return nil
}

// Stop detiene el router
func (r *Router) Stop() error {
	close(r.stopChan)
	r.wg.Wait()
	close(r.eventChan)
	return nil
}

// SetEventCallback configura el callback para enviar eventos
func (r *Router) SetEventCallback(callback func(clientID string, event *connectors.CanonicalEvent) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onSendEvent = callback
}

// eventLoop procesa eventos entrantes
func (r *Router) eventLoop(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-r.stopChan:
			return

		case event := <-r.eventChan:
			r.processEvent(event)

		case <-ticker.C:
			// Procesar eventos buffereados periódicamente
			r.processBufferedEvents()
		}
	}
}

// RouteEvent enruta un evento a los clientes apropiados
func (r *Router) RouteEvent(event *connectors.CanonicalEvent) error {
	select {
	case r.eventChan <- event:
		return nil
	default:
		r.mu.Lock()
		r.stats.EventsDropped++
		r.mu.Unlock()
		return fmt.Errorf("event channel full, event dropped")
	}
}

// processEvent procesa un evento individual
func (r *Router) processEvent(event *connectors.CanonicalEvent) {
	startTime := time.Now()

	// Resolver a qué clientes debe enviarse
	decision, err := r.resolver.Resolve(event)
	if err != nil {
		return
	}

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.EventsRouted++
	r.stats.EventsByKind[event.Kind]++
	r.mu.Unlock()

	// Actualizar métricas de Prometheus para eventos DATA
	r.updateDataMetrics(event)

	// Enviar a cada cliente
	for _, clientID := range decision.Clients {
		client, err := r.resolver.GetClient(clientID)
		if err != nil {
			continue
		}

		// Aplicar throttling
		canSend, reason := r.throttler.ProcessEvent(clientID, event, client)

		if canSend {
			r.sendToClient(clientID, event, client)
		} else if reason == "buffered" {
			// El evento fue buffereado, se enviará después
			continue
		} else {
			// Evento descartado
			r.mu.Lock()
			r.stats.EventsDropped++
			r.mu.Unlock()
			metrics.EventsDroppedTotal.Inc()
		}
	}

	// Registrar tiempo de routing
	durationMs := float64(time.Since(startTime).Microseconds()) / 1000.0
	r.mu.Lock()
	r.stats.RecordRoutingTime(durationMs)
	r.mu.Unlock()
}

// processBufferedEvents procesa eventos que están en buffers
func (r *Router) processBufferedEvents() {
	clients := r.resolver.ListClients()

	for _, client := range clients {
		events := r.throttler.GetPendingEvents(client.ClientID, client)

		// Aplicar coalescing si está habilitado
		if client.ThrottleConfig.CoalescingEnabled {
			events = r.throttler.CoalesceEvents(events)
		}

		// Enviar eventos pendientes
		for _, event := range events {
			r.sendToClient(client.ClientID, event, client)
		}
	}
}

// sendToClient envía un evento a un cliente específico
func (r *Router) sendToClient(clientID string, event *connectors.CanonicalEvent, client *ClientState) {
	if r.onSendEvent == nil {
		return
	}

	err := r.onSendEvent(clientID, event)
	if err != nil {
		client.Stats.EventsDropped++
		return
	}

	// Actualizar estadísticas del cliente
	client.Stats.EventsSent++
	client.Stats.EventsReceived++
	client.LastEvent = time.Now()

	// Actualizar estadísticas globales
	r.mu.Lock()
	r.stats.TotalBytesRouted += int64(len(event.Payload))
	r.mu.Unlock()
}

// RegisterClient registra un nuevo cliente en el router
func (r *Router) RegisterClient(
	clientID string,
	tenantID primitive.ObjectID,
	permissions []domain.Capability,
	scopes []domain.Scope,
	throttleConfig *ThrottleConfig,
) error {
	// Crear estado del cliente
	client := NewClientState(clientID, tenantID)
	client.Permissions = permissions
	client.Scopes = scopes

	if throttleConfig != nil {
		client.ThrottleConfig = *throttleConfig
	}

	// Registrar en resolver
	if err := r.resolver.RegisterClient(client); err != nil {
		return err
	}

	// Registrar en throttler
	r.throttler.RegisterClient(clientID, client.ThrottleConfig)

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.ActiveClients++
	r.stats.ClientsByTenant[tenantID.Hex()]++
	r.mu.Unlock()

	return nil
}

// UnregisterClient elimina un cliente del router
func (r *Router) UnregisterClient(clientID string) error {
	// Obtener cliente antes de eliminar para estadísticas
	client, err := r.resolver.GetClient(clientID)
	if err != nil {
		return err
	}

	// Eliminar del resolver
	if err := r.resolver.UnregisterClient(clientID); err != nil {
		return err
	}

	// Eliminar del throttler
	r.throttler.UnregisterClient(clientID)

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.ActiveClients--
	tenantID := client.TenantID.Hex()
	if count, exists := r.stats.ClientsByTenant[tenantID]; exists {
		r.stats.ClientsByTenant[tenantID] = count - 1
		if r.stats.ClientsByTenant[tenantID] <= 0 {
			delete(r.stats.ClientsByTenant, tenantID)
		}
	}
	r.mu.Unlock()

	return nil
}

// Subscribe crea una suscripción para un cliente
func (r *Router) Subscribe(clientID string, filter SubscriptionFilter) (*Subscription, error) {
	sub, err := r.resolver.Subscribe(clientID, filter)
	if err != nil {
		return nil, err
	}

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.ActiveSubscriptions++
	r.mu.Unlock()

	return sub, nil
}

// Unsubscribe elimina una suscripción
func (r *Router) Unsubscribe(subscriptionID string) error {
	err := r.resolver.Unsubscribe(subscriptionID)
	if err != nil {
		return err
	}

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.ActiveSubscriptions--
	r.mu.Unlock()

	return nil
}

// UpdateClientThrottle actualiza la configuración de throttle de un cliente
func (r *Router) UpdateClientThrottle(clientID string, config ThrottleConfig) error {
	client, err := r.resolver.GetClient(clientID)
	if err != nil {
		return err
	}

	client.ThrottleConfig = config
	return r.throttler.UpdateConfig(clientID, config)
}

// SetMultiConnectorPolicy configura la política multi-conector
func (r *Router) SetMultiConnectorPolicy(config *MultiConnectorConfig) error {
	return r.resolver.SetMultiConnectorConfig(config)
}

// GetMultiConnectorPolicy obtiene la política multi-conector
func (r *Router) GetMultiConnectorPolicy(tenantID string, kind domain.StreamKind) (*MultiConnectorConfig, bool) {
	return r.resolver.GetMultiConnectorConfig(tenantID, kind)
}

// SelectConnector selecciona el mejor conector según la política
func (r *Router) SelectConnector(tenantID string, kind domain.StreamKind) (*ConnectorConfig, error) {
	return r.resolver.SelectConnector(tenantID, kind)
}

// GetStats retorna estadísticas del router
func (r *Router) GetStats() *RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Crear copia para evitar race conditions
	stats := &RouterStats{
		EventsRouted:        r.stats.EventsRouted,
		EventsDropped:       r.stats.EventsDropped,
		EventsDataOut:       r.stats.EventsDataOut,
		EventsStatusOut:     r.stats.EventsStatusOut,
		ActiveClients:       r.stats.ActiveClients,
		ActiveSubscriptions: r.stats.ActiveSubscriptions,
		TotalBytesRouted:    r.stats.TotalBytesRouted,
		AvgRoutingTimeMs:    r.stats.AvgRoutingTimeMs,
		RouteP95Ms:          r.stats.RouteP95Ms,
		EventsByKind:        make(map[string]int64),
		ClientsByTenant:     make(map[string]int),
	}

	for k, v := range r.stats.EventsByKind {
		stats.EventsByKind[k] = v
	}

	for k, v := range r.stats.ClientsByTenant {
		stats.ClientsByTenant[k] = v
	}

	return stats
}

// GetClientStats retorna estadísticas de un cliente específico
func (r *Router) GetClientStats(clientID string) (map[string]interface{}, error) {
	client, err := r.resolver.GetClient(clientID)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"client_id":       client.ClientID,
		"tenant_id":       client.TenantID.Hex(),
		"subscriptions":   len(client.Subscriptions),
		"events_sent":     client.Stats.EventsSent,
		"events_received": client.Stats.EventsReceived,
		"events_dropped":  client.Stats.EventsDropped,
		"throttled":       client.Stats.Throttled,
		"bytes_sent":      client.Stats.BytesSent,
		"last_event":      client.LastEvent,
		"throttle_config": client.ThrottleConfig,
	}

	// Agregar stats del throttler
	throttleStats := r.throttler.GetClientStats(clientID)
	stats["throttle_state"] = throttleStats

	return stats, nil
}

// updateAvgRoutingTime actualiza el tiempo promedio de routing
func (r *Router) updateAvgRoutingTime(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Usar EWMA (Exponentially Weighted Moving Average)
	alpha := 0.1
	newValue := duration.Seconds() * 1000 // Convertir a ms

	if r.stats.AvgRoutingTimeMs == 0 {
		r.stats.AvgRoutingTimeMs = newValue
	} else {
		r.stats.AvgRoutingTimeMs = alpha*newValue + (1-alpha)*r.stats.AvgRoutingTimeMs
	}
}

// GetClient retorna el estado de un cliente
func (r *Router) GetClient(clientID string) (*ClientState, error) {
	return r.resolver.GetClient(clientID)
}

// ListClients retorna todos los clientes activos
func (r *Router) ListClients() []*ClientState {
	return r.resolver.ListClients()
}

// UpdateClientPermissions actualiza permisos de un cliente
func (r *Router) UpdateClientPermissions(clientID string, permissions []domain.Capability, scopes []domain.Scope) error {
	return r.resolver.UpdateClientPermissions(clientID, permissions, scopes)
}

// OnRequesterResult maneja resultados del requester y los transforma en eventos DATA
func (r *Router) OnRequesterResult(result requester.Result) {
	// Convertir TenantID string a ObjectID
	tenantOID, err := primitive.ObjectIDFromHex(result.TenantID)
	if err != nil {
		// Si no es un ObjectID válido, intentar parsearlo como hex
		tenantOID = primitive.NilObjectID
	}

	// Construir StreamKey
	streamKey := domain.StreamKey{
		TenantID: tenantOID,
		Kind:     domain.StreamKind(result.Metric),
		SiteID:   result.SiteID,
	}

	if result.CageID != nil && *result.CageID != "" {
		streamKey.CageID = result.CageID
	}

	// Construir envelope
	envelope := connectors.Envelope{
		Version:   "1.0",
		Timestamp: result.CompletedAt,
		Stream:    streamKey,
		Source:    string(result.Source),
		Sequence:  0, // TODO: mantener secuencia
		Flags:     connectors.EventFlagNone,
	}

	// Si hubo error, marcar como sintético
	if result.Err != nil {
		envelope.Flags = connectors.EventFlagSynthetic
	}

	// Construir payload
	payload := map[string]interface{}{
		"metric":     result.Metric,
		"time_range": result.TsRange,
		"latency_ms": result.LatencyMS,
	}

	if result.IsSuccess() {
		payload["data"] = result.Payload
		payload["status"] = "success"
	} else {
		payload["error"] = result.ErrorMsg
		payload["status"] = "error"
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Construir evento canónico
	event := &connectors.CanonicalEvent{
		Envelope:      envelope,
		Payload:       payloadBytes,
		Kind:          result.Metric,
		SchemaVersion: "1.0",
	}

	// Enrutar evento
	r.RouteEvent(event)

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.EventsDataOut++
	r.mu.Unlock()
}

// OnStatusHeartbeat maneja heartbeats de estado y los transforma en eventos STATUS
func (r *Router) OnStatusHeartbeat(st status.Status) {
	// Convertir TenantID string a ObjectID
	tenantOID, err := primitive.ObjectIDFromHex(st.TenantID)
	if err != nil {
		// Si no es un ObjectID válido, intentar parsearlo como hex
		tenantOID = primitive.NilObjectID
	}

	// Construir StreamKey
	streamKey := domain.StreamKey{
		TenantID: tenantOID,
		Kind:     domain.StreamKind("status"),
		SiteID:   st.SiteID,
	}

	if st.CageID != nil {
		streamKey.CageID = st.CageID
	}

	// Construir envelope
	envelope := connectors.Envelope{
		Version:   "1.0",
		Timestamp: st.EmittedAt,
		Stream:    streamKey,
		Source:    st.Source,
		Sequence:  0,                             // TODO: mantener secuencia
		Flags:     connectors.EventFlagSynthetic, // Los heartbeats son sintéticos
	}

	// Construir payload de status
	statusPayload := map[string]interface{}{
		"metric":        st.Metric,
		"source":        st.Source,
		"state":         st.State,
		"staleness_sec": st.StalenessSec,
		"in_flight":     st.InFlight,
	}

	if st.LastSuccessTS != nil {
		statusPayload["last_success"] = st.LastSuccessTS
	}

	if st.LastErrorTS != nil {
		statusPayload["last_error"] = st.LastErrorTS
	}

	if st.LastErrorMsg != nil {
		statusPayload["last_error_msg"] = *st.LastErrorMsg
	}

	if st.LastLatencyMS != nil {
		statusPayload["last_latency_ms"] = *st.LastLatencyMS
	}

	if st.Notes != nil {
		statusPayload["notes"] = *st.Notes
	}

	payloadBytes, err := json.Marshal(statusPayload)
	if err != nil {
		return
	}

	// Construir evento canónico de tipo STATUS
	event := &connectors.CanonicalEvent{
		Envelope:      envelope,
		Payload:       payloadBytes,
		Kind:          "status." + st.Metric, // Prefijo "status." para distinguir
		SchemaVersion: "1.0",
	}

	// Enrutar solo a suscriptores con IncludeStatus=true
	r.RouteStatusEvent(event)

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.EventsStatusOut++
	r.mu.Unlock()

	// Actualizar métricas de Prometheus
	r.updateStatusMetrics(st)
}

// RouteStatusEvent enruta un evento STATUS solo a suscriptores que lo soliciten
func (r *Router) RouteStatusEvent(event *connectors.CanonicalEvent) error {
	startTime := time.Now()

	// Resolver a qué clientes debe enviarse
	decision, err := r.resolver.ResolveStatus(event)
	if err != nil {
		return err
	}

	// Actualizar estadísticas
	r.mu.Lock()
	r.stats.EventsRouted++
	r.stats.EventsByKind[event.Kind]++
	r.mu.Unlock()

	// Enviar a cada cliente
	for _, clientID := range decision.Clients {
		client, err := r.resolver.GetClient(clientID)
		if err != nil {
			continue
		}

		// Verificar que el cliente quiere status
		hasStatusSub := false
		for _, sub := range client.Subscriptions {
			if sub.IncludeStatus && sub.Filter.Matches(event) {
				hasStatusSub = true
				break
			}
		}

		if !hasStatusSub {
			continue
		}

		// Enviar directamente (sin throttle para status)
		r.sendToClient(clientID, event, client)
	}

	// Registrar tiempo de routing
	durationMs := float64(time.Since(startTime).Microseconds()) / 1000.0
	r.mu.Lock()
	r.stats.RecordRoutingTime(durationMs)
	r.mu.Unlock()

	return nil
}

// updateDataMetrics actualiza métricas de Prometheus para eventos DATA
func (r *Router) updateDataMetrics(event *connectors.CanonicalEvent) {
	// Extraer labels del envelope
	tenant := metrics.SanitizeTenantID(event.Envelope.Stream.TenantID.Hex())
	site := metrics.SanitizeSiteID(event.Envelope.Stream.SiteID)
	metric := metrics.SanitizeMetric(event.Kind)

	// Incrementar contador de eventos DATA in (recibidos de requester)
	metrics.EventsDataInTotal.WithLabelValues(tenant, site, metric).Inc()

	// Incrementar contador de eventos DATA out (enviados a clientes)
	// Esto se hace por cada cliente en sendToClient, pero aquí contamos el total
	metrics.EventsDataOutTotal.WithLabelValues(tenant, site, metric).Inc()
}

// updateStatusMetrics actualiza métricas de Prometheus para eventos STATUS
func (r *Router) updateStatusMetrics(st status.Status) {
	tenant := metrics.SanitizeTenantID(st.TenantID)
	site := metrics.SanitizeSiteID(st.SiteID)
	metric := metrics.SanitizeMetric(st.Metric)

	// Incrementar contador de eventos STATUS out
	metrics.EventsStatusOutTotal.WithLabelValues(tenant, site, metric).Inc()
}
