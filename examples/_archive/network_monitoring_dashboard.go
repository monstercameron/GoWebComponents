// ./examples/network_monitoring_dashboard.go
// This file demonstrates a VMware-style Network Monitoring Dashboard using GoWebComponents.
// It showcases real-time system metrics visualization with professional UI design and interactive controls.
//
// Key Features Demonstrated:
// - Real-time system metrics with simulated data and jitter
// - Chart.js integration for professional data visualization
// - Toggle controls for enabling/disabling monitoring
// - VMware-style professional dashboard design
// - Tailwind CSS dark mode styling
// - Multiple metric categories (CPU, GPU, RAM, Network, etc.)

//go:build js && wasm
// +build js,wasm

package example

import (
	"fmt"
	"math"
	"math/rand"
	"syscall/js"
	"time"

	// Dot import allows us to use fiber package functions directly
	. "github.com/monstercameron/GoWebComponents/fiber"
)

// SystemMetrics represents a comprehensive collection of system performance metrics
type SystemMetrics struct {
	// Unique identifier for change detection
	ID string `json:"id"` // Unique ID for this metrics snapshot

	// Core system metrics
	CPUUsage    float64 `json:"cpuUsage"`    // CPU usage percentage (0-100)
	GPUUsage    float64 `json:"gpuUsage"`    // GPU usage percentage (0-100)
	RAMUsage    float64 `json:"ramUsage"`    // RAM usage percentage (0-100)
	MemoryUsage float64 `json:"memoryUsage"` // Memory usage in GB
	NetworkRx   float64 `json:"networkRx"`   // Network receive throughput in Mbps
	NetworkTx   float64 `json:"networkTx"`   // Network transmit throughput in Mbps

	// Additional metrics (5 more as requested)
	DiskIORead       float64 `json:"diskIORead"`       // Disk I/O read speed in MB/s
	DiskIOWrite      float64 `json:"diskIOWrite"`      // Disk I/O write speed in MB/s
	PowerConsumption float64 `json:"powerConsumption"` // Power consumption in watts
	Temperature      float64 `json:"temperature"`      // System temperature in Celsius
	ActiveProcesses  int     `json:"activeProcesses"`  // Number of active processes

	// VM Hardware specs
	VMCores     int `json:"vmCores"`     // Number of virtual CPU cores
	VMMemoryGB  int `json:"vmMemoryGB"`  // VM allocated memory in GB
	VMStorageGB int `json:"vmStorageGB"` // VM storage capacity in GB

	Timestamp int64 `json:"timestamp"` // Unix timestamp when metrics were collected
}

// DashboardState manages the state of the monitoring dashboard
type DashboardState struct {
	IsMonitoring   bool            `json:"isMonitoring"`   // Whether monitoring is active
	MetricsHistory []SystemMetrics `json:"metricsHistory"` // Historical metrics data
	CurrentMetrics SystemMetrics   `json:"currentMetrics"` // Current system metrics
	UpdateInterval int             `json:"updateInterval"` // Update interval in milliseconds
}

// BaselineMetrics holds baseline values for realistic metric simulation
var BaselineMetrics = SystemMetrics{
	CPUUsage:         45.0,
	GPUUsage:         30.0,
	RAMUsage:         65.0,
	MemoryUsage:      8.5,
	NetworkRx:        125.0,
	NetworkTx:        85.0,
	DiskIORead:       45.0,
	DiskIOWrite:      25.0,
	PowerConsumption: 150.0,
	Temperature:      42.0,
	ActiveProcesses:  285,
	VMCores:          8,
	VMMemoryGB:       16,
	VMStorageGB:      500,
}

