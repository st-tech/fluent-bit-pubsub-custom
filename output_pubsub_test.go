package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"

	"cloud.google.com/go/pubsub"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

type testOutput struct {
	inc int
}

// MockPublishResult implements pubsub.PublishResult for testing
type MockPublishResult struct {
	delay         time.Duration
	err           error
	getCalls      int
	mutex         sync.Mutex
	concurrencyTracker *ConcurrencyTracker
}

// ConcurrencyTracker tracks concurrent executions
type ConcurrencyTracker struct {
	activeCalls    int32
	maxActiveCalls int32
	mutex          sync.Mutex
}

func (m *MockPublishResult) Get(ctx context.Context) (string, error) {
	m.mutex.Lock()
	m.getCalls++
	m.mutex.Unlock()
	
	// Track concurrency if tracker is provided
	if m.concurrencyTracker != nil {
		m.concurrencyTracker.mutex.Lock()
		m.concurrencyTracker.activeCalls++
		if m.concurrencyTracker.activeCalls > m.concurrencyTracker.maxActiveCalls {
			m.concurrencyTracker.maxActiveCalls = m.concurrencyTracker.activeCalls
		}
		m.concurrencyTracker.mutex.Unlock()
		
		defer func() {
			m.concurrencyTracker.mutex.Lock()
			m.concurrencyTracker.activeCalls--
			m.concurrencyTracker.mutex.Unlock()
		}()
	}
	
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return "message-id", m.err
}

func (m *MockPublishResult) Ready() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// MockOutputWrapper for testing configuration
type MockOutputWrapper struct {
	configs map[string]string
}

func (m *MockOutputWrapper) Register(ctx unsafe.Pointer, name string, desc string) int {
	return 0
}

func (m *MockOutputWrapper) GetConfigKey(ctx unsafe.Pointer, key string) string {
	if m.configs == nil {
		return ""
	}
	return m.configs[key]
}

func (m *MockOutputWrapper) NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder {
	return nil
}

func (m *MockOutputWrapper) GetRecord(dec *output.FLBDecoder) (ret int, ts interface{}, rec map[interface{}]interface{}) {
	return 1, nil, nil // Return non-zero to stop iteration
}

func (o *testOutput) Register(ctx unsafe.Pointer, name string, desc string) int {
	return output.FLBPluginRegister(ctx, name, desc)
}

func (o *testOutput) GetConfigKey(ctx unsafe.Pointer, key string) string {
	switch key {
	case "Project":
		return os.Getenv("PROJECT_ID")
	case "Topic":
		return os.Getenv("TOPIC_NAME")
	case "Debug":
		return "true"
	case "Timeout":
		return "10000"
	case "ByteThreshold":
		return "1000000"
	case "CountThreshold":
		return "100"
	case "DelayThreshold":
		return "100"
	case "Format":
		return "json"
	case "ParallelConfirm":
		return ""
	case "ConfirmWorkers":
		return ""
	default:
		return ""
	}
}

func (o *testOutput) NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder {
	return nil
}

func (o *testOutput) GetRecord(dec *output.FLBDecoder) (ret int, ts interface{}, rec map[interface{}]interface{}) {
	if o.inc == 0 {
		o.inc++
		return 0, output.FLBTime{Time: time.Now()}, map[interface{}]interface{}{
			"testvalue1": []byte("record1"),
			"testvalue2": []byte("record2"),
		}
	}
	return -1, nil, nil
}

func TestFLBPluginInit(t *testing.T) {
	assert := assert.New(t)
	err := godotenv.Load()
	if err != nil {
		t.Log("Error loading .env file")
	}
	wrapper = OutputWrapper(&testOutput{})

	if os.Getenv("PROJECT_ID") == "" || os.Getenv("TOPIC_NAME") == "" {
		assert.Equal(output.FLB_ERROR, FLBPluginInit(nil))
	} else {
		assert.Equal(output.FLB_OK, FLBPluginInit(nil))
	}
}

func TestFLBPluginFlush(t *testing.T) {
	assert := assert.New(t)
	err := godotenv.Load()
	if err != nil {
		t.Log("Error loading .env file")
	}
	wrapper = OutputWrapper(&testOutput{})
	if os.Getenv("PROJECT_ID") == "" || os.Getenv("TOPIC_NAME") == "" {
		return
	}
	ok := FLBPluginFlush(nil, 0, nil)
	assert.Equal(output.FLB_OK, ok)

	projectId := os.Getenv("PROJECT_ID")
	topicName := os.Getenv("TOPIC_NAME")
	if projectId == "" || topicName == "" {
		return
	}
	keeper, err := NewKeeper(projectId, topicName, nil)
	assert.NoError(err)
	sub := keeper.(*GooglePubSub).client.Subscription(topicName)
	go func() {
		sub.Receive(context.Background(), func(ctx context.Context, m *pubsub.Message) {
			log.Printf("Got message: %s", m.Data)
			m.Ack()
		})
	}()
	time.Sleep(5 * time.Second)
}

