package metrics

import (
	"testing"
)

func TestSanitizeTenantID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "normal tenant ID",
			input:    "655f1c2e8c4b2a1234567890",
			expected: "655f1c2e8c4b2a1234567890",
		},
		{
			name:     "too long tenant ID",
			input:    "655f1c2e8c4b2a1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "655f1c2e8c4b2a1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeTenantID(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeTenantID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeMetric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "feeding metric",
			input:    "feeding.appetite",
			expected: "feeding",
		},
		{
			name:     "biometric metric",
			input:    "biometric.weight",
			expected: "biometric",
		},
		{
			name:     "climate metric",
			input:    "climate.temperature",
			expected: "climate",
		},
		{
			name:     "unknown metric",
			input:    "custom.something",
			expected: "other",
		},
		{
			name:     "status metric",
			input:    "status.health",
			expected: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeMetric(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeMetric(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeSiteID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "normal site ID",
			input:    "greenhouse-1",
			expected: "greenhouse-1",
		},
		{
			name:     "too long site ID",
			input:    "greenhouse-complex-section-A-subsection-1-unit-alpha-zone-beta",
			expected: "greenhouse-complex-section-A-sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSiteID(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeSiteID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty code",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "timeout",
			input:    "timeout",
			expected: "timeout",
		},
		{
			name:     "connection refused",
			input:    "connection_refused",
			expected: "connection_refused",
		},
		{
			name:     "400 client error",
			input:    "400",
			expected: "client_error",
		},
		{
			name:     "401 client error",
			input:    "401",
			expected: "client_error",
		},
		{
			name:     "500 server error",
			input:    "500",
			expected: "server_error",
		},
		{
			name:     "503 server error",
			input:    "503",
			expected: "server_error",
		},
		{
			name:     "unknown code",
			input:    "custom_error",
			expected: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorCode(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeErrorCode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMetricsRegistered(t *testing.T) {
	// Verificar que las métricas principales estén registradas
	// Esto previene errores de inicialización

	if RequesterInFlight == nil {
		t.Error("RequesterInFlight is nil")
	}
	if RequesterSuccessTotal == nil {
		t.Error("RequesterSuccessTotal is nil")
	}
	if StatusEmittedTotal == nil {
		t.Error("StatusEmittedTotal is nil")
	}
	if EventsDataOutTotal == nil {
		t.Error("EventsDataOutTotal is nil")
	}
	if WSConnectionsActive == nil {
		t.Error("WSConnectionsActive is nil")
	}
	if WSDeliveryLatencyMS == nil {
		t.Error("WSDeliveryLatencyMS is nil")
	}
}
