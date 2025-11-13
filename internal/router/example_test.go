package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"
	"omniapi/internal/router"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Example demonstrates basic usage of the router
func Example_basicUsage() {
	// Create router
	r := router.NewRouter()

	// Start processing
	ctx := context.Background()
	r.Start(ctx)
	defer r.Stop()

	// Setup callback to send events to WebSocket
	r.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
		fmt.Printf("Sending event to client %s: %s\n", clientID, event.Kind)
		return nil
	})

	// Register a client
	clientID := "ws-client-1"
	tenantID := primitive.NewObjectID()

	permissions := []domain.Capability{
		domain.CapabilityFeedingRead,
	}

	scopes := []domain.Scope{
		{
			TenantID:    tenantID,
			Resource:    "farm:farm-001",
			Permissions: permissions,
			FarmIDs:     []string{"farm-001"},
			SiteIDs:     []string{"site-001"},
		},
	}

	throttleConfig := &router.ThrottleConfig{
		ThrottleMs:        100,
		MaxRate:           10.0,
		BurstSize:         5,
		CoalescingEnabled: true,
		KeepLatest:        true,
		BufferSize:        100,
	}

	r.RegisterClient(clientID, tenantID, permissions, scopes, throttleConfig)

	// Subscribe to feeding events
	kind := domain.StreamKindFeeding
	siteID := "site-001"

	filter := router.SubscriptionFilter{
		TenantID: &tenantID,
		Kind:     &kind,
		SiteID:   &siteID,
	}

	r.Subscribe(clientID, filter)

	// Route an event
	event := &connectors.CanonicalEvent{
		Envelope: connectors.Envelope{
			Version:   "1.0",
			Timestamp: time.Now(),
			Stream: domain.StreamKey{
				TenantID: tenantID,
				Kind:     domain.StreamKindFeeding,
				FarmID:   "farm-001",
				SiteID:   "site-001",
			},
			Source:   "mqtt-connector",
			Sequence: 1,
		},
		Kind:          "feeding",
		SchemaVersion: "v1",
		Payload:       json.RawMessage(`{"amount": 100}`),
	}

	r.RouteEvent(event)

	// Give some time for processing
	time.Sleep(200 * time.Millisecond)

	// Get stats
	stats := r.GetStats()
	fmt.Printf("Events routed: %d\n", stats.EventsRouted)
	fmt.Printf("Active clients: %d\n", stats.ActiveClients)
}

// Example_multiConnectorPolicy demonstrates multi-connector policy usage
func Example_multiConnectorPolicy() {
	r := router.NewRouter()
	tenantID := primitive.NewObjectID()

	// Configure multi-connector policy
	multiConfig := &router.MultiConnectorConfig{
		TenantID: tenantID,
		Kind:     domain.StreamKindFeeding,
		Policy:   router.PolicyFallback,
		Connectors: []router.ConnectorConfig{
			{
				ID:         "mqtt-primary",
				Type:       "mqtt",
				Priority:   100,
				Enabled:    true,
				Timeout:    5000,
				MaxRetries: 3,
			},
			{
				ID:         "rest-backup",
				Type:       "rest",
				Priority:   50,
				Enabled:    true,
				Timeout:    10000,
				MaxRetries: 2,
			},
		},
	}

	r.SetMultiConnectorPolicy(multiConfig)

	// Select connector
	connector, err := r.SelectConnector(tenantID.Hex(), domain.StreamKindFeeding)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Selected connector: %s (priority: %d)\n", connector.ID, connector.Priority)
	// Output: Selected connector: mqtt-primary (priority: 100)
}