// NetworkDashboardHeader creates the main header for the VMware-style dashboard
func NetworkDashboardHeader(props Attrs) *Element {
	return Header(Attrs{
		"class": "bg-gray-900 text-gray-100 p-6 shadow-2xl border-b border-gray-700",
	},
		Div(Attrs{
			"class": "max-w-7xl mx-auto",
		},
			// Main title with VMware-style branding
			Div(Attrs{
				"class": "flex items-center justify-between",
			},
				Div(Attrs{
					"class": "flex items-center",
				},
					Div(Attrs{
						"class": "w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center mr-4",
					},
						Span(Attrs{
							"class": "text-white text-xl font-bold",
						}, Text("VM")),
					),
					Div(Attrs{},
						H1(Attrs{
							"class": "text-3xl font-bold text-white",
						}, Text("vSphere Monitoring Center")),
						P(Attrs{
							"class": "text-gray-400 text-sm mt-1",
						}, Text("Real-time Infrastructure Performance Dashboard")),
					),
				),
				// Status indicator
				Div(Attrs{
					"class": "flex items-center space-x-2",
				},
					Div(Attrs{
						"class": "w-3 h-3 bg-green-500 rounded-full animate-pulse",
					}),
					Span(Attrs{
						"class": "text-green-400 text-sm font-medium",
					}, Text("Online")),
				),
			),
		),
	)
}

// NetworkDashboardControlPanel creates the control panel with toggle and settings
func NetworkDashboardControlPanel(props Attrs) *Element {
	var onToggleMonitoring interface{}
	var isMonitoring bool
	var updateInterval int

	if props != nil {
		if handler, exists := props["onToggleMonitoring"]; exists {
			onToggleMonitoring = handler
		}
		if monitoring, exists := props["isMonitoring"]; exists {
			isMonitoring = monitoring.(bool)
		}
		if interval, exists := props["updateInterval"]; exists {
			updateInterval = interval.(int)
		}
	}

	return Section(Attrs{
		"class": "bg-gray-800 rounded-lg shadow-xl p-6 mb-6 border border-gray-700",
	},
		Div(Attrs{
			"class": "flex items-center justify-between",
		},
			// Control title
			H2(Attrs{
				"class": "text-xl font-semibold text-gray-100 flex items-center",
			},
				Span(Attrs{
					"class": "text-blue-400 mr-2",
				}, Text("⚡")),
				Text("Monitoring Controls"),
			),

			// Controls section
			Div(Attrs{
				"class": "flex items-center space-x-6",
			},
				// Monitoring toggle
				Div(Attrs{
					"class": "flex items-center space-x-3",
				},
					Label(Attrs{
						"class": "text-gray-300 font-medium",
					}, Text("Real-time Monitoring")),
					Button(Attrs{
						"class": func() string {
							baseClass := "relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-800"
							if isMonitoring {
								return baseClass + " bg-blue-600"
							}
							return baseClass + " bg-gray-600"
						}(),
						"onclick": onToggleMonitoring,
					},
						Span(Attrs{
							"class": func() string {
								baseClass := "inline-block h-4 w-4 transform rounded-full bg-white transition duration-200 ease-in-out"
								if isMonitoring {
									return baseClass + " translate-x-6"
								}
								return baseClass + " translate-x-1"
							}(),
						}),
					),
				),

				// Status indicator
				Div(Attrs{
					"class": "flex items-center space-x-2",
				},
					Div(Attrs{
						"class": func() string {
							if isMonitoring {
								return "w-2 h-2 bg-green-500 rounded-full animate-pulse"
							}
							return "w-2 h-2 bg-gray-500 rounded-full"
						}(),
					}),
					Span(Attrs{
						"class": func() string {
							if isMonitoring {
								return "text-green-400 text-sm font-medium"
							}
							return "text-gray-500 text-sm font-medium"
						}(),
					}, Text(func() string {
						if isMonitoring {
							return "Active"
						}
						return "Inactive"
					}())),
				),

				// Update interval display
				Div(Attrs{
					"class": "text-gray-400 text-sm",
				},
					Text(fmt.Sprintf("Update: %dms", updateInterval)),
				),
			),
		),
	)
}

