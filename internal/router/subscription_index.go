package router

import (
	"fmt"
	"sync"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SubscriptionIndex indexa suscripciones para búsqueda rápida
type SubscriptionIndex struct {
	// Índices por diferentes criterios
	byClient map[string][]*Subscription             // ClientID -> Subscriptions
	byTenant map[primitive.ObjectID][]*Subscription // TenantID -> Subscriptions
	byKind   map[domain.StreamKind][]*Subscription  // Kind -> Subscriptions
	bySite   map[string][]*Subscription             // SiteID -> Subscriptions
	byCage   map[string][]*Subscription             // CageID -> Subscriptions
	byFarm   map[string][]*Subscription             // FarmID -> Subscriptions
	all      map[string]*Subscription               // SubscriptionID -> Subscription

	mu sync.RWMutex
}

// NewSubscriptionIndex crea un nuevo índice de suscripciones
func NewSubscriptionIndex() *SubscriptionIndex {
	return &SubscriptionIndex{
		byClient: make(map[string][]*Subscription),
		byTenant: make(map[primitive.ObjectID][]*Subscription),
		byKind:   make(map[domain.StreamKind][]*Subscription),
		bySite:   make(map[string][]*Subscription),
		byCage:   make(map[string][]*Subscription),
		byFarm:   make(map[string][]*Subscription),
		all:      make(map[string]*Subscription),
	}
}

// Add agrega una suscripción al índice
func (si *SubscriptionIndex) Add(sub *Subscription) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	if _, exists := si.all[sub.ID]; exists {
		return fmt.Errorf("subscription %s already exists", sub.ID)
	}

	// Indexar en todos los mapas relevantes
	si.all[sub.ID] = sub
	si.byClient[sub.ClientID] = append(si.byClient[sub.ClientID], sub)

	// Indexar por TenantID si existe
	if sub.Filter.TenantID != nil {
		si.byTenant[*sub.Filter.TenantID] = append(si.byTenant[*sub.Filter.TenantID], sub)
	}

	// Indexar por Kind si existe
	if sub.Filter.Kind != nil {
		si.byKind[*sub.Filter.Kind] = append(si.byKind[*sub.Filter.Kind], sub)
	}

	// Indexar por FarmID si existe
	if sub.Filter.FarmID != nil {
		si.byFarm[*sub.Filter.FarmID] = append(si.byFarm[*sub.Filter.FarmID], sub)
	}

	// Indexar por SiteID si existe
	if sub.Filter.SiteID != nil {
		si.bySite[*sub.Filter.SiteID] = append(si.bySite[*sub.Filter.SiteID], sub)
	}

	// Indexar por CageID si existe
	if sub.Filter.CageID != nil {
		si.byCage[*sub.Filter.CageID] = append(si.byCage[*sub.Filter.CageID], sub)
	}

	return nil
}

// Remove elimina una suscripción del índice
func (si *SubscriptionIndex) Remove(subscriptionID string) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	sub, exists := si.all[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	// Remover de todos los índices
	delete(si.all, subscriptionID)
	si.byClient[sub.ClientID] = si.removeFromSlice(si.byClient[sub.ClientID], subscriptionID)

	if sub.Filter.TenantID != nil {
		si.byTenant[*sub.Filter.TenantID] = si.removeFromSlice(si.byTenant[*sub.Filter.TenantID], subscriptionID)
	}

	if sub.Filter.Kind != nil {
		si.byKind[*sub.Filter.Kind] = si.removeFromSlice(si.byKind[*sub.Filter.Kind], subscriptionID)
	}

	if sub.Filter.FarmID != nil {
		si.byFarm[*sub.Filter.FarmID] = si.removeFromSlice(si.byFarm[*sub.Filter.FarmID], subscriptionID)
	}

	if sub.Filter.SiteID != nil {
		si.bySite[*sub.Filter.SiteID] = si.removeFromSlice(si.bySite[*sub.Filter.SiteID], subscriptionID)
	}

	if sub.Filter.CageID != nil {
		si.byCage[*sub.Filter.CageID] = si.removeFromSlice(si.byCage[*sub.Filter.CageID], subscriptionID)
	}

	return nil
}

// removeFromSlice elimina una suscripción de un slice y retorna el nuevo slice
func (si *SubscriptionIndex) removeFromSlice(subs []*Subscription, subID string) []*Subscription {
	for i, sub := range subs {
		if sub.ID == subID {
			return append(subs[:i], subs[i+1:]...)
		}
	}
	return subs
}

// RemoveByClient elimina todas las suscripciones de un cliente
func (si *SubscriptionIndex) RemoveByClient(clientID string) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	subs, exists := si.byClient[clientID]
	if !exists || len(subs) == 0 {
		return nil // No hay suscripciones para este cliente
	}

	// Obtener lista de IDs antes de modificar
	subIDs := make([]string, len(subs))
	for i, sub := range subs {
		subIDs[i] = sub.ID
	}

	// Desbloquear temporalmente y usar Remove para cada una
	si.mu.Unlock()
	for _, subID := range subIDs {
		si.Remove(subID)
	}
	si.mu.Lock()

	return nil
}

// GetByClient retorna todas las suscripciones de un cliente
func (si *SubscriptionIndex) GetByClient(clientID string) []*Subscription {
	si.mu.RLock()
	defer si.mu.RUnlock()

	subs := si.byClient[clientID]
	result := make([]*Subscription, len(subs))
	copy(result, subs)
	return result
}

// GetByID retorna una suscripción por su ID
func (si *SubscriptionIndex) GetByID(subscriptionID string) (*Subscription, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	sub, exists := si.all[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}

	return sub, nil
}

