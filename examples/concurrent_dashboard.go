// ./examples/concurrent_dashboard.go
// This file demonstrates a Real-Time Concurrent Data Processing Dashboard using GoWebComponents.
// It showcases Go's concurrency features (goroutines, channels) in a web browser environment,
// demonstrating capabilities that would be complex or impossible in regular JavaScript.
//
// Key Features Demonstrated:
// - Multiple concurrent data streams processing simultaneously using goroutines
// - Real-time communication between goroutines using channels
// - Type-safe state management with GoUseState generics
// - Performance monitoring and visualization
// - Interactive controls for dynamic goroutine management
// - WebAssembly performance advantages for computational work

package examples

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"syscall/js"
	"time"

	// Dot import allows us to use fiber package functions directly
	. "github.com/monstercameron/GoWebComponents/fiber"
)

// StockPrice represents a single stock price data point for financial data simulation
type StockPrice struct {
	Symbol    string  `json:"symbol"`    // Stock symbol (e.g., "AAPL", "GOOGL")
	Price     float64 `json:"price"`     // Current price in dollars
	Change    float64 `json:"change"`    // Price change from previous
	Timestamp int64   `json:"timestamp"` // Unix timestamp when price was recorded
}

// LogEntry represents a single log entry for text processing simulation
type LogEntry struct {
	Level     string `json:"level"`     // Log level (INFO, WARN, ERROR, DEBUG)
	Message   string `json:"message"`   // The actual log message
	Source    string `json:"source"`    // Source component that generated the log
	Timestamp int64  `json:"timestamp"` // Unix timestamp when log was created
}

// ProcessingStats holds real-time performance metrics for the dashboard
type ProcessingStats struct {
	StockPricesProcessed    int     `json:"stockPricesProcessed"`    // Total stock prices processed
	LogEntriesProcessed     int     `json:"logEntriesProcessed"`     // Total log entries processed
	ItemsPerSecond          float64 `json:"itemsPerSecond"`          // Processing rate (items/second)
	ActiveGoroutines        int     `json:"activeGoroutines"`        // Number of active worker goroutines
	MemoryUsageMB           float64 `json:"memoryUsageMB"`           // Estimated memory usage in MB
	TotalProcessingTimeMs   int64   `json:"totalProcessingTimeMs"`   // Total processing time in milliseconds
	AverageProcessingTimeMs float64 `json:"averageProcessingTimeMs"` // Average processing time per item
}

// WorkerPool manages a pool of goroutines for concurrent processing
type WorkerPool struct {
	workerCount      int                  // Number of worker goroutines
	stockDataChannel chan StockPrice      // Channel for stock price data
	logDataChannel   chan LogEntry        // Channel for log entry data
	resultChannel    chan ProcessingStats // Channel for processing results
	stopChannel      chan bool            // Channel to signal workers to stop
	isRunning        bool                 // Flag indicating if the pool is running
	mutex            sync.RWMutex         // Mutex for thread-safe access to isRunning
}

// ProcessorType represents the type of data processor
type ProcessorType int

const (
	StockProcessor ProcessorType = iota // Stock price processor
	LogProcessor                        // Log entry processor
)

// stockSymbolsBase holds the base prices for stock symbols (moved outside component to prevent recreation)
var stockSymbolsBase = map[string]float64{
	"AAPL":  150.0,
	"GOOGL": 120.0,
	"MSFT":  300.0,
	"TSLA":  200.0,
	"AMZN":  140.0,
}

// DashboardHeader creates the main header component for the concurrent dashboard
// It displays the title, description, and key features of the demonstration
func DashboardHeader(props Attrs) *Element {
	return Header(Attrs{
		"class": "bg-gradient-to-r from-gray-900 via-gray-800 to-gray-900 text-gray-100 p-6 shadow-xl border-b border-gray-700",
	},
		Div(Attrs{
			"class": "max-w-7xl mx-auto",
		},
			// Main title with icon
			H1(Attrs{
				"class": "text-4xl font-bold mb-3 flex items-center text-white",
			},
				Span(Attrs{
					"class": "text-5xl mr-4",
				}, Text("⚡")),
				Text("Concurrent Data Processing Dashboard"),
			),

			// Subtitle description
			P(Attrs{
				"class": "text-xl text-gray-300 mb-4 leading-relaxed",
			}, Text("Real-time demonstration of Go's concurrency superpowers in WebAssembly")),

			// Feature highlights
			Div(Attrs{
				"class": "flex flex-wrap gap-6 text-sm",
			},
				// Goroutines feature
				Div(Attrs{
					"class": "flex items-center bg-gray-800/50 px-3 py-2 rounded-lg border border-gray-600",
				},
					Span(Attrs{
						"class": "text-lg mr-2",
					}, Text("🔄")),
					Text("Multiple Goroutines"),
				),

				// Channels feature
				Div(Attrs{
					"class": "flex items-center bg-gray-800/50 px-3 py-2 rounded-lg border border-gray-600",
				},
					Span(Attrs{
						"class": "text-lg mr-2",
					}, Text("📡")),
					Text("Channel Communication"),
				),

				// Type Safety feature
				Div(Attrs{
					"class": "flex items-center bg-gray-800/50 px-3 py-2 rounded-lg border border-gray-600",
				},
					Span(Attrs{
						"class": "text-lg mr-2",
					}, Text("🛡️")),
					Text("Type-Safe Generics"),
				),

				// Performance feature
				Div(Attrs{
					"class": "flex items-center bg-gray-800/50 px-3 py-2 rounded-lg border border-gray-600",
				},
					Span(Attrs{
						"class": "text-lg mr-2",
					}, Text("🚀")),
					Text("WebAssembly Performance"),
				),
			),
		),
	)
}