// Example_throttling demonstrates throttling and coalescing
func Example_throttling() {
	r := router.NewRouter()
	ctx := context.Background()
	r.Start(ctx)
	defer r.Stop()

	clientID := "ws-client-throttle"
	tenantID := primitive.NewObjectID()

	// Configure aggressive throttling
	throttleConfig := &router.ThrottleConfig{
		ThrottleMs:        500, // Minimum 500ms between events
		MaxRate:           2.0, // Max 2 events per second
		BurstSize:         2,
		CoalescingEnabled: true,
		KeepLatest:        true,
		BufferSize:        50,
	}

	permissions := []domain.Capability{domain.CapabilityFeedingRead}
	scopes := []domain.Scope{
		{
			TenantID:    tenantID,
			Resource:    "*",
			Permissions: permissions,
		},
	}

	r.RegisterClient(clientID, tenantID, permissions, scopes, throttleConfig)

	// Subscribe
	kind := domain.StreamKindFeeding
	filter := router.SubscriptionFilter{
		TenantID: &tenantID,
		Kind:     &kind,
	}

	r.Subscribe(clientID, filter)

	// Setup callback to count events
	eventCount := 0
	r.SetEventCallback(func(cID string, event *connectors.CanonicalEvent) error {
		eventCount++
		return nil
	})

	// Send 10 events rapidly
	for i := 0; i < 10; i++ {
		event := &connectors.CanonicalEvent{
			Envelope: connectors.Envelope{
				Version:   "1.0",
				Timestamp: time.Now(),
				Stream: domain.StreamKey{
					TenantID: tenantID,
					Kind:     domain.StreamKindFeeding,
					FarmID:   "farm-001",
					SiteID:   "site-001",
				},
				Source:   "test",
				Sequence: uint64(i),
			},
			Kind:          "feeding",
			SchemaVersion: "v1",
			Payload:       json.RawMessage(fmt.Sprintf(`{"seq": %d}`, i)),
		}
		r.RouteEvent(event)
	}

	// Wait for processing
	time.Sleep(2 * time.Second)

	fmt.Printf("Sent 10 events, delivered: %d (throttled)\n", eventCount)
}

// Example_hierarchicalFiltering demonstrates filtering at different levels
func Example_hierarchicalFiltering() {
	r := router.NewRouter()
	tenantID := primitive.NewObjectID()

	// Client 1: Farm-level access
	client1 := "farm-level-client"
	permissions := []domain.Capability{domain.CapabilityFeedingRead}
	scopes := []domain.Scope{
		{
			TenantID:    tenantID,
			Resource:    "farm:farm-001",
			Permissions: permissions,
			FarmIDs:     []string{"farm-001"},
		},
	}
	r.RegisterClient(client1, tenantID, permissions, scopes, nil)

	farmID := "farm-001"
	kind := domain.StreamKindFeeding
	r.Subscribe(client1, router.SubscriptionFilter{
		TenantID: &tenantID,
		Kind:     &kind,
		FarmID:   &farmID,
	})

	// Client 2: Site-level access
	client2 := "site-level-client"
	scopes2 := []domain.Scope{
		{
			TenantID:    tenantID,
			Resource:    "site:site-001",
			Permissions: permissions,
			SiteIDs:     []string{"site-001"},
		},
	}
	r.RegisterClient(client2, tenantID, permissions, scopes2, nil)

	siteID := "site-001"
	r.Subscribe(client2, router.SubscriptionFilter{
		TenantID: &tenantID,
		Kind:     &kind,
		SiteID:   &siteID,
	})

	// Client 3: Cage-level access
	client3 := "cage-level-client"
	scopes3 := []domain.Scope{
		{
			TenantID:    tenantID,
			Resource:    "cage:cage-001",
			Permissions: permissions,
			CageIDs:     []string{"cage-001"},
		},
	}
	r.RegisterClient(client3, tenantID, permissions, scopes3, nil)

	cageID := "cage-001"
	r.Subscribe(client3, router.SubscriptionFilter{
		TenantID: &tenantID,
		Kind:     &kind,
		CageID:   &cageID,
	})

	stats := r.GetStats()
	fmt.Printf("Registered %d clients with hierarchical filtering\n", stats.ActiveClients)
	// Output: Registered 3 clients with hierarchical filtering
}
