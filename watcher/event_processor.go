package main

import (
	"context"
	"errors"
	"sync"
	"time"
	externalservices "watcher/external-services"
	"watcher/logger"
)

type EventProcessor struct {
	mu                     sync.Mutex
	activeBuffer           []externalservices.MongoChangeEvent
	remainingBuffer        []externalservices.MongoChangeEvent
	lastFlush              time.Time
	typesenseService       externalservices.TypesenseServiceContract
	primaryRedis           externalservices.RedisServiceContract
	backupRedis            externalservices.RedisServiceContract
	watchingMongoDBService externalservices.MongoServiceContract
	fallbackMongoService   externalservices.MongoServiceContract
	ticker                 *time.Ticker
	backupFlushTicker      *time.Ticker
	done                   chan struct{}
	eventChan              chan externalservices.MongoChangeEvent
	backupFlusher          BackupFlusherContract
	maxBufferSize          int
	processInterval        time.Duration
	backupFlushInterval    time.Duration
	stopChan               chan struct{}
	processBusy            chan bool
	isNearLimit            chan bool         // Channel to signal when buffer is near limit
	metrics                *MetricsCollector // Add metrics collector
	closed                 bool              // Add closed flag
}

func NewEventProcessor(
	typesenseService externalservices.TypesenseServiceContract,
	primaryRedis externalservices.RedisServiceContract,
	backupRedis externalservices.RedisServiceContract,
	watchingMongoDBService externalservices.MongoServiceContract,
	fallbackMongoService externalservices.MongoServiceContract,
	maxBufferSize int,
	processInterval time.Duration,
	backupFlushInterval time.Duration,
) (*EventProcessor, error) {

	backupFlusher := NewBackupFlusher(
		primaryRedis,
		backupRedis,
		watchingMongoDBService,
	)

	processor := &EventProcessor{
		activeBuffer:           make([]externalservices.MongoChangeEvent, 0),
		remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
		lastFlush:              time.Now(),
		typesenseService:       typesenseService,
		primaryRedis:           primaryRedis,
		backupRedis:            backupRedis,
		watchingMongoDBService: watchingMongoDBService,
		fallbackMongoService:   fallbackMongoService,
		done:                   make(chan struct{}),
		eventChan:              make(chan externalservices.MongoChangeEvent, maxBufferSize),
		backupFlusher:          backupFlusher,
		maxBufferSize:          maxBufferSize,
		processInterval:        processInterval,
		backupFlushInterval:    backupFlushInterval,
		stopChan:               make(chan struct{}),
		processBusy:            make(chan bool, 1),
		isNearLimit:            make(chan bool, 1),    // Buffered channel to prevent blocking
		metrics:                NewMetricsCollector(), // Initialize metrics collector
	}

	processor.ticker = time.NewTicker(processInterval)
	processor.backupFlushTicker = time.NewTicker(backupFlushInterval)
	go processor.accumulateEvents()
	go processor.startProcessingTimer()
	go processor.startBackupFlushTimer()

	return processor, nil
}

func (w *EventProcessor) accumulateEvents() {
	for {
		select {
		case <-w.done:
			close(w.eventChan)
			return
		case event := <-w.eventChan:
			w.mu.Lock()
			w.activeBuffer = append(w.activeBuffer, event)

			// Update queue metrics
			w.metrics.UpdateQueueLength(len(w.eventChan))
			w.metrics.UpdateBufferSizes(len(w.activeBuffer), len(w.remainingBuffer))

			// Signal when we're 2 items away from maxBufferSize
			if len(w.activeBuffer) >= (w.maxBufferSize - 2) {
				select {
				case w.isNearLimit <- true:
					logger.Warn().
						Int("buffer_size", len(w.activeBuffer)).
						Int("max_buffer_size", w.maxBufferSize).
						Msg("Buffer approaching limit")
				default:
					// Channel already has a signal
				}
			}
			w.mu.Unlock()
		}
	}
}