// DashboardControlPanel creates the control panel for managing concurrent workers
// It provides buttons to start/stop processing and sliders to adjust worker counts
func DashboardControlPanel(props Attrs) *Element {
	// Extract handler functions from props
	var onStartProcessing, onStopProcessing, onWorkerCountChange interface{}
	var isProcessingActive bool
	var currentWorkerCount int

	if props != nil {
		if handler, exists := props["onStartProcessing"]; exists {
			onStartProcessing = handler
		}
		if handler, exists := props["onStopProcessing"]; exists {
			onStopProcessing = handler
		}
		if handler, exists := props["onWorkerCountChange"]; exists {
			onWorkerCountChange = handler
		}
		if active, exists := props["isProcessingActive"]; exists {
			isProcessingActive = active.(bool)
		}
		if count, exists := props["currentWorkerCount"]; exists {
			currentWorkerCount = count.(int)
		}
	}

	return Section(Attrs{
		"class": "bg-gray-800 rounded-lg shadow-xl p-6 mb-6 border border-gray-700",
	},
		// Control panel header
		H2(Attrs{
			"class": "text-2xl font-bold text-gray-100 mb-4 flex items-center",
		},
			Span(Attrs{
				"class": "text-2xl mr-3",
			}, Text("🎛️")),
			Text("Processing Controls"),
		),

		// Control buttons and settings container
		Div(Attrs{
			"class": "flex flex-col md:flex-row gap-6 items-start md:items-center",
		},
			// Start/Stop buttons section
			Div(Attrs{
				"class": "flex gap-3",
			},
				// Start processing button
				Button(Attrs{
					"onclick": onStartProcessing,
					"disabled": func() interface{} {
						if isProcessingActive {
							return true
						}
						return nil
					}(),
					"class": func() string {
						baseClasses := "px-6 py-3 rounded-lg font-semibold transition-all duration-200 flex items-center"
						if isProcessingActive {
							return baseClasses + " bg-gray-600 text-gray-400 cursor-not-allowed"
						}
						return baseClasses + " bg-green-600 hover:bg-green-500 text-white shadow-lg hover:shadow-xl transform hover:scale-105"
					}(),
				},
					Span(Attrs{
						"class": "text-lg mr-2",
					}, Text("▶️")),
					Text("Start Processing"),
				),

				// Stop processing button
				Button(Attrs{
					"onclick": onStopProcessing,
					"disabled": func() interface{} {
						if !isProcessingActive {
							return true
						}
						return nil
					}(),
					"class": func() string {
						baseClasses := "px-6 py-3 rounded-lg font-semibold transition-all duration-200 flex items-center"
						if !isProcessingActive {
							return baseClasses + " bg-gray-600 text-gray-400 cursor-not-allowed"
						}
						return baseClasses + " bg-red-600 hover:bg-red-500 text-white shadow-lg hover:shadow-xl transform hover:scale-105"
					}(),
				},
					Span(Attrs{
						"class": "text-lg mr-2",
					}, Text("⏹️")),
					Text("Stop Processing"),
				),
			),

			// Worker count controls section
			Div(Attrs{
				"class": "flex flex-col gap-2",
			},
				Label(Attrs{
					"class": "text-sm font-medium text-gray-300",
				}, Text("Worker Goroutines")),

				Div(Attrs{
					"class": "flex items-center gap-3",
				},
					// Worker count slider
					Input(Attrs{
						"type":     "range",
						"min":      "1",
						"max":      "10",
						"value":    fmt.Sprintf("%d", currentWorkerCount),
						"onchange": onWorkerCountChange,
						"class":    "w-32 h-2 bg-gray-600 rounded-lg appearance-none cursor-pointer slider",
					}),

					// Current count display
					Span(Attrs{
						"class": "text-lg font-bold text-blue-400 bg-gray-700 px-3 py-1 rounded-lg min-w-[3rem] text-center border border-gray-600",
					}, Text(fmt.Sprintf("%d", currentWorkerCount))),
				),

				// Worker count description
				P(Attrs{
					"class": "text-xs text-gray-400",
				}, Text("Adjust the number of concurrent worker goroutines")),
			),

			// Processing status indicator
			Div(Attrs{
				"class": "flex items-center gap-2",
			},
				Div(Attrs{
					"class": func() string {
						baseClasses := "w-3 h-3 rounded-full"
						if isProcessingActive {
							return baseClasses + " bg-green-400 animate-pulse"
						}
						return baseClasses + " bg-gray-400"
					}(),
				}),
				Span(Attrs{
					"class": func() string {
						if isProcessingActive {
							return "text-green-400 font-medium"
						}
						return "text-gray-400"
					}(),
				}, Text(func() string {
					if isProcessingActive {
						return "Processing Active"
					}
					return "Processing Stopped"
				}())),
			),
		),
	)
}

