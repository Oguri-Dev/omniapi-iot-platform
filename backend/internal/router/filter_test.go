package router

import (
	"encoding/json"
	"testing"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSubscriptionFilter_Matches(t *testing.T) {
	tenantID := primitive.NewObjectID()
	kind := domain.StreamKindFeeding
	farmID := "farm-001"
	siteID := "site-001"
	cageID := "cage-001"

	tests := []struct {
		name     string
		filter   SubscriptionFilter
		event    *connectors.CanonicalEvent
		expected bool
	}{
		{
			name: "Match by TenantID only",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: true,
		},
		{
			name: "Match by TenantID and Kind",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Kind:     &kind,
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: true,
		},
		{
			name: "Match by TenantID, Kind, and FarmID",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Kind:     &kind,
				FarmID:   &farmID,
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: true,
		},
		{
			name: "Match by TenantID, Kind, FarmID, and SiteID",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Kind:     &kind,
				FarmID:   &farmID,
				SiteID:   &siteID,
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: true,
		},
		{
			name: "Match all including CageID",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Kind:     &kind,
				FarmID:   &farmID,
				SiteID:   &siteID,
				CageID:   &cageID,
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: true,
		},
		{
			name: "No match - different TenantID",
			filter: SubscriptionFilter{
				TenantID: func() *primitive.ObjectID { id := primitive.NewObjectID(); return &id }(),
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: false,
		},
		{
			name: "No match - different Kind",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Kind:     func() *domain.StreamKind { k := domain.StreamKindBiometric; return &k }(),
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: false,
		},
		{
			name: "No match - different SiteID",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Kind:     &kind,
				SiteID:   func() *string { s := "site-999"; return &s }(),
			},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: false,
		},
		{
			name: "Match by Source",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Sources:  []string{"connector-1", "connector-2"},
			},
			event:    createTestEventWithSource(tenantID, kind, farmID, siteID, &cageID, "connector-1"),
			expected: true,
		},
		{
			name: "No match - different Source",
			filter: SubscriptionFilter{
				TenantID: &tenantID,
				Sources:  []string{"connector-1", "connector-2"},
			},
			event:    createTestEventWithSource(tenantID, kind, farmID, siteID, &cageID, "connector-3"),
			expected: false,
		},
		{
			name:     "Match wildcard (no filters)",
			filter:   SubscriptionFilter{},
			event:    createTestEvent(tenantID, kind, farmID, siteID, &cageID),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(tt.event)
			if result != tt.expected {
				t.Errorf("Filter.Matches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSubscriptionIndex_AddAndFind(t *testing.T) {
	index := NewSubscriptionIndex()
	tenantID := primitive.NewObjectID()
	kind := domain.StreamKindFeeding
	farmID := "farm-001"
	siteID := "site-001"
	cageID := "cage-001"

	// Crear suscripciones
	sub1 := &Subscription{
		ID:       "sub-1",
		ClientID: "client-1",
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
			Kind:     &kind,
		},
	}

	sub2 := &Subscription{
		ID:       "sub-2",
		ClientID: "client-2",
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
			Kind:     &kind,
			SiteID:   &siteID,
		},
	}

	sub3 := &Subscription{
		ID:       "sub-3",
		ClientID: "client-3",
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
			Kind:     &kind,
			SiteID:   &siteID,
			CageID:   &cageID,
		},
	}

	// Agregar suscripciones
	if err := index.Add(sub1); err != nil {
		t.Fatalf("Failed to add sub1: %v", err)
	}
	if err := index.Add(sub2); err != nil {
		t.Fatalf("Failed to add sub2: %v", err)
	}
	if err := index.Add(sub3); err != nil {
		t.Fatalf("Failed to add sub3: %v", err)
	}

	// Test: Encontrar por evento
	event := createTestEvent(tenantID, kind, farmID, siteID, &cageID)
	matched := index.FindMatching(event)

	if len(matched) != 3 {
		t.Errorf("Expected 3 matching subscriptions, got %d", len(matched))
	}

	// Verificar que todas las suscripciones coinciden
	matchedIDs := make(map[string]bool)
	for _, sub := range matched {
		matchedIDs[sub.ID] = true
	}

	if !matchedIDs["sub-1"] || !matchedIDs["sub-2"] || !matchedIDs["sub-3"] {
		t.Errorf("Not all expected subscriptions matched")
	}
}

func TestSubscriptionIndex_RemoveByClient(t *testing.T) {
	index := NewSubscriptionIndex()
	tenantID := primitive.NewObjectID()
	kind := domain.StreamKindFeeding

	// Agregar múltiples suscripciones para el mismo cliente
	clientID := "client-1"
	for i := 0; i < 3; i++ {
		sub := &Subscription{
			ID:       primitive.NewObjectID().Hex(),
			ClientID: clientID,
			Filter: SubscriptionFilter{
				TenantID: &tenantID,
				Kind:     &kind,
			},
		}
		if err := index.Add(sub); err != nil {
			t.Fatalf("Failed to add subscription: %v", err)
		}
	}

	// Verificar que se agregaron
	if index.Count() != 3 {
		t.Errorf("Expected 3 subscriptions, got %d", index.Count())
	}

	// Remover todas las suscripciones del cliente
	if err := index.RemoveByClient(clientID); err != nil {
		t.Fatalf("Failed to remove client subscriptions: %v", err)
	}

	// Verificar que se eliminaron
	if index.Count() != 0 {
		t.Errorf("Expected 0 subscriptions after removal, got %d", index.Count())
	}
}

func TestSubscriptionIndex_GetByClient(t *testing.T) {
	index := NewSubscriptionIndex()
	tenantID := primitive.NewObjectID()
	kind := domain.StreamKindFeeding

	client1 := "client-1"
	client2 := "client-2"

	// Agregar suscripciones para diferentes clientes
	sub1 := &Subscription{
		ID:       "sub-1",
		ClientID: client1,
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
			Kind:     &kind,
		},
	}

	sub2 := &Subscription{
		ID:       "sub-2",
		ClientID: client1,
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
		},
	}

	sub3 := &Subscription{
		ID:       "sub-3",
		ClientID: client2,
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
		},
	}

	index.Add(sub1)
	index.Add(sub2)
	index.Add(sub3)

	// Obtener suscripciones de client1
	subs := index.GetByClient(client1)
	if len(subs) != 2 {
		t.Errorf("Expected 2 subscriptions for client1, got %d", len(subs))
	}

	// Obtener suscripciones de client2
	subs = index.GetByClient(client2)
	if len(subs) != 1 {
		t.Errorf("Expected 1 subscription for client2, got %d", len(subs))
	}
}