func (w *EventProcessor) setRemainingBuffer(batch []externalservices.MongoChangeEvent, i int) {
	w.remainingBuffer = make([]externalservices.MongoChangeEvent, len(batch[i:]))
	w.remainingBuffer = batch[i:]
}

func (w *EventProcessor) processBatch() {
	// Signal that processing has started
	w.processBusy <- true
	startTime := time.Now()
	defer func() {
		w.mu.Lock()
		// After processing, add any remaining buffer to the start of the active buffer
		w.activeBuffer = append(w.remainingBuffer, w.activeBuffer...)
		// Clear the remaining buffer
		w.remainingBuffer = make([]externalservices.MongoChangeEvent, 0)
		w.mu.Unlock()

		// Update buffer size metrics
		w.metrics.UpdateBufferSizes(len(w.activeBuffer), len(w.remainingBuffer))

		// Signal when we're 2 items away from maxBufferSize
		if len(w.activeBuffer) >= (w.maxBufferSize - 2) {
			select {
			case w.isNearLimit <- true:
				logger.Warn().
					Int("buffer_size", len(w.activeBuffer)).
					Int("max_buffer_size", w.maxBufferSize).
					Msg("Buffer approaching limit")
			default:
				// Channel already has a signal
			}
		}

		// Record batch processing duration
		w.metrics.RecordBatchProcessingDuration(time.Since(startTime).Seconds())

		// Signal that processing has finished
		w.processBusy <- false
	}()

	w.mu.Lock()
	// Create a copy of the current buffer for processing
	batch := make([]externalservices.MongoChangeEvent, len(w.activeBuffer))
	copy(batch, w.activeBuffer)
	// Clear the active buffer
	w.activeBuffer = make([]externalservices.MongoChangeEvent, 0)
	w.mu.Unlock()

	// Record batch size
	w.metrics.RecordBatchSize(len(batch))

	// Process events in order
	for i := 0; i < len(batch); i++ {
		event := batch[i]
		ctx := context.Background()
		eventStartTime := time.Now()

		// Log event processing
		logger.Debug().
			Str("operation_type", event.OperationType).
			Str("document_id", event.DocumentID).
			Int64("timestamp", event.Timestamp).
			Int("batch_position", i+1).
			Int("batch_size", len(batch)).
			Msg("Processing event")

		// For insert/update operations with nil documents, keep them in the active buffer
		if (event.OperationType == "insert" || event.OperationType == "update") && event.Document == nil {
			logger.Warn().
				Str("operation_type", event.OperationType).
				Str("document_id", event.DocumentID).
				Int("batch_position", i+1).
				Int("batch_size", len(batch)).
				Msg("Keeping event with nil document in active buffer")
			w.mu.Lock()
			w.activeBuffer = append(w.activeBuffer, event)
			w.mu.Unlock()
			w.metrics.RecordEventProcessed(event.OperationType, "skipped_nil_document")
			continue
		}

		var err error
		switch event.OperationType {
		case "insert", "update":
			opStartTime := time.Now()
			err = w.typesenseService.UpsertDocument(ctx, event.Document)
			w.metrics.RecordTypesenseOperationDuration("upsert", time.Since(opStartTime).Seconds())
		case "delete":
			opStartTime := time.Now()
			err = w.typesenseService.DeleteDocument(ctx, event.DocumentID)
			w.metrics.RecordTypesenseOperationDuration("delete", time.Since(opStartTime).Seconds())
		default:
			logger.Warn().
				Str("operation_type", event.OperationType).
				Msg("Unsupported operation type")
			w.metrics.RecordEventProcessed(event.OperationType, "unsupported")
			continue
		}

		if err != nil {
			logger.Error().
				Err(err).
				Str("operation_type", event.OperationType).
				Str("document_id", event.DocumentID).
				Int("batch_position", i+1).
				Int("batch_size", len(batch)).
				Msg("Failed to process event")

			w.metrics.RecordEventProcessed(event.OperationType, "error")
			w.metrics.RecordError("typesense_" + event.OperationType)

			// Store remaining events in backup
			w.setRemainingBuffer(batch, i)
			w.backupFlusher.Flush(w.remainingBuffer)
			return
		}

		// Record successful event processing
		w.metrics.RecordEventProcessed(event.OperationType, "success")
		w.metrics.RecordEventProcessingDuration(event.OperationType, time.Since(eventStartTime).Seconds())

		// Log successful event processing
		logger.Debug().
			Str("operation_type", event.OperationType).
			Str("document_id", event.DocumentID).
			Int("batch_position", i+1).
			Int("batch_size", len(batch)).
			Msg("Successfully processed event")
	}

	// Update last flush time
	w.lastFlush = time.Now()
}