// MetricsDisplayCard creates a single metric card component
// It displays a metric value with an icon, label, and description
func MetricsDisplayCard(props Attrs) *Element {
	var iconEmoji, label, value, description, colorClass string

	if props != nil {
		if emoji, exists := props["icon"]; exists {
			iconEmoji = emoji.(string)
		}
		if l, exists := props["label"]; exists {
			label = l.(string)
		}
		if v, exists := props["value"]; exists {
			value = v.(string)
		}
		if desc, exists := props["description"]; exists {
			description = desc.(string)
		}
		if color, exists := props["colorClass"]; exists {
			colorClass = color.(string)
		}
	}

	// Default color class if not provided
	if colorClass == "" {
		colorClass = "text-blue-600"
	}

	return Div(Attrs{
		"class": "bg-gray-800 rounded-lg shadow-xl p-6 border-l-4 border-blue-400 border border-gray-700",
	},
		// Metric header with icon and label
		Div(Attrs{
			"class": "flex items-center justify-between mb-2",
		},
			Div(Attrs{
				"class": "flex items-center",
			},
				Span(Attrs{
					"class": "text-2xl mr-3",
				}, Text(iconEmoji)),
				H3(Attrs{
					"class": "text-lg font-semibold text-gray-200",
				}, Text(label)),
			),
		),

		// Metric value display
		Div(Attrs{
			"class": "mb-2",
		},
			P(Attrs{
				"class": fmt.Sprintf("text-3xl font-bold %s", colorClass),
			}, Text(value)),
		),

		// Metric description
		P(Attrs{
			"class": "text-sm text-gray-400",
		}, Text(description)),
	)
}

// DashboardMetricsPanel creates the metrics display panel
// It shows real-time processing statistics using multiple metric cards
func DashboardMetricsPanel(props Attrs) *Element {
	var processingStats ProcessingStats

	if props != nil {
		if stats, exists := props["processingStats"]; exists {
			processingStats = stats.(ProcessingStats)
		}
	}

	return Section(Attrs{
		"class": "mb-6",
	},
		// Metrics panel header
		H2(Attrs{
			"class": "text-2xl font-bold text-gray-100 mb-6 flex items-center",
		},
			Span(Attrs{
				"class": "text-2xl mr-3",
			}, Text("📊")),
			Text("Real-Time Processing Metrics"),
		),

		// Metrics grid
		Div(Attrs{
			"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6",
		},
			// Stock prices processed metric
			MetricsDisplayCard(Attrs{
				"icon":        "📈",
				"label":       "Stock Prices",
				"value":       fmt.Sprintf("%d", processingStats.StockPricesProcessed),
				"description": "Total stock prices processed",
				"colorClass":  "text-green-400",
			}),

			// Log entries processed metric
			MetricsDisplayCard(Attrs{
				"icon":        "📝",
				"label":       "Log Entries",
				"value":       fmt.Sprintf("%d", processingStats.LogEntriesProcessed),
				"description": "Total log entries processed",
				"colorClass":  "text-blue-400",
			}),

			// Processing rate metric
			MetricsDisplayCard(Attrs{
				"icon":        "⚡",
				"label":       "Items/Second",
				"value":       fmt.Sprintf("%.1f", processingStats.ItemsPerSecond),
				"description": "Current processing rate",
				"colorClass":  "text-purple-400",
			}),

			// Active goroutines metric
			MetricsDisplayCard(Attrs{
				"icon":        "🔄",
				"label":       "Active Workers",
				"value":       fmt.Sprintf("%d", processingStats.ActiveGoroutines),
				"description": "Number of active goroutines",
				"colorClass":  "text-orange-400",
			}),
		),

		// Secondary metrics row
		Div(Attrs{
			"class": "grid grid-cols-1 md:grid-cols-3 gap-6 mt-6",
		},
			// Memory usage metric
			MetricsDisplayCard(Attrs{
				"icon":        "💾",
				"label":       "Memory Usage",
				"value":       fmt.Sprintf("%.1f MB", processingStats.MemoryUsageMB),
				"description": "Estimated memory consumption",
				"colorClass":  "text-red-400",
			}),

			// Total processing time metric
			MetricsDisplayCard(Attrs{
				"icon":        "⏱️",
				"label":       "Total Time",
				"value":       fmt.Sprintf("%d ms", processingStats.TotalProcessingTimeMs),
				"description": "Total processing time",
				"colorClass":  "text-indigo-400",
			}),

			// Average processing time metric
			MetricsDisplayCard(Attrs{
				"icon":        "⏳",
				"label":       "Avg Time",
				"value":       fmt.Sprintf("%.2f ms", processingStats.AverageProcessingTimeMs),
				"description": "Average time per item",
				"colorClass":  "text-teal-400",
			}),
		),
	)
}

// LiveDataStreamCard creates a card displaying live data entries
// It shows recent data items with timestamps and processing status
func LiveDataStreamCard(props Attrs) *Element {
	var title, streamType, iconEmoji string
	var dataItems []interface{}
	var maxDisplayItems int = 5

	if props != nil {
		if t, exists := props["title"]; exists {
			title = t.(string)
		}
		if st, exists := props["streamType"]; exists {
			streamType = st.(string)
		}
		if icon, exists := props["icon"]; exists {
			iconEmoji = icon.(string)
		}
		if items, exists := props["dataItems"]; exists {
			dataItems = items.([]interface{})
		}
		if max, exists := props["maxDisplayItems"]; exists {
			maxDisplayItems = max.(int)
		}
	}

	// Limit displayed items to most recent ones
	displayItems := dataItems
	if len(dataItems) > maxDisplayItems {
		displayItems = dataItems[len(dataItems)-maxDisplayItems:]
	}

	return Div(Attrs{
		"class": "bg-gray-800 rounded-lg shadow-xl p-6 border border-gray-700",
	},
		// Stream header
		Div(Attrs{
			"class": "flex items-center justify-between mb-4",
		},
			H3(Attrs{
				"class": "text-xl font-bold text-gray-100 flex items-center",
			},
				Span(Attrs{
					"class": "text-xl mr-3",
				}, Text(iconEmoji)),
				Text(title),
			),

			// Stream type badge
			Span(Attrs{
				"class": "px-3 py-1 bg-blue-900/50 text-blue-300 text-sm font-medium rounded-full border border-blue-700",
			}, Text(streamType)),
		),

		// Data items list
		Div(Attrs{
			"class": "space-y-2 max-h-60 overflow-y-auto",
		},
			func() []interface{} {
				var elements []interface{}

				if len(displayItems) == 0 {
					// Show empty state
					elements = append(elements, Div(Attrs{
						"class": "text-center py-8 text-gray-400",
					},
						Div(Attrs{
							"class": "text-4xl mb-2",
						}, Text("⏳")),
						P(nil, Text("Waiting for data...")),
					))
				} else {
					// Show data items
					for i := len(displayItems) - 1; i >= 0; i-- { // Reverse order to show newest first
						item := displayItems[i]

						if streamType == "stock" {
							if stockPrice, ok := item.(StockPrice); ok {
								elements = append(elements, createStockPriceItem(stockPrice))
							}
						} else if streamType == "logs" {
							if logEntry, ok := item.(LogEntry); ok {
								elements = append(elements, createLogEntryItem(logEntry))
							}
						}
					}
				}

				return elements
			}()...,
		),

		// Footer with total count
		Div(Attrs{
			"class": "mt-4 pt-4 border-t border-gray-600",
		},
			P(Attrs{
				"class": "text-sm text-gray-400 text-center",
			}, Text(fmt.Sprintf("Showing %d of %d items", len(displayItems), len(dataItems)))),
		),
	)
}