func TestSubscriptionIndex_FindMatching_Specificity(t *testing.T) {
	index := NewSubscriptionIndex()
	tenantID := primitive.NewObjectID()
	kind := domain.StreamKindClimate
	farmID := "farm-001"
	siteID := "site-001"
	cageID := "cage-001"

	// Suscripción muy específica (cage level)
	subCage := &Subscription{
		ID:       "sub-cage",
		ClientID: "client-cage",
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
			Kind:     &kind,
			CageID:   &cageID,
		},
	}

	// Suscripción a nivel site
	subSite := &Subscription{
		ID:       "sub-site",
		ClientID: "client-site",
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
			Kind:     &kind,
			SiteID:   &siteID,
		},
	}

	// Suscripción a nivel farm
	subFarm := &Subscription{
		ID:       "sub-farm",
		ClientID: "client-farm",
		Filter: SubscriptionFilter{
			TenantID: &tenantID,
			Kind:     &kind,
			FarmID:   &farmID,
		},
	}

	index.Add(subCage)
	index.Add(subSite)
	index.Add(subFarm)

	// Evento a nivel cage
	event := createTestEvent(tenantID, kind, farmID, siteID, &cageID)
	matched := index.FindMatching(event)

	// Todas las suscripciones deben coincidir
	if len(matched) != 3 {
		t.Errorf("Expected 3 matching subscriptions, got %d", len(matched))
	}
}

// Funciones helper para crear eventos de prueba

func createTestEvent(tenantID primitive.ObjectID, kind domain.StreamKind, farmID, siteID string, cageID *string) *connectors.CanonicalEvent {
	return createTestEventWithSource(tenantID, kind, farmID, siteID, cageID, "test-source")
}

func createTestEventWithSource(tenantID primitive.ObjectID, kind domain.StreamKind, farmID, siteID string, cageID *string, source string) *connectors.CanonicalEvent {
	streamKey := domain.StreamKey{
		TenantID: tenantID,
		Kind:     kind,
		FarmID:   farmID,
		SiteID:   siteID,
		CageID:   cageID,
	}

	payload := map[string]interface{}{
		"test": "data",
	}
	payloadBytes, _ := json.Marshal(payload)

	return &connectors.CanonicalEvent{
		Envelope: connectors.Envelope{
			Version:   "1.0",
			Timestamp: time.Now(),
			Stream:    streamKey,
			Source:    source,
			Sequence:  1,
		},
		Payload:       payloadBytes,
		Kind:          string(kind),
		SchemaVersion: "v1",
	}
}