// MetricCard creates a single metric display card
func MetricCard(props Attrs) *Element {
	var title, value, unit, icon string
	var percentage float64
	var trend string

	if props != nil {
		if t, exists := props["title"]; exists {
			title = t.(string)
		}
		if v, exists := props["value"]; exists {
			value = v.(string)
		}
		if u, exists := props["unit"]; exists {
			unit = u.(string)
		}
		if i, exists := props["icon"]; exists {
			icon = i.(string)
		}
		if p, exists := props["percentage"]; exists {
			percentage = p.(float64)
		}
		if tr, exists := props["trend"]; exists {
			trend = tr.(string)
		}
	}

	// Determine color based on percentage
	var colorClass string
	if percentage < 50 {
		colorClass = "text-green-400"
	} else if percentage < 80 {
		colorClass = "text-yellow-400"
	} else {
		colorClass = "text-red-400"
	}

	return Div(Attrs{
		"class": "bg-gray-800 rounded-lg p-4 border border-gray-700 shadow-lg hover:shadow-xl transition-shadow duration-200",
	},
		// Card header
		Div(Attrs{
			"class": "flex items-center justify-between mb-3",
		},
			Div(Attrs{
				"class": "flex items-center",
			},
				Span(Attrs{
					"class": "text-2xl mr-2",
				}, Text(icon)),
				H3(Attrs{
					"class": "text-gray-200 font-medium text-sm",
				}, Text(title)),
			),
			// Trend indicator
			Span(Attrs{
				"class": func() string {
					if trend == "up" {
						return "text-green-400 text-xs"
					} else if trend == "down" {
						return "text-red-400 text-xs"
					}
					return "text-gray-400 text-xs"
				}(),
			}, Text(func() string {
				if trend == "up" {
					return "↗"
				} else if trend == "down" {
					return "↘"
				}
				return "→"
			}())),
		),

		// Metric value
		Div(Attrs{
			"class": "mb-3",
		},
			Span(Attrs{
				"class": colorClass + " text-2xl font-bold",
			}, Text(value)),
			Span(Attrs{
				"class": "text-gray-400 text-sm ml-1",
			}, Text(unit)),
		),

		// Progress bar
		Div(Attrs{
			"class": "w-full bg-gray-700 rounded-full h-2",
		},
			Div(Attrs{
				"class": func() string {
					baseClass := "h-2 rounded-full transition-all duration-300"
					if percentage < 50 {
						return baseClass + " bg-green-500"
					} else if percentage < 80 {
						return baseClass + " bg-yellow-500"
					}
					return baseClass + " bg-red-500"
				}(),
				"style": fmt.Sprintf("width: %.1f%%", percentage),
			}),
		),
	)
}