// createStockPriceItem creates a single stock price display item
func createStockPriceItem(stockPrice StockPrice) *Element {
	// Format timestamp
	timestamp := time.Unix(stockPrice.Timestamp, 0).Format("15:04:05")

	// Determine price change color
	changeColor := "text-gray-400"
	changeSymbol := ""
	if stockPrice.Change > 0 {
		changeColor = "text-green-400"
		changeSymbol = "+"
	} else if stockPrice.Change < 0 {
		changeColor = "text-red-400"
	}

	return Div(Attrs{
		"class": "bg-gray-700 rounded-lg p-3 flex items-center justify-between hover:bg-gray-600 transition-colors border border-gray-600",
	},
		// Stock symbol and price
		Div(Attrs{
			"class": "flex items-center",
		},
			Span(Attrs{
				"class": "font-bold text-gray-200",
			}, Text(stockPrice.Symbol)),
			Span(Attrs{
				"class": "ml-3 text-lg font-semibold text-gray-100",
			}, Text(fmt.Sprintf("$%.2f", stockPrice.Price))),
		),

		// Price change and timestamp
		Div(Attrs{
			"class": "flex items-center gap-3",
		},
			Span(Attrs{
				"class": fmt.Sprintf("text-sm font-medium %s", changeColor),
			}, Text(fmt.Sprintf("%s%.2f", changeSymbol, stockPrice.Change))),
			Span(Attrs{
				"class": "text-xs text-gray-400",
			}, Text(timestamp)),
		),
	)
}

// createLogEntryItem creates a single log entry display item
func createLogEntryItem(logEntry LogEntry) *Element {
	// Format timestamp
	timestamp := time.Unix(logEntry.Timestamp, 0).Format("15:04:05")

	// Determine log level color and icon
	var levelColor, levelIcon string
	switch logEntry.Level {
	case "ERROR":
		levelColor = "text-red-300 bg-red-900/50 border-red-700"
		levelIcon = "❌"
	case "WARN":
		levelColor = "text-yellow-300 bg-yellow-900/50 border-yellow-700"
		levelIcon = "⚠️"
	case "INFO":
		levelColor = "text-blue-300 bg-blue-900/50 border-blue-700"
		levelIcon = "ℹ️"
	case "DEBUG":
		levelColor = "text-gray-300 bg-gray-700/50 border-gray-600"
		levelIcon = "🔍"
	default:
		levelColor = "text-gray-300 bg-gray-700/50 border-gray-600"
		levelIcon = "📝"
	}

	return Div(Attrs{
		"class": "bg-gray-700 rounded-lg p-3 hover:bg-gray-600 transition-colors border border-gray-600",
	},
		// Log header with level and timestamp
		Div(Attrs{
			"class": "flex items-center justify-between mb-2",
		},
			Div(Attrs{
				"class": "flex items-center gap-2",
			},
				Span(Attrs{
					"class": "text-sm",
				}, Text(levelIcon)),
				Span(Attrs{
					"class": fmt.Sprintf("px-2 py-1 rounded text-xs font-medium border %s", levelColor),
				}, Text(logEntry.Level)),
				Span(Attrs{
					"class": "text-xs text-gray-400",
				}, Text(logEntry.Source)),
			),
			Span(Attrs{
				"class": "text-xs text-gray-400",
			}, Text(timestamp)),
		),

		// Log message
		P(Attrs{
			"class": "text-sm text-gray-200 leading-relaxed",
		}, Text(logEntry.Message)),
	)
}

// DashboardDataVisualization creates the data visualization panel
// It displays live data streams in side-by-side cards
func DashboardDataVisualization(props Attrs) *Element {
	var stockPrices []interface{}
	var logEntries []interface{}

	if props != nil {
		if stocks, exists := props["stockPrices"]; exists {
			stockPrices = stocks.([]interface{})
		}
		if logs, exists := props["logEntries"]; exists {
			logEntries = logs.([]interface{})
		}
	}

	return Section(Attrs{
		"class": "mb-6",
	},
		// Visualization panel header
		H2(Attrs{
			"class": "text-2xl font-bold text-gray-100 mb-6 flex items-center",
		},
			Span(Attrs{
				"class": "text-2xl mr-3",
			}, Text("📺")),
			Text("Live Data Streams"),
		),

		// Data streams grid
		Div(Attrs{
			"class": "grid grid-cols-1 lg:grid-cols-2 gap-6",
		},
			// Stock prices stream
			LiveDataStreamCard(Attrs{
				"title":           "Stock Market Data",
				"streamType":      "stock",
				"icon":            "📈",
				"dataItems":       stockPrices,
				"maxDisplayItems": 6,
			}),

			// Log entries stream
			LiveDataStreamCard(Attrs{
				"title":           "System Logs",
				"streamType":      "logs",
				"icon":            "📄",
				"dataItems":       logEntries,
				"maxDisplayItems": 6,
			}),
		),
	)
}

