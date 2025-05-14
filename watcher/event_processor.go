package main

import (
	"context"
	"log"
	"sync"
	"time"
	externalservices "watcher/external-services"
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
	isNearLimit            chan bool // Channel to signal when buffer is near limit
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
		isNearLimit:            make(chan bool, 1), // Buffered channel to prevent blocking
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

			// Signal when we're 2 items away from maxBufferSize
			if len(w.activeBuffer) >= (w.maxBufferSize - 2) {
				select {
				case w.isNearLimit <- true:
					log.Printf("Buffer approaching limit (%d/%d items)", len(w.activeBuffer), w.maxBufferSize)
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
	defer func() {
		w.mu.Lock()
		// After processing, add any remaining buffer to the start of the active buffer
		w.activeBuffer = append(w.remainingBuffer, w.activeBuffer...)
		// Clear the remaining buffer
		w.remainingBuffer = make([]externalservices.MongoChangeEvent, 0)
		w.mu.Unlock()

		// Signal when we're 2 items away from maxBufferSize
		if len(w.activeBuffer) >= (w.maxBufferSize - 2) {
			select {
			case w.isNearLimit <- true:
				log.Printf("Buffer approaching limit (%d/%d items)", len(w.activeBuffer), w.maxBufferSize)
			default:
				// Channel already has a signal
			}
		}

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

	// Process events in order
	for i, event := range batch {
		select {
		case <-w.done:
			return
		default:
			var err error
			switch event.OperationType {
			case "insert", "update":
				err = retryOperation(context.Background(), func() error {
					return w.typesenseService.UpsertDocument(context.Background(), event.Document)
				}, 3, time.Second)

				if err != nil {
					log.Printf("Failed to upsert document in Typesense after retries (ID: %s): %v",
						event.DocumentID, err)
					w.setRemainingBuffer(batch, i)
					return
				}
			case "delete":
				err = retryOperation(context.Background(), func() error {
					return w.typesenseService.DeleteDocument(context.Background(), event.DocumentID)
				}, 3, time.Second)

				if err != nil {
					log.Printf("Failed to delete document from Typesense after retries (ID: %s): %v",
						event.DocumentID, err)
					w.setRemainingBuffer(batch, i)
					return
				}
			}
		}
	}

	w.mu.Lock()
	w.lastFlush = time.Now()
	w.mu.Unlock()
}

func (w *EventProcessor) Enqueue(ctx context.Context, event externalservices.MongoChangeEvent) error {
	select {
	case w.eventChan <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// FlushAndClose flushes any remaining events to backup and then closes the processor.
func (w *EventProcessor) FlushAndClose() {
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