// SystemOverviewPanel creates the main metrics overview panel
func SystemOverviewPanel(props Attrs) *Element {
	var metrics SystemMetrics

	if props != nil {
		if m, exists := props["metrics"]; exists {
			metrics = m.(SystemMetrics)
		}
	}

	return Section(Attrs{
		"class": "mb-6",
	},
		H2(Attrs{
			"class": "text-xl font-semibold text-gray-100 mb-4 flex items-center",
		},
			Span(Attrs{
				"class": "text-blue-400 mr-2",
			}, Text("📊")),
			Text("System Overview"),
		),

		// Metrics grid
		Div(Attrs{
			"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4",
		},
			// CPU Usage
			MetricCard(Attrs{
				"title":      "CPU Usage",
				"value":      fmt.Sprintf("%.1f", metrics.CPUUsage),
				"unit":       "%",
				"icon":       "🔥",
				"percentage": metrics.CPUUsage,
				"trend":      "up",
			}),

			// GPU Usage
			MetricCard(Attrs{
				"title":      "GPU Usage",
				"value":      fmt.Sprintf("%.1f", metrics.GPUUsage),
				"unit":       "%",
				"icon":       "🎮",
				"percentage": metrics.GPUUsage,
				"trend":      "down",
			}),

			// RAM Usage
			MetricCard(Attrs{
				"title":      "RAM Usage",
				"value":      fmt.Sprintf("%.1f", metrics.RAMUsage),
				"unit":       "%",
				"icon":       "💾",
				"percentage": metrics.RAMUsage,
				"trend":      "stable",
			}),

			// Memory Usage
			MetricCard(Attrs{
				"title":      "Memory Used",
				"value":      fmt.Sprintf("%.1f", metrics.MemoryUsage),
				"unit":       "GB",
				"icon":       "🧠",
				"percentage": (metrics.MemoryUsage / 16.0) * 100, // Assuming 16GB total
				"trend":      "up",
			}),

			// Network RX
			MetricCard(Attrs{
				"title":      "Network RX",
				"value":      fmt.Sprintf("%.1f", metrics.NetworkRx),
				"unit":       "Mbps",
				"icon":       "📥",
				"percentage": (metrics.NetworkRx / 1000.0) * 100, // Assuming 1Gbps max
				"trend":      "up",
			}),

			// Network TX
			MetricCard(Attrs{
				"title":      "Network TX",
				"value":      fmt.Sprintf("%.1f", metrics.NetworkTx),
				"unit":       "Mbps",
				"icon":       "📤",
				"percentage": (metrics.NetworkTx / 1000.0) * 100, // Assuming 1Gbps max
				"trend":      "down",
			}),

			// Disk I/O Read
			MetricCard(Attrs{
				"title":      "Disk Read",
				"value":      fmt.Sprintf("%.1f", metrics.DiskIORead),
				"unit":       "MB/s",
				"icon":       "💿",
				"percentage": (metrics.DiskIORead / 200.0) * 100, // Assuming 200MB/s max
				"trend":      "stable",
			}),

			// Disk I/O Write
			MetricCard(Attrs{
				"title":      "Disk Write",
				"value":      fmt.Sprintf("%.1f", metrics.DiskIOWrite),
				"unit":       "MB/s",
				"icon":       "💽",
				"percentage": (metrics.DiskIOWrite / 200.0) * 100, // Assuming 200MB/s max
				"trend":      "up",
			}),

			// Power Consumption
			MetricCard(Attrs{
				"title":      "Power Usage",
				"value":      fmt.Sprintf("%.0f", metrics.PowerConsumption),
				"unit":       "W",
				"icon":       "⚡",
				"percentage": (metrics.PowerConsumption / 300.0) * 100, // Assuming 300W max
				"trend":      "stable",
			}),

			// Temperature
			MetricCard(Attrs{
				"title":      "Temperature",
				"value":      fmt.Sprintf("%.1f", metrics.Temperature),
				"unit":       "°C",
				"icon":       "🌡️",
				"percentage": (metrics.Temperature / 100.0) * 100, // Assuming 100°C max
				"trend":      "down",
			}),

			// Active Processes
			MetricCard(Attrs{
				"title":      "Processes",
				"value":      fmt.Sprintf("%d", metrics.ActiveProcesses),
				"unit":       "",
				"icon":       "⚙️",
				"percentage": float64(metrics.ActiveProcesses) / 500.0 * 100, // Assuming 500 max
				"trend":      "up",
			}),
		),
	)
}

// VMHardwarePanel creates the VM hardware specifications panel
func VMHardwarePanel(props Attrs) *Element {
	var metrics SystemMetrics

	if props != nil {
		if m, exists := props["metrics"]; exists {
			metrics = m.(SystemMetrics)
		}
	}

	return Section(Attrs{
		"class": "bg-gray-800 rounded-lg shadow-xl p-6 mb-6 border border-gray-700",
	},
		H2(Attrs{
			"class": "text-xl font-semibold text-gray-100 mb-4 flex items-center",
		},
			Span(Attrs{
				"class": "text-purple-400 mr-2",
			}, Text("🖥️")),
			Text("Virtual Machine Specifications"),
		),

		Div(Attrs{
			"class": "grid grid-cols-1 md:grid-cols-3 gap-6",
		},
			// CPU Cores
			Div(Attrs{
				"class": "text-center",
			},
				Div(Attrs{
					"class": "w-16 h-16 bg-purple-600 rounded-full flex items-center justify-center mx-auto mb-3",
				},
					Span(Attrs{
						"class": "text-white text-2xl",
					}, Text("🔧")),
				),
				H3(Attrs{
					"class": "text-gray-200 font-medium mb-1",
				}, Text("CPU Cores")),
				P(Attrs{
					"class": "text-purple-400 text-2xl font-bold",
				}, Text(fmt.Sprintf("%d", metrics.VMCores))),
				P(Attrs{
					"class": "text-gray-400 text-sm",
				}, Text("vCPUs Allocated")),
			),

			// Memory
			Div(Attrs{
				"class": "text-center",
			},
				Div(Attrs{
					"class": "w-16 h-16 bg-purple-600 rounded-full flex items-center justify-center mx-auto mb-3",
				},
					Span(Attrs{
						"class": "text-white text-2xl",
					}, Text("💾")),
				),
				H3(Attrs{
					"class": "text-gray-200 font-medium mb-1",
				}, Text("Memory")),
				P(Attrs{
					"class": "text-purple-400 text-2xl font-bold",
				}, Text(fmt.Sprintf("%d GB", metrics.VMMemoryGB))),
				P(Attrs{
					"class": "text-gray-400 text-sm",
				}, Text("RAM Allocated")),
			),

			// Storage
			Div(Attrs{
				"class": "text-center",
			},
				Div(Attrs{
					"class": "w-16 h-16 bg-purple-600 rounded-full flex items-center justify-center mx-auto mb-3",
				},
					Span(Attrs{
						"class": "text-white text-2xl",
					}, Text("💿")),
				),
				H3(Attrs{
					"class": "text-gray-200 font-medium mb-1",
				}, Text("Storage")),
				P(Attrs{
					"class": "text-purple-400 text-2xl font-bold",
				}, Text(fmt.Sprintf("%d GB", metrics.VMStorageGB))),
				P(Attrs{
					"class": "text-gray-400 text-sm",
				}, Text("Disk Capacity")),
			),
		),
	)
}

