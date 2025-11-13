package router

import (
	"fmt"
	"sync"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"
)

// Resolver gestiona la resolución de eventos a clientes
type Resolver struct {
	clients      map[string]*ClientState
	index        *SubscriptionIndex
	multiConfigs map[string]*MultiConnectorConfig // key: "tenantID:kind"
	mu           sync.RWMutex
}

// NewResolver crea un nuevo resolver
func NewResolver() *Resolver {
	return &Resolver{
		clients:      make(map[string]*ClientState),
		index:        NewSubscriptionIndex(),
		multiConfigs: make(map[string]*MultiConnectorConfig),
	}
}

// RegisterClient registra un nuevo cliente
func (r *Resolver) RegisterClient(client *ClientState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[client.ClientID]; exists {
		return fmt.Errorf("client %s already registered", client.ClientID)
	}

	r.clients[client.ClientID] = client
	return nil
}

// UnregisterClient elimina un cliente y todas sus suscripciones
func (r *Resolver) UnregisterClient(clientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[clientID]; !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	// Remover todas las suscripciones del cliente
	r.index.RemoveByClient(clientID)
	delete(r.clients, clientID)

	return nil
}

// GetClient retorna el estado de un cliente
func (r *Resolver) GetClient(clientID string) (*ClientState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientID)
	}

	return client, nil
}

// Subscribe crea una nueva suscripción para un cliente
func (r *Resolver) Subscribe(clientID string, filter SubscriptionFilter) (*Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, exists := r.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientID)
	}

	// Crear la suscripción
	sub := &Subscription{
		ID:         fmt.Sprintf("%s-%d", clientID, time.Now().UnixNano()),
		ClientID:   clientID,
		Filter:     filter,
		CreatedAt:  time.Now(),
		EventCount: 0,
	}

	// Agregar al índice
	if err := r.index.Add(sub); err != nil {
		return nil, err
	}

	// Agregar al cliente
	client.Subscriptions = append(client.Subscriptions, sub)

	return sub, nil
}

// Unsubscribe elimina una suscripción
func (r *Resolver) Unsubscribe(subscriptionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Obtener la suscripción
	sub, err := r.index.GetByID(subscriptionID)
	if err != nil {
		return err
	}

	// Remover del índice
	if err := r.index.Remove(subscriptionID); err != nil {
		return err
	}

	// Remover del cliente
	client, exists := r.clients[sub.ClientID]
	if exists {
		for i, s := range client.Subscriptions {
			if s.ID == subscriptionID {
				client.Subscriptions = append(client.Subscriptions[:i], client.Subscriptions[i+1:]...)
				break
			}
		}
	}

	return nil
}

// Resolve resuelve un evento a los clientes que deben recibirlo
func (r *Resolver) Resolve(event *connectors.CanonicalEvent) (*RoutingDecision, error) {
	startTime := time.Now()

	// Encontrar suscripciones que coinciden
	matchedSubs := r.index.FindMatching(event)

	// Agrupar por cliente y verificar permisos
	clientMap := make(map[string]bool)
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, sub := range matchedSubs {
		client, exists := r.clients[sub.ClientID]
		if !exists {
			continue
		}

		// Verificar permisos del cliente
		if r.hasPermission(client, event) {
			clientMap[sub.ClientID] = true

			// Actualizar estadísticas de la suscripción
			r.index.UpdateEventStats(sub.ID)
		}
	}

	// Convertir map a slice
	clients := make([]string, 0, len(clientMap))
	for clientID := range clientMap {
		clients = append(clients, clientID)
	}

	decision := &RoutingDecision{
		Event:       event,
		Clients:     clients,
		Timestamp:   time.Now(),
		ProcessedIn: time.Since(startTime),
	}

	if len(clients) == 0 {
		decision.Reason = "no matching subscriptions or permissions"
	} else {
		decision.Reason = fmt.Sprintf("matched %d subscriptions, %d clients authorized", len(matchedSubs), len(clients))
	}

	return decision, nil
}

// hasPermission verifica si un cliente tiene permiso para recibir un evento
func (r *Resolver) hasPermission(client *ClientState, event *connectors.CanonicalEvent) bool {
	streamKey := &event.Envelope.Stream

	// Verificar tenant
	if client.TenantID != streamKey.TenantID {
		return false
	}

	// Verificar scopes del cliente
	hasAccess := false
	for _, scope := range client.Scopes {
		if scope.CanAccessStream(*streamKey) {
			hasAccess = true

			// Verificar capability requerida
			requiredCap := streamKey.GetCapabilityRequired()
			if requiredCap != "" && !scope.HasCapability(requiredCap) {
				return false
			}
			break
		}
	}

	if !hasAccess {
		return false
	}

	// Verificar capabilities específicas del cliente
	requiredCap := streamKey.GetCapabilityRequired()
	if requiredCap != "" {
		hasCapability := false
		for _, cap := range client.Permissions {
			if cap == requiredCap {
				hasCapability = true
				break
			}
		}
		if !hasCapability {
			return false
		}
	}

	return true
}