// generateRandomStockPrice creates a realistic stock price with random fluctuations
// This simulates real market data for demonstration purposes
func generateRandomStockPrice(symbol string, basePrice float64) StockPrice {
	// Generate realistic price change between -5% and +5%
	changePercent := (rand.Float64() - 0.5) * 0.1 // -0.05 to +0.05
	priceChange := basePrice * changePercent
	newPrice := basePrice + priceChange

	return StockPrice{
		Symbol:    symbol,
		Price:     newPrice,
		Change:    priceChange,
		Timestamp: time.Now().Unix(),
	}
}

// generateRandomLogEntry creates a realistic log entry with random levels and messages
// This simulates system log generation for demonstration purposes
func generateRandomLogEntry() LogEntry {
	// Predefined log levels with different probabilities
	levels := []string{"INFO", "INFO", "INFO", "DEBUG", "DEBUG", "WARN", "ERROR"} // INFO most common
	sources := []string{"WebServer", "Database", "Cache", "Authentication", "FileSystem", "Network"}

	// Predefined message templates for realistic logs
	messageTemplates := map[string][]string{
		"INFO": {
			"Request processed successfully",
			"User authentication completed",
			"Cache hit for key: %s",
			"Database connection established",
			"File uploaded successfully",
			"Background task completed",
		},
		"DEBUG": {
			"Processing request from %s",
			"Cache miss for key: %s",
			"SQL query executed in %dms",
			"Memory usage: %d MB",
			"Thread pool size: %d",
		},
		"WARN": {
			"High memory usage detected",
			"Slow query detected: %dms",
			"Connection pool nearly exhausted",
			"Disk space running low",
			"Rate limit threshold approached",
		},
		"ERROR": {
			"Database connection failed",
			"Authentication failed for user",
			"File system error occurred",
			"Network timeout after %ds",
			"Memory allocation failed",
		},
	}

	// Select random level and source
	level := levels[rand.Intn(len(levels))]
	source := sources[rand.Intn(len(sources))]

	// Select random message template and format it
	templates := messageTemplates[level]
	template := templates[rand.Intn(len(templates))]

	// Format message with random values
	var message string
	switch {
	case strings.Contains(template, "%s"):
		randomValues := []string{"user123", "session456", "file789", "cache_key_abc"}
		message = fmt.Sprintf(template, randomValues[rand.Intn(len(randomValues))])
	case strings.Contains(template, "%d"):
		randomNumber := rand.Intn(1000) + 1
		message = fmt.Sprintf(template, randomNumber)
	default:
		message = template
	}

	return LogEntry{
		Level:     level,
		Message:   message,
		Source:    source,
		Timestamp: time.Now().Unix(),
	}
}

// stockPriceWorker is a goroutine worker that processes stock price data
// It demonstrates Go's concurrent processing capabilities
func stockPriceWorker(workerID int, stockDataChannel <-chan StockPrice, resultChannel chan<- ProcessingStats, stopChannel <-chan bool) {
	fmt.Printf("Stock price worker %d started\n", workerID)
	processedCount := 0
	startTime := time.Now()

	for {
		select {
		case <-stopChannel:
			// Worker received stop signal
			fmt.Printf("Stock price worker %d stopping, processed %d items\n", workerID, processedCount)
			return

		case stockPrice := <-stockDataChannel:
			// Process the stock price data
			processedCount++

			// Simulate some processing work (price validation, calculations, etc.)
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)+10)) // 10-60ms processing time

			// Simulate more complex processing for demonstration
			_ = stockPrice.Price * 1.1 // Simple calculation
			_ = fmt.Sprintf("Processed %s at $%.2f", stockPrice.Symbol, stockPrice.Price)

			// Send processing stats update more frequently for demo
			if processedCount%1 == 0 { // Send stats after every item for real-time updates
				elapsedTime := time.Since(startTime)
				avgProcessingTime := float64(elapsedTime.Milliseconds()) / float64(processedCount)

				stats := ProcessingStats{
					StockPricesProcessed:    processedCount,
					ItemsPerSecond:          float64(processedCount) / elapsedTime.Seconds(),
					ActiveGoroutines:        1, // This worker
					AverageProcessingTimeMs: avgProcessingTime,
					TotalProcessingTimeMs:   elapsedTime.Milliseconds(),
					MemoryUsageMB:           float64(processedCount) * 0.001, // Estimate: 1KB per item
				}

				// Non-blocking send to avoid deadlock
				select {
				case resultChannel <- stats:
					fmt.Printf("📊 Stock worker %d: Sent stats - processed %d items\n", workerID, processedCount)
				default:
					fmt.Printf("⚠️ Stock worker %d: Stats channel full, dropped update\n", workerID)
				}
			}
		}
	}
}