// ChartPanel creates a panel with Chart.js integration for historical data
func ChartPanel(props Attrs) *Element {
	var chartId string
	var title string

	if props != nil {
		if id, exists := props["chartId"]; exists {
			chartId = id.(string)
		}
		if t, exists := props["title"]; exists {
			title = t.(string)
		}
	}

	return Section(Attrs{
		"class": "bg-gray-800 rounded-lg shadow-xl p-6 mb-6 border border-gray-700",
	},
		H2(Attrs{
			"class": "text-xl font-semibold text-gray-100 mb-4 flex items-center",
		},
			Span(Attrs{
				"class": "text-green-400 mr-2",
			}, Text("📈")),
			Text(title),
		),

		// Chart container
		Div(Attrs{
			"class": "relative h-64 w-full",
		},
			Canvas(Attrs{
				"id":     chartId,
				"class":  "w-full h-full",
				"width":  "800",
				"height": "400",
			}),
		),
	)
}

// generateRandomMetrics creates realistic system metrics with jitter
func generateRandomMetrics() SystemMetrics {
	now := time.Now().UnixMilli()

	// Generate jitter (±10% variance from baseline)
	jitter := func(baseline float64) float64 {
		variance := baseline * 0.1
		return baseline + (rand.Float64()-0.5)*2*variance
	}

	// Generate trending jitter (includes time-based patterns)
	trendingJitter := func(baseline float64, timeOffset float64) float64 {
		// Add sine wave pattern for realistic fluctuation
		sineComponent := math.Sin(timeOffset/10000) * baseline * 0.05
		randomComponent := (rand.Float64() - 0.5) * baseline * 0.1
		return baseline + sineComponent + randomComponent
	}

	timeOffset := float64(now % 100000)

	return SystemMetrics{
		ID:               fmt.Sprintf("metrics-%d-%d", now, rand.Intn(10000)),
		CPUUsage:         math.Max(0, math.Min(100, trendingJitter(BaselineMetrics.CPUUsage, timeOffset))),
		GPUUsage:         math.Max(0, math.Min(100, trendingJitter(BaselineMetrics.GPUUsage, timeOffset*1.3))),
		RAMUsage:         math.Max(0, math.Min(100, jitter(BaselineMetrics.RAMUsage))),
		MemoryUsage:      math.Max(0, math.Min(16, jitter(BaselineMetrics.MemoryUsage))),
		NetworkRx:        math.Max(0, trendingJitter(BaselineMetrics.NetworkRx, timeOffset*0.8)),
		NetworkTx:        math.Max(0, trendingJitter(BaselineMetrics.NetworkTx, timeOffset*0.7)),
		DiskIORead:       math.Max(0, trendingJitter(BaselineMetrics.DiskIORead, timeOffset*1.1)),
		DiskIOWrite:      math.Max(0, trendingJitter(BaselineMetrics.DiskIOWrite, timeOffset*1.2)),
		PowerConsumption: math.Max(50, math.Min(300, jitter(BaselineMetrics.PowerConsumption))),
		Temperature:      math.Max(20, math.Min(85, jitter(BaselineMetrics.Temperature))),
		ActiveProcesses:  int(math.Max(100, math.Min(500, jitter(float64(BaselineMetrics.ActiveProcesses))))),
		VMCores:          BaselineMetrics.VMCores,
		VMMemoryGB:       BaselineMetrics.VMMemoryGB,
		VMStorageGB:      BaselineMetrics.VMStorageGB,
		Timestamp:        now,
	}
}

