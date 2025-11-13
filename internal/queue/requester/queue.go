package requester

import (
	"container/heap"
	"sync"
	"time"
)

// QueueItem representa un item en la cola con prioridad
type QueueItem struct {
	Request   Request
	Priority  int // Mayor valor = mayor prioridad
	Index     int // Índice en el heap
	EnqueueAt time.Time
}

// PriorityQueue implementa heap.Interface para ordenar por prioridad
type PriorityQueue []*QueueItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Mayor prioridad primero
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	// Si misma prioridad, FIFO (más antiguo primero)
	return pq[i].EnqueueAt.Before(pq[j].EnqueueAt)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*QueueItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

// RequestQueue gestiona la cola de solicitudes con prioridad
type RequestQueue struct {
	queue    PriorityQueue
	pending  map[string]*QueueItem // Para coalescing por key
	mu       sync.RWMutex
	notEmpty chan struct{}
	maxSize  int
}

// NewRequestQueue crea una nueva cola
func NewRequestQueue(maxSize int) *RequestQueue {
	rq := &RequestQueue{
		queue:    make(PriorityQueue, 0),
		pending:  make(map[string]*QueueItem),
		notEmpty: make(chan struct{}, 1),
		maxSize:  maxSize,
	}
	heap.Init(&rq.queue)
	return rq
}

// Enqueue agrega o actualiza una solicitud en la cola
func (rq *RequestQueue) Enqueue(req Request, coalescing bool) error {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	// Verificar límite de tamaño
	if len(rq.queue) >= rq.maxSize {
		return ErrQueueFull
	}

	key := req.Key()

	if coalescing {
		// Si ya existe una solicitud pendiente con la misma key, actualizar
		if existing, exists := rq.pending[key]; exists {
			// Actualizar la solicitud existente (coalescing)
			existing.Request = req
			existing.EnqueueAt = time.Now()
			// Reordenar el heap si cambió la prioridad
			heap.Fix(&rq.queue, existing.Index)
			return nil
		}
	}

	// Crear nuevo item
	priority := rq.getPriorityValue(req.Priority)
	item := &QueueItem{
		Request:   req,
		Priority:  priority,
		EnqueueAt: time.Now(),
	}

	// Agregar a la cola
	heap.Push(&rq.queue, item)
	rq.pending[key] = item

	// Notificar que hay elementos
	select {
	case rq.notEmpty <- struct{}{}:
	default:
	}

	return nil
}

// Dequeue obtiene la siguiente solicitud de mayor prioridad
func (rq *RequestQueue) Dequeue() (*Request, bool) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if len(rq.queue) == 0 {
		return nil, false
	}

	item := heap.Pop(&rq.queue).(*QueueItem)
	delete(rq.pending, item.Request.Key())

	return &item.Request, true
}

// Peek retorna la siguiente solicitud sin removerla
func (rq *RequestQueue) Peek() (*Request, bool) {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	if len(rq.queue) == 0 {
		return nil, false
	}

	return &rq.queue[0].Request, true
}

// Len retorna el número de solicitudes en cola
func (rq *RequestQueue) Len() int {
	rq.mu.RLock()
	defer rq.mu.RUnlock()
	return len(rq.queue)
}

// WaitNotEmpty espera hasta que haya elementos en la cola o se cierre el contexto
func (rq *RequestQueue) WaitNotEmpty() <-chan struct{} {
	return rq.notEmpty
}

// Clear limpia la cola
func (rq *RequestQueue) Clear() {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.queue = make(PriorityQueue, 0)
	rq.pending = make(map[string]*QueueItem)
	heap.Init(&rq.queue)
}

// GetStats retorna estadísticas de la cola
func (rq *RequestQueue) GetStats() QueueStats {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	stats := QueueStats{
		Size:       len(rq.queue),
		ByPriority: make(map[Priority]int),
	}

	var oldest, newest *time.Time

	for _, item := range rq.queue {
		stats.ByPriority[item.Request.Priority]++

		if oldest == nil || item.EnqueueAt.Before(*oldest) {
			t := item.EnqueueAt
			oldest = &t
		}
		if newest == nil || item.EnqueueAt.After(*newest) {
			t := item.EnqueueAt
			newest = &t
		}
	}

	stats.OldestEnqueued = oldest
	stats.NewestEnqueued = newest

	return stats
}

// getPriorityValue convierte Priority a valor numérico
func (rq *RequestQueue) getPriorityValue(p Priority) int {
	switch p {
	case PriorityHigh:
		return 100
	case PriorityNormal:
		return 50
	case PriorityLow:
		return 10
	default:
		return 50
	}
}

// IsPending verifica si una solicitud está pendiente
func (rq *RequestQueue) IsPending(key string) bool {
	rq.mu.RLock()
	defer rq.mu.RUnlock()
	_, exists := rq.pending[key]
	return exists
}

// Remove remueve una solicitud específica por key
func (rq *RequestQueue) Remove(key string) bool {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	item, exists := rq.pending[key]
	if !exists {
		return false
	}

	heap.Remove(&rq.queue, item.Index)
	delete(rq.pending, key)
	return true
}