// logEntryWorker is a goroutine worker that processes log entry data
// It demonstrates concurrent text processing and analysis
func logEntryWorker(workerID int, logDataChannel <-chan LogEntry, resultChannel chan<- ProcessingStats, stopChannel <-chan bool) {
	fmt.Printf("Log entry worker %d started\n", workerID)
	processedCount := 0
	startTime := time.Now()

	for {
		select {
		case <-stopChannel:
			// Worker received stop signal
			fmt.Printf("Log entry worker %d stopping, processed %d items\n", workerID, processedCount)
			return

		case logEntry := <-logDataChannel:
			// Process the log entry data
			processedCount++

			// Simulate log processing work (parsing, categorization, indexing, etc.)
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(30)+5)) // 5-35ms processing time

			// Simulate more complex log analysis for demonstration
			messageLength := len(logEntry.Message)
			_ = strings.ToUpper(logEntry.Level)
			_ = fmt.Sprintf("Log from %s: %d chars", logEntry.Source, messageLength)

			// Send processing stats update more frequently for demo
			if processedCount%1 == 0 { // Send stats after every item for real-time updates
				elapsedTime := time.Since(startTime)
				avgProcessingTime := float64(elapsedTime.Milliseconds()) / float64(processedCount)

				stats := ProcessingStats{
					LogEntriesProcessed:     processedCount,
					ItemsPerSecond:          float64(processedCount) / elapsedTime.Seconds(),
					ActiveGoroutines:        1, // This worker
					AverageProcessingTimeMs: avgProcessingTime,
					TotalProcessingTimeMs:   elapsedTime.Milliseconds(),
					MemoryUsageMB:           float64(processedCount) * 0.0005, // Estimate: 0.5KB per log entry
				}

				// Non-blocking send to avoid deadlock
				select {
				case resultChannel <- stats:
				default:
				}
			}
		}
	}
}

// NewWorkerPool creates a new worker pool for concurrent data processing
// It initializes channels and sets up the infrastructure for goroutine management
func NewWorkerPool(workerCount int) *WorkerPool {
	return &WorkerPool{
		workerCount:      workerCount,
		stockDataChannel: make(chan StockPrice, 100),     // Buffered channel for stock data
		logDataChannel:   make(chan LogEntry, 100),       // Buffered channel for log data
		resultChannel:    make(chan ProcessingStats, 50), // Buffered channel for results
		stopChannel:      make(chan bool),                // Unbuffered stop signal channel
		isRunning:        false,
	}
}

// StartWorkers launches the specified number of worker goroutines
// It demonstrates Go's goroutine creation and management capabilities
func (wp *WorkerPool) StartWorkers() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if wp.isRunning {
		fmt.Println("Workers already running")
		return
	}

	wp.isRunning = true
	fmt.Printf("Starting %d worker goroutines\n", wp.workerCount)

	// Launch stock price workers (half of total workers)
	stockWorkers := wp.workerCount / 2
	if stockWorkers == 0 {
		stockWorkers = 1
	}

	for i := 0; i < stockWorkers; i++ {
		go stockPriceWorker(i+1, wp.stockDataChannel, wp.resultChannel, wp.stopChannel)
	}

	// Launch log entry workers (remaining workers)
	logWorkers := wp.workerCount - stockWorkers
	for i := 0; i < logWorkers; i++ {
		go logEntryWorker(i+1, wp.logDataChannel, wp.resultChannel, wp.stopChannel)
	}

	fmt.Printf("Started %d stock workers and %d log workers\n", stockWorkers, logWorkers)
}

// StopWorkers gracefully shuts down all worker goroutines
// It demonstrates proper cleanup of concurrent resources
func (wp *WorkerPool) StopWorkers() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if !wp.isRunning {
		fmt.Println("Workers are not running")
		return
	}

	fmt.Println("Stopping all worker goroutines...")

	// Send stop signal to all workers
	for i := 0; i < wp.workerCount; i++ {
		wp.stopChannel <- true
	}

	wp.isRunning = false
	fmt.Println("All workers stopped")
}

// IsRunning returns whether the worker pool is currently active
// Thread-safe getter method
func (wp *WorkerPool) IsRunning() bool {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	return wp.isRunning
}

// SendStockPrice sends a stock price to the processing queue
// Demonstrates channel-based communication between goroutines
func (wp *WorkerPool) SendStockPrice(stockPrice StockPrice) {
	if wp.IsRunning() {
		// Non-blocking send to avoid UI freezing
		select {
		case wp.stockDataChannel <- stockPrice:
		default:
			fmt.Println("Stock data channel full, dropping data point")
		}
	}
}

// SendLogEntry sends a log entry to the processing queue
// Demonstrates channel-based communication between goroutines
func (wp *WorkerPool) SendLogEntry(logEntry LogEntry) {
	if wp.IsRunning() {
		// Non-blocking send to avoid UI freezing
		select {
		case wp.logDataChannel <- logEntry:
		default:
			fmt.Println("Log data channel full, dropping log entry")
		}
	}
}

// GetResultChannel returns the channel for receiving processing results
// This allows the UI to receive real-time updates from workers
func (wp *WorkerPool) GetResultChannel() <-chan ProcessingStats {
	return wp.resultChannel
}

