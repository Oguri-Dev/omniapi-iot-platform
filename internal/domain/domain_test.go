package domain

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCapability_IsValid(t *testing.T) {
	tests := []struct {
		capability Capability
		valid      bool
	}{
		{CapabilityFeedingRead, true},
		{CapabilityBiometricRead, true},
		{CapabilityClimateRead, true},
		{CapabilityOpsRead, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.capability), func(t *testing.T) {
			if got := tt.capability.IsValid(); got != tt.valid {
				t.Errorf("Capability.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestTenant_Validate(t *testing.T) {
	validTenant := NewTenant("test-tenant", "Test Tenant", "admin")

	// Test valid tenant
	if err := validTenant.Validate(); err != nil {
		t.Errorf("Valid tenant should pass validation, got error: %v", err)
	}

	// Test invalid tenant (empty name)
	invalidTenant := *validTenant
	invalidTenant.Name = ""
	if err := invalidTenant.Validate(); err == nil {
		t.Error("Tenant with empty name should fail validation")
	}

	// Test invalid tenant (empty display name)
	invalidTenant2 := *validTenant
	invalidTenant2.DisplayName = ""
	if err := invalidTenant2.Validate(); err == nil {
		t.Error("Tenant with empty display name should fail validation")
	}
}

func TestStreamKey_Validate(t *testing.T) {
	tenantID := primitive.NewObjectID()

	// Test valid StreamKey
	validKey := NewStreamKey(tenantID, StreamKindFeeding, "farm123", "site456", nil)
	if err := validKey.Validate(); err != nil {
		t.Errorf("Valid StreamKey should pass validation, got error: %v", err)
	}

	// Test invalid StreamKey (zero tenant ID)
	invalidKey := *validKey
	invalidKey.TenantID = primitive.NilObjectID
	if err := invalidKey.Validate(); err == nil {
		t.Error("StreamKey with zero tenant ID should fail validation")
	}

	// Test invalid StreamKey (empty farm ID)
	invalidKey2 := *validKey
	invalidKey2.FarmID = ""
	if err := invalidKey2.Validate(); err == nil {
		t.Error("StreamKey with empty farm ID should fail validation")
	}
}

func TestStreamKey_StringAndParse(t *testing.T) {
	tenantID := primitive.NewObjectID()
	cageID := "cage789"

	// Test StreamKey with cage
	key := NewStreamKey(tenantID, StreamKindBiometric, "farm123", "site456", &cageID)
	keyStr := key.String()

	parsed, err := ParseStreamKey(keyStr)
	if err != nil {
		t.Errorf("Failed to parse StreamKey string: %v", err)
	}

	if !key.Equals(parsed) {
		t.Error("Parsed StreamKey should equal original")
	}

	// Test StreamKey without cage
	key2 := NewStreamKey(tenantID, StreamKindClimate, "farm123", "site456", nil)
	keyStr2 := key2.String()

	parsed2, err := ParseStreamKey(keyStr2)
	if err != nil {
		t.Errorf("Failed to parse StreamKey string without cage: %v", err)
	}

	if !key2.Equals(parsed2) {
		t.Error("Parsed StreamKey without cage should equal original")
	}
}

func TestConnectionInstance_Validate(t *testing.T) {
	tenantID := primitive.NewObjectID()
	typeID := primitive.NewObjectID()

	// Test valid connection
	conn := NewConnectionInstance(tenantID, typeID, "Test Connection", "admin")
	if err := conn.Validate(); err != nil {
		t.Errorf("Valid ConnectionInstance should pass validation, got error: %v", err)
	}

	// Test invalid connection (zero tenant ID)
	invalidConn := *conn
	invalidConn.TenantID = primitive.NilObjectID
	if err := invalidConn.Validate(); err == nil {
		t.Error("ConnectionInstance with zero tenant ID should fail validation")
	}

	// Test invalid connection (empty display name)
	invalidConn2 := *conn
	invalidConn2.DisplayName = ""
	if err := invalidConn2.Validate(); err == nil {
		t.Error("ConnectionInstance with empty display name should fail validation")
	}
}

func TestMapping_Validate(t *testing.T) {
	// Test valid mapping
	mapping := NewMapping("Temperature Mapping", CapabilityClimateRead)
	if err := mapping.Validate(); err != nil {
		t.Errorf("Valid Mapping should pass validation, got error: %v", err)
	}

	// Test invalid mapping (empty name)
	invalidMapping := *mapping
	invalidMapping.Name = ""
	if err := invalidMapping.Validate(); err == nil {
		t.Error("Mapping with empty name should fail validation")
	}

	// Test invalid mapping (invalid capability)
	invalidMapping2 := *mapping
	invalidMapping2.Capability = "invalid"
	if err := invalidMapping2.Validate(); err == nil {
		t.Error("Mapping with invalid capability should fail validation")
	}
}

func TestScope_HasCapability(t *testing.T) {
	scope := &Scope{
		TenantID:    primitive.NewObjectID(),
		Resource:    "*",
		Permissions: []Capability{CapabilityFeedingRead, CapabilityClimateRead},
	}

	if !scope.HasCapability(CapabilityFeedingRead) {
		t.Error("Scope should have CapabilityFeedingRead")
	}

	if !scope.HasCapability(CapabilityClimateRead) {
		t.Error("Scope should have CapabilityClimateRead")
	}

	if scope.HasCapability(CapabilityBiometricRead) {
		t.Error("Scope should not have CapabilityBiometricRead")
	}
}

func TestCanAccess(t *testing.T) {
	tenantID := primitive.NewObjectID()

	scope := Scope{
		TenantID:    tenantID,
		Resource:    "*",
		Permissions: []Capability{CapabilityFeedingRead},
	}

	streamKey := StreamKey{
		TenantID: tenantID,
		Kind:     StreamKindFeeding,
		FarmID:   "farm123",
		SiteID:   "site456",
	}

	// Test valid access
	if !CanAccess(scope, CapabilityFeedingRead, streamKey) {
		t.Error("Should have access to feeding capability")
	}

	// Test invalid capability
	if CanAccess(scope, CapabilityBiometricRead, streamKey) {
		t.Error("Should not have access to biometric capability")
	}

	// Test different tenant
	differentTenantKey := streamKey
	differentTenantKey.TenantID = primitive.NewObjectID()
	if CanAccess(scope, CapabilityFeedingRead, differentTenantKey) {
		t.Error("Should not have access to different tenant's stream")
	}
}