// SetMultiConnectorConfig configura políticas multi-conector
func (r *Resolver) SetMultiConnectorConfig(config *MultiConnectorConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", config.TenantID.Hex(), config.Kind)
	r.multiConfigs[key] = config

	return nil
}

// GetMultiConnectorConfig obtiene la configuración multi-conector para un tenant y kind
func (r *Resolver) GetMultiConnectorConfig(tenantID string, kind domain.StreamKind) (*MultiConnectorConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, kind)
	config, exists := r.multiConfigs[key]
	return config, exists
}

// SelectConnector selecciona el conector apropiado según la política
func (r *Resolver) SelectConnector(tenantID string, kind domain.StreamKind) (*ConnectorConfig, error) {
	config, exists := r.GetMultiConnectorConfig(tenantID, kind)
	if !exists {
		return nil, fmt.Errorf("no multi-connector config found for tenant %s, kind %s", tenantID, kind)
	}

	activeConnectors := config.GetActiveConnectors()
	if len(activeConnectors) == 0 {
		return nil, fmt.Errorf("no active connectors available")
	}

	switch config.Policy {
	case PolicyPriority, PolicyFallback:
		// Retornar el de mayor prioridad (ya está ordenado)
		return &activeConnectors[0], nil

	case PolicyRoundRobin:
		// Implementación simple: rotar entre conectores
		// En producción, mantener un contador persistente
		idx := int(time.Now().Unix()) % len(activeConnectors)
		return &activeConnectors[idx], nil

	case PolicyMerge:
		// Para merge, retornar todos (el caller debe manejar esto)
		return &activeConnectors[0], nil

	default:
		return &activeConnectors[0], nil
	}
}

// GetStats retorna estadísticas del resolver
func (r *Resolver) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clientsByTenant := make(map[string]int)
	totalSubs := 0

	for _, client := range r.clients {
		tenantID := client.TenantID.Hex()
		clientsByTenant[tenantID]++
		totalSubs += len(client.Subscriptions)
	}

	stats := map[string]interface{}{
		"active_clients":          len(r.clients),
		"total_subscriptions":     totalSubs,
		"clients_by_tenant":       clientsByTenant,
		"multi_connector_configs": len(r.multiConfigs),
	}

	// Agregar stats del índice
	indexStats := r.index.GetStats()
	for k, v := range indexStats {
		stats["index_"+k] = v
	}

	return stats
}

// GetClientCount retorna el número de clientes activos
func (r *Resolver) GetClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// GetSubscriptionCount retorna el número total de suscripciones
func (r *Resolver) GetSubscriptionCount() int {
	return r.index.Count()
}

// ListClients retorna la lista de todos los clientes
func (r *Resolver) ListClients() []*ClientState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clients := make([]*ClientState, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}

	return clients
}

// UpdateClientPermissions actualiza los permisos de un cliente
func (r *Resolver) UpdateClientPermissions(clientID string, permissions []domain.Capability, scopes []domain.Scope) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, exists := r.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	client.Permissions = permissions
	client.Scopes = scopes

	return nil
}

// ResolveStatus resuelve un evento STATUS a los clientes con IncludeStatus=true
func (r *Resolver) ResolveStatus(event *connectors.CanonicalEvent) (*RoutingDecision, error) {
	startTime := time.Now()

	// Encontrar suscripciones que coinciden y tienen IncludeStatus=true
	matchedSubs := r.index.FindMatchingStatus(event)

	// Agrupar por cliente y verificar permisos
	clientMap := make(map[string]bool)
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, sub := range matchedSubs {
		// Verificar que la suscripción incluye status
		if !sub.IncludeStatus {
			continue
		}

		client, exists := r.clients[sub.ClientID]
		if !exists {
			continue
		}

		// Verificar permisos del cliente
		if r.hasPermission(client, event) {
			clientMap[sub.ClientID] = true

			// Actualizar estadísticas de la suscripción
			r.index.UpdateEventStats(sub.ID)
		}
	}

	// Convertir map a slice
	clients := make([]string, 0, len(clientMap))
	for clientID := range clientMap {
		clients = append(clients, clientID)
	}

	decision := &RoutingDecision{
		Event:       event,
		Clients:     clients,
		Timestamp:   time.Now(),
		ProcessedIn: time.Since(startTime),
	}

	if len(clients) == 0 {
		decision.Reason = "no matching status subscriptions"
	} else {
		decision.Reason = fmt.Sprintf("matched %d status subscriptions, %d clients authorized", len(matchedSubs), len(clients))
	}

	return decision, nil
}