// NetworkMonitoringDashboard creates the main dashboard component
func NetworkMonitoringDashboard(props Attrs) *Element {
	// Initialize dashboard state
	dashboardState, setDashboardState := GoUseState(DashboardState{
		IsMonitoring:   false,
		MetricsHistory: make([]SystemMetrics, 0),
		CurrentMetrics: generateRandomMetrics(),
		UpdateInterval: 1000,
	})

	state := dashboardState()

	// Toggle monitoring handler
	handleToggleMonitoring := func(this js.Value, args []js.Value) interface{} {
		currentState := dashboardState()
		newState := DashboardState{
			IsMonitoring:   !currentState.IsMonitoring,
			MetricsHistory: currentState.MetricsHistory,
			CurrentMetrics: currentState.CurrentMetrics,
			UpdateInterval: currentState.UpdateInterval,
		}
		fmt.Printf("🎛️ Toggling monitoring: %v -> %v\n", currentState.IsMonitoring, newState.IsMonitoring)
		setDashboardState(newState)
		return nil
	}

	// Add an update counter to force state change detection
	updateCounter, setUpdateCounter := GoUseState[int](0)

	// Create a cancellation channel for the current monitoring session
	cancelChan, setCancelChan := GoUseState[chan bool](make(chan bool, 1))

	// Update metrics periodically when monitoring is active
	GoUseEffect(func() {
		if state.IsMonitoring {
			fmt.Printf("🔄 Starting metrics update loop with %dms interval\n", state.UpdateInterval)

			// Create a new cancellation channel for this session
			newCancelChan := make(chan bool, 1)
			setCancelChan(newCancelChan)

			ticker := time.NewTicker(time.Duration(state.UpdateInterval) * time.Millisecond)

			go func() {
				defer func() {
					ticker.Stop()
					fmt.Println("🧹 Cleaned up ticker")
				}()

				for {
					select {
					case <-newCancelChan:
						fmt.Println("🛑 Stopping metrics generation - cancelled via channel")
						return
					case <-ticker.C:
						// Double-check state is still active
						currentState := dashboardState()
						if !currentState.IsMonitoring {
							fmt.Println("🛑 Stopping metrics generation - monitoring disabled")
							return
						}

						// Increment update counter to force state change detection
						currentCounter := updateCounter()
						newCounter := currentCounter + 1
						setUpdateCounter(newCounter)

						fmt.Printf("📊 Generated metrics #%d\n", newCounter)

						// Generate new metrics
						newMetrics := generateRandomMetrics()

						fmt.Printf("🎨 Updating UI with new metrics - CPU: %.1f%%, GPU: %.1f%%, RAM: %.1f%%\n",
							newMetrics.CPUUsage, newMetrics.GPUUsage, newMetrics.RAMUsage)

						// Create new history slice (don't modify existing)
						newHistory := make([]SystemMetrics, len(currentState.MetricsHistory))
						copy(newHistory, currentState.MetricsHistory)
						newHistory = append(newHistory, newMetrics)
						if len(newHistory) > 50 {
							newHistory = newHistory[1:]
						}

						// Create completely new state object with unique timestamp
						newState := DashboardState{
							IsMonitoring:   currentState.IsMonitoring,
							MetricsHistory: newHistory,
							CurrentMetrics: newMetrics,
							UpdateInterval: currentState.UpdateInterval,
						}

						fmt.Printf("📈 Setting new state #%d\n", newCounter)
						setDashboardState(newState)
					}
				}
			}()
		} else {
			// When monitoring is turned off, signal the goroutine to stop
			fmt.Println("🛑 Monitoring turned off - sending cancellation signal")
			currentCancelChan := cancelChan()
			if currentCancelChan != nil {
				select {
				case currentCancelChan <- true:
					fmt.Println("✅ Cancellation signal sent successfully")
				default:
					fmt.Println("⚠️ Cancellation channel was full or closed")
				}
			}
		}
	}, []interface{}{state.IsMonitoring})

	return Div(Attrs{
		"class": "min-h-screen bg-gray-900 text-gray-100",
	},
		// Dashboard header
		NetworkDashboardHeader(Attrs{}),

		// Main content
		Main(Attrs{
			"class": "max-w-7xl mx-auto p-6",
		},
			// Control panel
			NetworkDashboardControlPanel(Attrs{
				"onToggleMonitoring": js.FuncOf(handleToggleMonitoring),
				"isMonitoring":       state.IsMonitoring,
				"updateInterval":     state.UpdateInterval,
			}),

			// VM Hardware specs
			VMHardwarePanel(Attrs{
				"metrics": state.CurrentMetrics,
			}),

			// System overview
			SystemOverviewPanel(Attrs{
				"metrics": state.CurrentMetrics,
			}),

			// Charts section
			Div(Attrs{
				"class": "grid grid-cols-1 lg:grid-cols-2 gap-6",
			},
				ChartPanel(Attrs{
					"chartId": "cpuChart",
					"title":   "CPU & GPU Usage Trends",
				}),
				ChartPanel(Attrs{
					"chartId": "networkChart",
					"title":   "Network Throughput",
				}),
				ChartPanel(Attrs{
					"chartId": "memoryChart",
					"title":   "Memory & Storage I/O",
				}),
				ChartPanel(Attrs{
					"chartId": "systemChart",
					"title":   "System Health Overview",
				}),
			),
		),

		// Chart.js initialization script
		Script(Attrs{
			"src": "https://cdn.jsdelivr.net/npm/chart.js",
		}),

		// Custom Chart.js initialization
		Script(Attrs{},
			Text(`
				// Initialize Chart.js charts when the script loads
				document.addEventListener('DOMContentLoaded', function() {
					initializeCharts();
				});

				function initializeCharts() {
					// Chart.js configuration for dark theme
					const darkTheme = {
						color: '#e5e7eb',
						backgroundColor: 'rgba(75, 85, 99, 0.2)',
						borderColor: 'rgba(156, 163, 175, 0.5)',
						grid: {
							color: 'rgba(75, 85, 99, 0.3)'
						}
					};

					// CPU & GPU Chart
					const cpuCtx = document.getElementById('cpuChart');
					if (cpuCtx) {
						new Chart(cpuCtx, {
							type: 'line',
							data: {
								labels: [],
								datasets: [{
									label: 'CPU Usage (%)',
									data: [],
									borderColor: 'rgb(239, 68, 68)',
									backgroundColor: 'rgba(239, 68, 68, 0.1)',
									tension: 0.4
								}, {
									label: 'GPU Usage (%)',
									data: [],
									borderColor: 'rgb(34, 197, 94)',
									backgroundColor: 'rgba(34, 197, 94, 0.1)',
									tension: 0.4
								}]
							},
							options: {
								responsive: true,
								maintainAspectRatio: false,
								plugins: {
									legend: {
										labels: {
											color: '#e5e7eb'
										}
									}
								},
								scales: {
									x: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' }
									},
									y: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' },
										min: 0,
										max: 100
									}
								}
							}
						});
					}

					// Network Chart
					const networkCtx = document.getElementById('networkChart');
					if (networkCtx) {
						new Chart(networkCtx, {
							type: 'line',
							data: {
								labels: [],
								datasets: [{
									label: 'Network RX (Mbps)',
									data: [],
									borderColor: 'rgb(59, 130, 246)',
									backgroundColor: 'rgba(59, 130, 246, 0.1)',
									tension: 0.4
								}, {
									label: 'Network TX (Mbps)',
									data: [],
									borderColor: 'rgb(168, 85, 247)',
									backgroundColor: 'rgba(168, 85, 247, 0.1)',
									tension: 0.4
								}]
							},
							options: {
								responsive: true,
								maintainAspectRatio: false,
								plugins: {
									legend: {
										labels: {
											color: '#e5e7eb'
										}
									}
								},
								scales: {
									x: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' }
									},
									y: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' },
										min: 0
									}
								}
							}
						});
					}

					// Memory Chart
					const memoryCtx = document.getElementById('memoryChart');
					if (memoryCtx) {
						new Chart(memoryCtx, {
							type: 'line',
							data: {
								labels: [],
								datasets: [{
									label: 'RAM Usage (%)',
									data: [],
									borderColor: 'rgb(245, 158, 11)',
									backgroundColor: 'rgba(245, 158, 11, 0.1)',
									tension: 0.4
								}, {
									label: 'Disk I/O Read (MB/s)',
									data: [],
									borderColor: 'rgb(20, 184, 166)',
									backgroundColor: 'rgba(20, 184, 166, 0.1)',
									tension: 0.4
								}]
							},
							options: {
								responsive: true,
								maintainAspectRatio: false,
								plugins: {
									legend: {
										labels: {
											color: '#e5e7eb'
										}
									}
								},
								scales: {
									x: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' }
									},
									y: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' },
										min: 0
									}
								}
							}
						});
					}

					// System Health Chart
					const systemCtx = document.getElementById('systemChart');
					if (systemCtx) {
						new Chart(systemCtx, {
							type: 'line',
							data: {
								labels: [],
								datasets: [{
									label: 'Temperature (°C)',
									data: [],
									borderColor: 'rgb(251, 113, 133)',
									backgroundColor: 'rgba(251, 113, 133, 0.1)',
									tension: 0.4
								}, {
									label: 'Power (W)',
									data: [],
									borderColor: 'rgb(132, 204, 22)',
									backgroundColor: 'rgba(132, 204, 22, 0.1)',
									tension: 0.4
								}]
							},
							options: {
								responsive: true,
								maintainAspectRatio: false,
								plugins: {
									legend: {
										labels: {
											color: '#e5e7eb'
										}
									}
								},
								scales: {
									x: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' }
									},
									y: {
										ticks: { color: '#9ca3af' },
										grid: { color: 'rgba(75, 85, 99, 0.3)' },
										min: 0
									}
								}
							}
						});
					}
				}
			`),
		),
	)
}