// FindMatching encuentra todas las suscripciones que coinciden con un evento
func (si *SubscriptionIndex) FindMatching(event *connectors.CanonicalEvent) []*Subscription {
	si.mu.RLock()
	defer si.mu.RUnlock()

	streamKey := event.Envelope.Stream

	// Conjunto para evitar duplicados
	matched := make(map[string]*Subscription)

	// Estrategia: buscar en los índices más específicos primero
	// para reducir el número de comparaciones

	// 1. Por CageID (más específico)
	if streamKey.CageID != nil && *streamKey.CageID != "" {
		for _, sub := range si.byCage[*streamKey.CageID] {
			if sub.Filter.Matches(event) {
				matched[sub.ID] = sub
			}
		}
	}

	// 2. Por SiteID
	for _, sub := range si.bySite[streamKey.SiteID] {
		if sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 3. Por FarmID
	for _, sub := range si.byFarm[streamKey.FarmID] {
		if sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 4. Por Kind
	for _, sub := range si.byKind[streamKey.Kind] {
		if sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 5. Por TenantID
	for _, sub := range si.byTenant[streamKey.TenantID] {
		if sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 6. Verificar suscripciones sin filtros específicos (wildcard)
	for _, sub := range si.all {
		if _, alreadyMatched := matched[sub.ID]; !alreadyMatched {
			// Verificar si es una suscripción wildcard que aplica
			if si.isWildcardMatch(sub, event) {
				matched[sub.ID] = sub
			}
		}
	}

	// Convertir map a slice
	result := make([]*Subscription, 0, len(matched))
	for _, sub := range matched {
		result = append(result, sub)
	}

	return result
}

// isWildcardMatch verifica si una suscripción sin filtros específicos coincide
func (si *SubscriptionIndex) isWildcardMatch(sub *Subscription, event *connectors.CanonicalEvent) bool {
	// Si tiene filtros específicos, ya fue verificada en los índices
	if sub.Filter.TenantID != nil || sub.Filter.Kind != nil ||
		sub.Filter.FarmID != nil || sub.Filter.SiteID != nil ||
		sub.Filter.CageID != nil {
		return false
	}

	// Es un wildcard, verificar match general
	return sub.Filter.Matches(event)
}

// FindMatchingStatus encuentra suscripciones que coinciden con un evento STATUS
// Solo retorna suscripciones con IncludeStatus=true
func (si *SubscriptionIndex) FindMatchingStatus(event *connectors.CanonicalEvent) []*Subscription {
	si.mu.RLock()
	defer si.mu.RUnlock()

	streamKey := event.Envelope.Stream
	matched := make(map[string]*Subscription)

	// Buscar en todos los índices similiar a FindMatching, pero filtrar por IncludeStatus
	// 1. Por CageID
	if streamKey.CageID != nil {
		for _, sub := range si.byCage[*streamKey.CageID] {
			if sub.IncludeStatus && sub.Filter.Matches(event) {
				matched[sub.ID] = sub
			}
		}
	}

	// 2. Por SiteID
	for _, sub := range si.bySite[streamKey.SiteID] {
		if sub.IncludeStatus && sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 3. Por FarmID
	for _, sub := range si.byFarm[streamKey.FarmID] {
		if sub.IncludeStatus && sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 4. Por Kind
	for _, sub := range si.byKind[streamKey.Kind] {
		if sub.IncludeStatus && sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 5. Por TenantID
	for _, sub := range si.byTenant[streamKey.TenantID] {
		if sub.IncludeStatus && sub.Filter.Matches(event) {
			matched[sub.ID] = sub
		}
	}

	// 6. Verificar suscripciones wildcard con IncludeStatus
	for _, sub := range si.all {
		if !sub.IncludeStatus {
			continue
		}

		if _, alreadyMatched := matched[sub.ID]; !alreadyMatched {
			if si.isWildcardMatch(sub, event) {
				matched[sub.ID] = sub
			}
		}
	}

	// Convertir map a slice
	result := make([]*Subscription, 0, len(matched))
	for _, sub := range matched {
		result = append(result, sub)
	}

	return result
}

// UpdateEventStats actualiza las estadísticas de evento para una suscripción
func (si *SubscriptionIndex) UpdateEventStats(subscriptionID string) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	sub, exists := si.all[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	now := time.Now()
	sub.LastEvent = &now
	sub.EventCount++

	return nil
}

// GetStats retorna estadísticas del índice
func (si *SubscriptionIndex) GetStats() map[string]interface{} {
	si.mu.RLock()
	defer si.mu.RUnlock()

	return map[string]interface{}{
		"total_subscriptions": len(si.all),
		"clients":             len(si.byClient),
		"tenants":             len(si.byTenant),
		"kinds":               len(si.byKind),
		"sites":               len(si.bySite),
		"cages":               len(si.byCage),
		"farms":               len(si.byFarm),
	}
}

// Count retorna el número total de suscripciones
func (si *SubscriptionIndex) Count() int {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return len(si.all)
}

// Clear limpia todos los índices
func (si *SubscriptionIndex) Clear() {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.byClient = make(map[string][]*Subscription)
	si.byTenant = make(map[primitive.ObjectID][]*Subscription)
	si.byKind = make(map[domain.StreamKind][]*Subscription)
	si.bySite = make(map[string][]*Subscription)
	si.byCage = make(map[string][]*Subscription)
	si.byFarm = make(map[string][]*Subscription)
	si.all = make(map[string]*Subscription)
}