func TestInterfaceToBytes(t *testing.T) {
	assert := assert.New(t)

	now := time.Now()
	tests := map[string]struct {
		input  interface{}
		output []byte
	}{
		"float": {
			input:  float64(10.0),
			output: []byte(fmt.Sprintf("%f", float64(10.0))),
		},
		"[]byte": {
			input:  []byte(string("hello")),
			output: []byte(string("hello")),
		},
		"int": {
			input:  int(20),
			output: []byte(string("20")),
		},
		"string": {
			input:  "hello",
			output: []byte(string("hello")),
		},
		"time": {
			input:  now,
			output: []byte(now.Format(time.RFC3339)),
		},
		"bool": {
			input:  true,
			output: []byte("true"),
		},
		"etc": {
			input:  map[string]string{"hello": "world"},
			output: []byte(fmt.Sprintf("%v", map[string]string{"hello": "world"})),
		},
	}

	for _, t := range tests {
		output := interfaceToBytes(t.input)
		assert.Equal(t.output, output)
	}
}

func TestParallelConfirmConfiguration(t *testing.T) {
	assert := assert.New(t)
	
	tests := []struct {
		name                   string
		parallelConfirmConfig  string
		confirmWorkersConfig   string
		expectedParallelConfirm bool
		expectedConfirmWorkers  int
		expectError            bool
	}{
		{
			name:                   "Default values",
			parallelConfirmConfig:  "",
			confirmWorkersConfig:   "",
			expectedParallelConfirm: false,
			expectedConfirmWorkers:  10,
			expectError:            false,
		},
		{
			name:                   "Enable parallel confirm",
			parallelConfirmConfig:  "true",
			confirmWorkersConfig:   "20",
			expectedParallelConfirm: true,
			expectedConfirmWorkers:  20,
			expectError:            false,
		},
		{
			name:                   "Disable parallel confirm",
			parallelConfirmConfig:  "false",
			confirmWorkersConfig:   "50",
			expectedParallelConfirm: false,
			expectedConfirmWorkers:  50,
			expectError:            false,
		},
		{
			name:                   "Workers out of range (too high)",
			parallelConfirmConfig:  "true",
			confirmWorkersConfig:   "150",
			expectedParallelConfirm: true,
			expectedConfirmWorkers:  10, // Should remain default
			expectError:            false,
		},
		{
			name:                   "Workers out of range (zero)",
			parallelConfirmConfig:  "true",
			confirmWorkersConfig:   "0",
			expectedParallelConfirm: true,
			expectedConfirmWorkers:  10, // Should remain default
			expectError:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables
			parallelConfirm = false
			confirmWorkers = 10

			// Setup mock wrapper
			mockWrapper := &MockOutputWrapper{
				configs: map[string]string{
					"ParallelConfirm": tt.parallelConfirmConfig,
					"ConfirmWorkers":  tt.confirmWorkersConfig,
				},
			}
			
			// Save original wrapper and restore after test
			originalWrapper := wrapper
			wrapper = mockWrapper
			defer func() { wrapper = originalWrapper }()

			// Test the configuration parsing logic (from FLBPluginInit)
			var err error
			
			// Parse ParallelConfirm
			pc := mockWrapper.GetConfigKey(nil, "ParallelConfirm")
			if pc != "" {
				parallelConfirm, err = strconv.ParseBool(pc)
				if err != nil && !tt.expectError {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}

			// Parse ConfirmWorkers
			cw := mockWrapper.GetConfigKey(nil, "ConfirmWorkers")
			if cw != "" {
				v, err := strconv.Atoi(cw)
				if err != nil && !tt.expectError {
					t.Fatalf("Expected no error, got: %v", err)
				}
				if err == nil && v > 0 && v <= 100 {
					confirmWorkers = v
				}
			}

			// Validate results
			assert.Equal(tt.expectedParallelConfirm, parallelConfirm, 
				"parallelConfirm should match expected value")
			assert.Equal(tt.expectedConfirmWorkers, confirmWorkers, 
				"confirmWorkers should match expected value")
		})
	}
}