// NetworkMonitoringDashboardExample initializes and starts the dashboard
func NetworkMonitoringDashboardExample() {
	fmt.Println("Starting VMware-style Network Monitoring Dashboard Example...")

	// Create a wrapper component function that returns our main dashboard component
	// This follows the React pattern of having a root component
	dashboardPage := func(props Attrs) *Element {
		// Return our main dashboard component
		return NetworkMonitoringDashboard(nil)
	}

	// Find the HTML element where we'll render our Go component
	rootContainer := js.Global().Get("document").Call("getElementById", "app")

	// Error handling for missing container
	if rootContainer.IsUndefined() || rootContainer.IsNull() {
		fmt.Println("❌ ERROR - No element with id 'app' found in the DOM!")
		fmt.Println("💡 Make sure your HTML has a <div id='app'></div> element")
		return
	}

	// Create element and render to DOM
	dashboardElement := CreateElement(dashboardPage, nil)
	Render(dashboardElement, rootContainer)

	fmt.Println("VMware-style Network Monitoring Dashboard is now running!")
	fmt.Println("Features enabled:")
	fmt.Println("- Real-time system metrics with jitter simulation")
	fmt.Println("- Toggle control for monitoring on/off")
	fmt.Println("- Chart.js integration for data visualization")
	fmt.Println("- VMware-style professional UI with Tailwind dark mode")
	fmt.Println("- 11 different metrics including CPU, GPU, RAM, Network, and more")
	fmt.Println("- VM hardware specifications display")
}
