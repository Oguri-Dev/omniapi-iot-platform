package polling

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"omniapi/internal/broker"
	"omniapi/internal/models"
	"omniapi/internal/services"
)

// Worker ejecuta polling para una instancia de endpoint
type Worker struct {
	config   *PollingConfig
	instance EndpointInstance
	service  *models.ExternalService

	status   WorkerStatus
	statusMu sync.RWMutex

	ctx     context.Context
	cancel  context.CancelFunc
	running atomic.Bool

	// Callback para resultados
	onResult func(PollingResult)

	// Broker manager para publicar resultados
	brokerManager *broker.Manager

	// HTTP client reutilizable
	httpClient *http.Client
}

// NewWorker crea un nuevo worker de polling
func NewWorker(config *PollingConfig, instance EndpointInstance, service *models.ExternalService) *Worker {
	return &Worker{
		config:   config,
		instance: instance,
		service:  service,
		status: WorkerStatus{
			InstanceID: instance.InstanceID,
			EndpointID: instance.EndpointID,
			Label:      instance.Label,
			Status:     "stopped",
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OnResult registra callback para recibir resultados
func (w *Worker) OnResult(callback func(PollingResult)) {
	w.onResult = callback
}

// SetBrokerManager establece el broker manager para publicar resultados
func (w *Worker) SetBrokerManager(manager *broker.Manager) {
	w.brokerManager = manager
}

// Start inicia el worker
func (w *Worker) Start(ctx context.Context) error {
	if w.running.Load() {
		return fmt.Errorf("worker already running")
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.running.Store(true)

	w.statusMu.Lock()
	w.status.Status = "running"
	w.statusMu.Unlock()

	go w.pollLoop()

	return nil
}

// Stop detiene el worker
func (w *Worker) Stop() {
	if !w.running.Load() {
		return
	}

	w.cancel()
	w.running.Store(false)

	w.statusMu.Lock()
	w.status.Status = "stopped"
	w.statusMu.Unlock()
}

// GetStatus retorna el estado actual del worker
func (w *Worker) GetStatus() WorkerStatus {
	w.statusMu.RLock()
	defer w.statusMu.RUnlock()
	return w.status
}

// pollLoop ciclo principal de polling
// IMPORTANTE: No se solapan requests - espera a que termine antes de programar el siguiente
func (w *Worker) pollLoop() {
	interval := w.getEffectiveInterval()

	// Primera ejecución inmediata
	w.executePoll()

	for {
		// Crear timer DESPUÉS de que termine la ejecución anterior
		// Esto garantiza que siempre hay al menos 'interval' entre el FIN de una
		// request y el INICIO de la siguiente
		timer := time.NewTimer(interval)

		select {
		case <-w.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			w.executePoll()
		}
	}
}

// getEffectiveInterval calcula el intervalo efectivo para este endpoint
// Prioridad: endpoint > config global > default (2s)
// Mínimo permitido: 1 segundo
func (w *Worker) getEffectiveInterval() time.Duration {
	const minInterval = 1 * time.Second
	const defaultInterval = 2 * time.Second

	var interval time.Duration

	// 1. Usar intervalo del endpoint si está definido (> 0)
	if w.instance.IntervalMS > 0 {
		interval = time.Duration(w.instance.IntervalMS) * time.Millisecond
	} else if w.config.IntervalMS > 0 {
		// 2. Usar intervalo global del config
		interval = time.Duration(w.config.IntervalMS) * time.Millisecond
	} else {
		// 3. Default
		interval = defaultInterval
	}

	// Aplicar mínimo de 1 segundo
	if interval < minInterval {
		interval = minInterval
	}

	return interval
}

// executePoll ejecuta una consulta al endpoint
func (w *Worker) executePoll() {
	startTime := time.Now()

	result := PollingResult{
		InstanceID: w.instance.InstanceID,
		EndpointID: w.instance.EndpointID,
		Label:      w.instance.Label,
		Provider:   w.config.Provider,
		SiteID:     w.config.SiteID,
		TenantID:   w.config.TenantID,
		Path:       w.instance.Path,
		Method:     w.instance.Method,
		Params:     w.instance.Params,
		PolledAt:   startTime,
	}

	// Obtener token
	token, err := services.GetTokenManager().GetToken(w.service)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("error obteniendo token: %v", err)
		result.LatencyMS = time.Since(startTime).Milliseconds()
		w.handleResult(result)
		return
	}

	// Construir URL
	url := w.buildURL()
	result.FullURL = url

	// Crear request
	var req *http.Request
	if w.instance.Method == "POST" {
		// Para POST, los params van en el body
		bodyJSON, _ := json.Marshal(w.instance.Params)
		req, err = http.NewRequestWithContext(w.ctx, "POST", url, strings.NewReader(string(bodyJSON)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(w.ctx, "GET", url, nil)
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("error creando request: %v", err)
		result.LatencyMS = time.Since(startTime).Milliseconds()
		w.handleResult(result)
		return
	}

	// Headers según provider
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if w.config.Provider == "scaleaq" {
		req.Header.Set("Scale-Version", "2025-01-01")
		req.Header.Set("Accept", "application/json")
	}

	// Ejecutar request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("error en request: %v", err)
		result.LatencyMS = time.Since(startTime).Milliseconds()
		w.handleResult(result)
		return
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("error leyendo respuesta: %v", err)
		result.LatencyMS = time.Since(startTime).Milliseconds()
		w.handleResult(result)
		return
	}

	result.StatusCode = resp.StatusCode
	result.LatencyMS = time.Since(startTime).Milliseconds()
	result.ResponseSize = len(body)

	// Parsear JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Si no es JSON válido, guardar como string
		data = string(body)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
		result.Data = data
	} else {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		result.Data = data
	}

	w.handleResult(result)
}

// buildURL construye la URL final reemplazando placeholders con params
func (w *Worker) buildURL() string {
	baseURL := strings.TrimSuffix(w.service.BaseURL, "/")
	path := w.instance.Path

	// Reemplazar placeholders en el path: {monitor_id} -> valor real
	for key, value := range w.instance.Params {
		placeholder := fmt.Sprintf("{%s}", key)
		path = strings.ReplaceAll(path, placeholder, value)
	}

	// Si es GET y hay params que no están en el path, agregarlos como query string
	if w.instance.Method == "GET" {
		queryParams := []string{}
		for key, value := range w.instance.Params {
			// Solo agregar si no estaba como placeholder
			if !strings.Contains(w.instance.Path, fmt.Sprintf("{%s}", key)) {
				// Verificar si ya está en la query string del path
				if !strings.Contains(path, key+"=") {
					queryParams = append(queryParams, fmt.Sprintf("%s=%s", key, value))
				}
			}
		}
		if len(queryParams) > 0 {
			separator := "?"
			if strings.Contains(path, "?") {
				separator = "&"
			}
			path = path + separator + strings.Join(queryParams, "&")
		}
	}

	return baseURL + path
}

// handleResult procesa el resultado del polling
func (w *Worker) handleResult(result PollingResult) {
	// Actualizar estadísticas
	w.statusMu.Lock()
	w.status.TotalPolls++
	w.status.LastPollAt = result.PolledAt

	if result.Success {
		w.status.TotalSuccess++
		w.status.LastSuccessAt = result.PolledAt
		w.status.ConsecutiveErrs = 0
		w.status.LastError = ""
	} else {
		w.status.TotalErrors++
		w.status.LastErrorAt = result.PolledAt
		w.status.LastError = result.Error
		w.status.ConsecutiveErrs++
	}

	// Calcular latencia promedio
	if w.status.TotalPolls > 0 {
		// Media móvil simple
		w.status.AvgLatencyMS = (w.status.AvgLatencyMS*float64(w.status.TotalPolls-1) + float64(result.LatencyMS)) / float64(w.status.TotalPolls)
	}
	w.statusMu.Unlock()

	// Log a consola con formato detallado
	w.logResult(result)

	// Guardar último resultado en el engine
	GetEngine().SaveLastResult(result)

	// Publicar al broker si está configurado y habilitado
	w.publishToBroker(result)

	// Callback si está registrado
	if w.onResult != nil {
		w.onResult(result)
	}
}

// publishToBroker publica el resultado al broker MQTT si está configurado
func (w *Worker) publishToBroker(result PollingResult) {
	// Verificar si hay configuración de output
	if w.config.Output == nil || !w.config.Output.Enabled || w.brokerManager == nil {
		return
	}

	// Solo publicar resultados exitosos (con datos)
	if !result.Success || result.Data == nil {
		return
	}

	// Construir variables para el template del topic
	vars := map[string]string{
		"provider":     result.Provider,
		"site":         w.config.SiteCode,
		"site_id":      result.SiteID,
		"tenant":       w.config.TenantCode,
		"tenant_id":    result.TenantID,
		"data_type":    w.instance.TargetBlock,
		"target_block": w.instance.TargetBlock,
		"endpoint":     w.instance.EndpointID,
		"instance":     result.InstanceID,
		"instance_id":  result.InstanceID,
	}

	// Obtener el pattern del template (si es un ID, buscar el pattern correspondiente)
	topicPattern := broker.GetTopicPattern(w.config.Output.TopicTemplate)

	// Construir el topic
	topic := broker.BuildTopic(topicPattern, vars)

	// Serializar datos para publicar
	payload, err := json.Marshal(map[string]interface{}{
		"provider":      result.Provider,
		"site_id":       result.SiteID,
		"tenant_id":     result.TenantID,
		"endpoint":      result.EndpointID,
		"instance_id":   result.InstanceID,
		"label":         result.Label,
		"polled_at":     result.PolledAt,
		"latency_ms":    result.LatencyMS,
		"response_size": result.ResponseSize,
		"data":          result.Data,
	})
	if err != nil {
		fmt.Printf("⚠️  Error serializando datos para broker: %v\n", err)
		return
	}

	// Publicar de forma asíncrona para no bloquear el polling
	w.brokerManager.PublishAsync(w.config.Output.BrokerID, topic, payload)
	fmt.Printf("📤 Publicado a broker '%s' topic: %s (%d bytes)\n",
		w.config.Output.BrokerID, topic, len(payload))
}

// logResult imprime el resultado en consola con formato legible
func (w *Worker) logResult(result PollingResult) {
	statusIcon := "✅"
	if !result.Success {
		statusIcon = "❌"
	}

	fmt.Printf("\n%s ══════════════════════════════════════════════════════════════\n", statusIcon)
	fmt.Printf("│ POLLING RESULT: %s\n", result.Label)
	fmt.Printf("├─────────────────────────────────────────────────────────────────\n")
	fmt.Printf("│ Provider:    %s\n", result.Provider)
	fmt.Printf("│ Site:        %s (tenant: %s)\n", result.SiteID, result.TenantID)
	fmt.Printf("│ Endpoint:    %s %s\n", result.Method, result.Path)
	fmt.Printf("│ Full URL:    %s\n", result.FullURL)
	fmt.Printf("│ Instance:    %s\n", result.InstanceID)

	if len(result.Params) > 0 {
		fmt.Printf("│ Params:\n")
		for k, v := range result.Params {
			fmt.Printf("│   • %s: %s\n", k, v)
		}
	}

	fmt.Printf("├─────────────────────────────────────────────────────────────────\n")
	fmt.Printf("│ Status:      %d\n", result.StatusCode)
	fmt.Printf("│ Latency:     %d ms\n", result.LatencyMS)
	fmt.Printf("│ Size:        %d bytes\n", result.ResponseSize)
	fmt.Printf("│ Time:        %s\n", result.PolledAt.Format("15:04:05.000"))

	if result.Success {
		fmt.Printf("├─────────────────────────────────────────────────────────────────\n")
		fmt.Printf("│ DATA:\n")
		// Imprimir JSON formateado (primeros 1000 chars)
		dataJSON, _ := json.MarshalIndent(result.Data, "│ ", "  ")
		dataStr := string(dataJSON)
		if len(dataStr) > 1000 {
			dataStr = dataStr[:1000] + "\n│ ... (truncado)"
		}
		fmt.Printf("%s\n", dataStr)
	} else {
		fmt.Printf("├─────────────────────────────────────────────────────────────────\n")
		fmt.Printf("│ ERROR: %s\n", result.Error)
	}

	fmt.Printf("└─────────────────────────────────────────────────────────────────\n\n")
}