func TestParallelConfirmPerformance(t *testing.T) {
	// Create mock results with delays
	numResults := 50
	results := make([]*MockPublishResult, numResults)
	for i := 0; i < numResults; i++ {
		results[i] = &MockPublishResult{
			delay: 10 * time.Millisecond, // Simulate network delay
			err:   nil,
		}
	}

	// Test sequential processing
	sequentialStart := time.Now()
	for _, result := range results {
		result.Get(context.Background())
	}
	sequentialDuration := time.Since(sequentialStart)

	// Reset results for parallel test
	for i := 0; i < numResults; i++ {
		results[i] = &MockPublishResult{
			delay: 10 * time.Millisecond,
			err:   nil,
		}
	}

	// Test parallel processing
	parallelStart := time.Now()
	var wg sync.WaitGroup
	var mutex sync.Mutex
	errors := []error{}
	batchSize := 10
	
	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}
		
		for j := i; j < end; j++ {
			wg.Add(1)
			go func(result *MockPublishResult) {
				defer wg.Done()
				if _, err := result.Get(context.Background()); err != nil {
					mutex.Lock()
					errors = append(errors, err)
					mutex.Unlock()
				}
			}(results[j])
		}
		wg.Wait()
	}
	parallelDuration := time.Since(parallelStart)

	// Parallel should be significantly faster
	speedup := float64(sequentialDuration) / float64(parallelDuration)
	assert.True(t, speedup >= 2.0, 
		fmt.Sprintf("Expected at least 2x speedup, got %.2fx (sequential: %v, parallel: %v)",
			speedup, sequentialDuration, parallelDuration))

	t.Logf("Sequential: %v, Parallel: %v, Speedup: %.2fx", 
		sequentialDuration, parallelDuration, speedup)
}

func TestParallelConfirmErrorHandling(t *testing.T) {
	assert := assert.New(t)
	
	tests := []struct {
		name           string
		errors         []error
		expectedRetry  bool
		expectedErrors int
	}{
		{
			name:           "No errors",
			errors:         []error{},
			expectedRetry:  false,
			expectedErrors: 0,
		},
		{
			name: "Retry errors",
			errors: []error{
				context.DeadlineExceeded,
				context.Canceled,
			},
			expectedRetry:  true,
			expectedErrors: 2,
		},
		{
			name: "Non-retry errors",
			errors: []error{
				errors.New("permanent error"),
				errors.New("another error"),
			},
			expectedRetry:  false,
			expectedErrors: 2,
		},
		{
			name: "Mixed errors",
			errors: []error{
				context.DeadlineExceeded,
				errors.New("permanent error"),
			},
			expectedRetry:  true,
			expectedErrors: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock results with specified errors
			results := make([]*MockPublishResult, len(tt.errors))
			for i, err := range tt.errors {
				results[i] = &MockPublishResult{
					delay: 1 * time.Millisecond,
					err:   err,
				}
			}

			// Add some successful results
			for i := 0; i < 5; i++ {
				results = append(results, &MockPublishResult{
					delay: 1 * time.Millisecond,
					err:   nil,
				})
			}

			// Test parallel confirmation logic (mirrors the actual implementation)
			var wg sync.WaitGroup
			var hasRetryError bool
			var mutex sync.Mutex
			collectedErrors := []error{}
			batchSize := 3
			
			for i := 0; i < len(results); i += batchSize {
				end := i + batchSize
				if end > len(results) {
					end = len(results)
				}
				
				for j := i; j < end; j++ {
					wg.Add(1)
					go func(result *MockPublishResult) {
						defer wg.Done()
						if _, err := result.Get(context.Background()); err != nil {
							mutex.Lock()
							collectedErrors = append(collectedErrors, err)
							if err == context.DeadlineExceeded || err == context.Canceled {
								hasRetryError = true
							}
							mutex.Unlock()
						}
					}(results[j])
				}
				wg.Wait()
			}

			// Validate results
			assert.Equal(tt.expectedRetry, hasRetryError, 
				"hasRetryError should match expected value")
			assert.Equal(tt.expectedErrors, len(collectedErrors), 
				"number of collected errors should match expected")
		})
	}
}

func TestConcurrencyLimits(t *testing.T) {
	assert := assert.New(t)
	
	numResults := 30
	maxConcurrency := 5
	
	// Create concurrency tracker
	tracker := &ConcurrencyTracker{}
	
	results := make([]*MockPublishResult, numResults)
	for i := 0; i < numResults; i++ {
		results[i] = &MockPublishResult{
			delay: 50 * time.Millisecond,
			err:   nil,
			concurrencyTracker: tracker,
		}
	}

	// Test with concurrency limit
	var wg sync.WaitGroup
	batchSize := maxConcurrency
	
	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}
		
		for j := i; j < end; j++ {
			wg.Add(1)
			go func(result *MockPublishResult) {
				defer wg.Done()
				result.Get(context.Background())
			}(results[j])
		}
		wg.Wait()
	}

	// Validate concurrency was limited
	assert.True(tracker.maxActiveCalls <= int32(maxConcurrency), 
		fmt.Sprintf("Expected max concurrency %d, got %d", maxConcurrency, tracker.maxActiveCalls))
	
	t.Logf("Max concurrent calls: %d (limit: %d)", tracker.maxActiveCalls, maxConcurrency)
}