func (w *EventProcessor) Enqueue(ctx context.Context, event externalservices.MongoChangeEvent) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return ErrProcessorClosed
	}
	w.mu.Unlock()
	select {
	case w.eventChan <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// FlushAndClose flushes any remaining events to backup and then closes the processor.
func (w *EventProcessor) FlushAndClose() {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	// First close the done channel to stop event accumulation
	close(w.done)

	// Stop the tickers to prevent new processing
	if w.ticker != nil {
		w.ticker.Stop()
	}
	if w.backupFlushTicker != nil {
		w.backupFlushTicker.Stop()
	}

	// Signal that processing has started
	select {
	case w.processBusy <- true:
	default:
		// If processBusy is full, we'll skip the signal
	}

	// Get the mutex lock to safely access and clear the buffer
	w.mu.Lock()
	events := make([]externalservices.MongoChangeEvent, len(w.activeBuffer))
	copy(events, w.activeBuffer)
	w.activeBuffer = make([]externalservices.MongoChangeEvent, 0)
	w.mu.Unlock()

	// Flush events if any
	if len(events) > 0 {
		w.backupFlusher.Flush(events)
	}

	// Signal that processing has finished
	select {
	case w.processBusy <- false:
	default:
		// If processBusy is full, we'll skip the signal
	}

	// Close channels to unblock any listeners
	close(w.stopChan)
	close(w.isNearLimit)
}

func (w *EventProcessor) startProcessingTimer() {
	for {
		select {
		case <-w.ticker.C:
			w.mu.Lock()
			// Only process if we have events
			if len(w.activeBuffer) > 0 {
				go w.processBatch()
			}
			w.mu.Unlock()
		case <-w.stopChan:
			w.ticker.Stop()
			return
		case <-w.done:
			w.ticker.Stop()
			return
		}
	}
}

func (w *EventProcessor) startBackupFlushTimer() {
	for {
		select {
		case <-w.backupFlushTicker.C:
			w.mu.Lock()
			if len(w.activeBuffer) > 0 {

				// Create a copy of the buffer for flushing
				events := make([]externalservices.MongoChangeEvent, len(w.activeBuffer))
				copy(events, w.activeBuffer)
				w.mu.Unlock()

				// Flush to backup
				w.backupFlusher.Flush(events)
				w.lastFlush = time.Now()
			} else {
				w.mu.Unlock()
			}
		case <-w.stopChan:
			w.backupFlushTicker.Stop()
			return
		case <-w.done:
			w.backupFlushTicker.Stop()
			return
		}
	}
}

// Stop gracefully stops the event processor
func (e *EventProcessor) Stop() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.closed = true
	e.mu.Unlock()
	close(e.stopChan)
	close(e.done)
	if e.ticker != nil {
		e.ticker.Stop()
	}
	if e.backupFlushTicker != nil {
		e.backupFlushTicker.Stop()
	}
	close(e.isNearLimit)
}

// GetProcessBusyState returns a channel that signals whether the processor is currently busy processing
func (w *EventProcessor) GetProcessBusyState() <-chan bool {
	return w.processBusy
}

// GetNearLimitState returns a channel that signals whether the buffer is approaching its limit
func (w *EventProcessor) GetNearLimitState() <-chan bool {
	return w.isNearLimit
}

// Add error for closed processor
var ErrProcessorClosed = errors.New("event processor is closed")