// ConcurrentDashboard is the main dashboard component that demonstrates
// Go's concurrency features using GoWebComponents fiber hooks
func ConcurrentDashboard(props Attrs) *Element {
	// State management using GoUseState with type-safe generics
	currentWorkerCount, setCurrentWorkerCount := GoUseState[int](3)
	isProcessingActive, setIsProcessingActive := GoUseState[bool](false)
	processingStats, setProcessingStats := GoUseState[ProcessingStats](ProcessingStats{})
	stockPricesData, setStockPricesData := GoUseState[[]interface{}]([]interface{}{})
	logEntriesData, setLogEntriesData := GoUseState[[]interface{}]([]interface{}{})
	workerPool, setWorkerPool := GoUseState[*WorkerPool](NewWorkerPool(currentWorkerCount()))

	// Create a working copy of stock symbols using memoization to prevent recreation
	stockSymbols := GoUseMemo(func() interface{} {
		symbols := make(map[string]float64)
		for k, v := range stockSymbolsBase {
			symbols[k] = v
		}
		return symbols
	}, []interface{}{}).(map[string]float64)

	// Data generation effect - runs when processing is active
	GoUseEffect(func() {
		if !isProcessingActive() {
			return
		}

		fmt.Println("Starting data generation goroutines...")

		// Create ticker for stock price generation (balanced for demo responsiveness)
		stockTicker := time.NewTicker(time.Millisecond * 800) // Generate stock prices every 800ms
		logTicker := time.NewTicker(time.Millisecond * 600)   // Generate log entries every 600ms

		// Stock price generation goroutine
		go func() {
			for {
				select {
				case <-stockTicker.C:
					if !isProcessingActive() {
						stockTicker.Stop()
						return
					}

					// Generate random stock price for random symbol
					symbols := make([]string, 0, len(stockSymbols))
					for symbol := range stockSymbols {
						symbols = append(symbols, symbol)
					}

					randomSymbol := symbols[rand.Intn(len(symbols))]
					basePrice := stockSymbols[randomSymbol]
					newStockPrice := generateRandomStockPrice(randomSymbol, basePrice)

					// Update base price for next generation (simulate market movement)
					stockSymbols[randomSymbol] = newStockPrice.Price

					// Send to worker pool for processing
					pool := workerPool()
					if pool != nil {
						pool.SendStockPrice(newStockPrice)
					}

					// Update UI state with new stock price
					currentStockPrices := stockPricesData()
					updatedStockPrices := append(currentStockPrices, newStockPrice)

					// Keep only last 50 items for performance
					if len(updatedStockPrices) > 50 {
						updatedStockPrices = updatedStockPrices[len(updatedStockPrices)-50:]
					}

					setStockPricesData(updatedStockPrices)
				}
			}
		}()

		// Log entry generation goroutine
		go func() {
			for {
				select {
				case <-logTicker.C:
					if !isProcessingActive() {
						logTicker.Stop()
						return
					}

					// Generate random log entry
					newLogEntry := generateRandomLogEntry()

					// Send to worker pool for processing
					pool := workerPool()
					if pool != nil {
						pool.SendLogEntry(newLogEntry)
					}

					// Update UI state with new log entry
					currentLogEntries := logEntriesData()
					updatedLogEntries := append(currentLogEntries, newLogEntry)

					// Keep only last 50 items for performance
					if len(updatedLogEntries) > 50 {
						updatedLogEntries = updatedLogEntries[len(updatedLogEntries)-50:]
					}

					setLogEntriesData(updatedLogEntries)
				}
			}
		}()

		// Statistics collection goroutine
		go func() {
			pool := workerPool()
			if pool == nil {
				return
			}

			resultChannel := pool.GetResultChannel()

			for {
				select {
				case stats := <-resultChannel:
					if !isProcessingActive() {
						return
					}

					fmt.Printf("📈 Stats received: Stock=%d, Logs=%d, Rate=%.1f/s\n",
						stats.StockPricesProcessed, stats.LogEntriesProcessed, stats.ItemsPerSecond)

					// Aggregate stats and update UI
					currentStats := processingStats()

					// Combine statistics from different workers
					combinedStats := ProcessingStats{
						StockPricesProcessed:    currentStats.StockPricesProcessed + stats.StockPricesProcessed,
						LogEntriesProcessed:     currentStats.LogEntriesProcessed + stats.LogEntriesProcessed,
						ItemsPerSecond:          (currentStats.ItemsPerSecond + stats.ItemsPerSecond) / 2, // Average
						ActiveGoroutines:        currentWorkerCount(),
						MemoryUsageMB:           currentStats.MemoryUsageMB + stats.MemoryUsageMB,
						TotalProcessingTimeMs:   currentStats.TotalProcessingTimeMs + stats.TotalProcessingTimeMs,
						AverageProcessingTimeMs: (currentStats.AverageProcessingTimeMs + stats.AverageProcessingTimeMs) / 2,
					}

					fmt.Printf("🔄 UI Update: Combined Stock=%d, Logs=%d, Rate=%.1f/s\n",
						combinedStats.StockPricesProcessed, combinedStats.LogEntriesProcessed, combinedStats.ItemsPerSecond)

					setProcessingStats(combinedStats)
				}
			}
		}()

		// Cleanup function (commented out as GoUseEffect doesn't support cleanup returns in this implementation)
		// The tickers will be stopped when the goroutines check isProcessingActive()
		fmt.Println("Data generation goroutines started")
	}, []interface{}{isProcessingActive()})

	// Event handlers using GoUseFunc for type-safe event handling
	handleStartProcessing := GoUseFunc(func(event GoEvent) {
		fmt.Println("Starting processing...")
		setIsProcessingActive(true)

		// Start worker pool
		pool := workerPool()
		if pool != nil {
			pool.StartWorkers()
		}
	})

	handleStopProcessing := GoUseFunc(func(event GoEvent) {
		fmt.Println("Stopping processing...")
		setIsProcessingActive(false)

		// Stop worker pool
		pool := workerPool()
		if pool != nil {
			pool.StopWorkers()
		}

		// Reset statistics
		setProcessingStats(ProcessingStats{})
	})

	handleWorkerCountChange := GoUseFunc(func(event GoEvent) {
		newCountStr := event.GetValue()
		newCount := 1
		if countVal, err := fmt.Sscanf(newCountStr, "%d", &newCount); err == nil && countVal == 1 {
			fmt.Printf("Changing worker count to %d\n", newCount)

			// Stop current workers if running
			if isProcessingActive() {
				pool := workerPool()
				if pool != nil {
					pool.StopWorkers()
				}
			}

			// Create new worker pool with updated count
			newPool := NewWorkerPool(newCount)
			setWorkerPool(newPool)
			setCurrentWorkerCount(newCount)

			// Restart workers if processing was active
			if isProcessingActive() {
				newPool.StartWorkers()
			}
		}
	})

	// Render the complete dashboard
	return Html(nil,
		Head(nil,
			Meta(Attrs{"charset": "UTF-8"}),
			Meta(Attrs{
				"name":    "viewport",
				"content": "width=device-width, initial-scale=1.0",
			}),
			Title(nil, Text("Concurrent Dashboard - GoWebComponents")),
			// Include Tailwind CSS for styling
			Link(Attrs{
				"href": "https://cdn.tailwindcss.com",
				"rel":  "stylesheet",
			}),
		),

		Body(Attrs{
			"class": "min-h-screen bg-gray-900",
		},
			// Dashboard header
			DashboardHeader(nil),

			// Main dashboard content
			Main(Attrs{
				"class": "max-w-7xl mx-auto px-4 py-8",
			},
				// Control panel for managing workers
				DashboardControlPanel(Attrs{
					"onStartProcessing":   handleStartProcessing,
					"onStopProcessing":    handleStopProcessing,
					"onWorkerCountChange": handleWorkerCountChange,
					"isProcessingActive":  isProcessingActive(),
					"currentWorkerCount":  currentWorkerCount(),
				}),

				// Real-time metrics display
				DashboardMetricsPanel(Attrs{
					"processingStats": processingStats(),
				}),

				// Live data visualization
				DashboardDataVisualization(Attrs{
					"stockPrices": stockPricesData(),
					"logEntries":  logEntriesData(),
				}),

				// Footer with technology information
				Footer(Attrs{
					"class": "mt-12 text-center text-gray-400",
				},
					P(nil, Text("Built with GoWebComponents - Demonstrating Go's Concurrency in WebAssembly")),
					P(Attrs{
						"class": "text-sm mt-2 text-gray-500",
					}, Text("Features: Goroutines • Channels • Type-Safe Generics • Real-time Processing")),
				),
			),
		),
	)
}

// ConcurrentDashboardExample is the main entry point for the concurrent dashboard demonstration
// It initializes the random number generator and renders the dashboard to the DOM
func ConcurrentDashboardExample() {
	// Initialize random seed for realistic data generation
	rand.Seed(time.Now().UnixNano())

	fmt.Println("🚀 ConcurrentDashboardExample: Starting concurrent dashboard application...")
	fmt.Println("Features demonstrated:")
	fmt.Println("  • Multiple goroutines processing data concurrently")
	fmt.Println("  • Channel-based communication between workers")
	fmt.Println("  • Type-safe state management with GoUseState generics")
	fmt.Println("  • Real-time UI updates from background processing")
	fmt.Println("  • Interactive controls for dynamic goroutine management")
	fmt.Println("  • WebAssembly performance for computational workloads")

	// COMPONENT CREATION SECTION
	// Create a wrapper component function that returns our main dashboard component
	// This follows the React pattern of having a root component
	fmt.Println("🔧 ConcurrentDashboardExample: Creating dashboard page component...")
	dashboardPage := func(props Attrs) *Element {
		// Return our main dashboard component
		return ConcurrentDashboard(nil)
	}

	// DOM CONTAINER DISCOVERY SECTION
	// Find the HTML element where we'll render our Go component
	fmt.Println("🔍 ConcurrentDashboardExample: Searching for DOM container with id 'root'...")
	rootContainer := js.Global().Get("document").Call("getElementById", "root")

	// Error handling for missing container
	if rootContainer.IsUndefined() || rootContainer.IsNull() {
		fmt.Println("❌ ConcurrentDashboardExample: ERROR - No element with id 'root' found in the DOM!")
		fmt.Println("💡 ConcurrentDashboardExample: Make sure your HTML has a <div id='root'></div> element")
		return
	}
	fmt.Println("✅ ConcurrentDashboardExample: DOM container found successfully")

	// RENDERING SECTION
	// Convert our Go component into a virtual DOM element and render it
	fmt.Println("🎨 ConcurrentDashboardExample: Creating element and rendering to container...")

	// CreateElement converts our component function into a virtual DOM element
	// This is similar to React.createElement()
	dashboardElement := CreateElement(dashboardPage, nil)
	fmt.Printf("🧩 ConcurrentDashboardExample: Element created: %+v\n", dashboardElement.Type)

	// Render takes our virtual DOM element and converts it to real DOM
	// It then inserts it into the specified container
	Render(dashboardElement, rootContainer)

	fmt.Println("🎉 ConcurrentDashboardExample: Concurrent Dashboard rendered successfully!")
	fmt.Println("👆 ConcurrentDashboardExample: Ready for user interaction!")
	fmt.Println("You can now:")
	fmt.Println("  1. Click 'Start Processing' to begin concurrent data processing")
	fmt.Println("  2. Adjust the worker count slider to see scaling effects")
	fmt.Println("  3. Watch real-time metrics update as goroutines process data")
	fmt.Println("  4. Observe live data streams showing processed items")
	fmt.Println("  5. Click 'Stop Processing' to gracefully shutdown workers")
}
