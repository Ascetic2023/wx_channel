package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	
	"wx_channel/hub_server/cache"
)

// MetricsSummary 监控指标摘要
type MetricsSummary struct {
	Connections        int                `json:"connections"`
	ConnectionsTrend   float64            `json:"connectionsTrend"`
	APICalls           int                `json:"apiCalls"`
	APICallsTrend      float64            `json:"apiCallsTrend"`
	SuccessRate        float64            `json:"successRate"`
	AvgResponseTime    float64            `json:"avgResponseTime"`
	ResponseTimeTrend  float64            `json:"responseTimeTrend"`
	HeartbeatsSent     int                `json:"heartbeatsSent"`
	HeartbeatsFailed   int                `json:"heartbeatsFailed"`
	CompressionRate    float64            `json:"compressionRate"`
	BytesSaved         int64              `json:"bytesSaved"`
	DetailedMetrics    []DetailedMetric   `json:"detailedMetrics"`
}

// DetailedMetric 详细指标
type DetailedMetric struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// TimeSeriesData 时序数据
type TimeSeriesData struct {
	Connections   TimeSeriesPoints `json:"connections"`
	APICalls      APICallsPoints   `json:"apiCalls"`
	ResponseTime  ResponseTimePoints `json:"responseTime"`
	LoadBalancer  LoadBalancerPoints `json:"loadBalancer"`
}

type TimeSeriesPoints struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

type APICallsPoints struct {
	Labels  []string  `json:"labels"`
	Success []float64 `json:"success"`
	Failed  []float64 `json:"failed"`
}

type ResponseTimePoints struct {
	Labels []string  `json:"labels"`
	P50    []float64 `json:"p50"`
	P95    []float64 `json:"p95"`
	P99    []float64 `json:"p99"`
}

type LoadBalancerPoints struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

// GetMetricsSummary 获取监控指标摘要
func GetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	// 从缓存获取监控指标
	metricsData, err := cache.GetClientMetrics()
	if err != nil {
		http.Error(w, "Failed to fetch metrics", http.StatusInternalServerError)
		return
	}

	// 解析指标（如果没有数据，返回空指标）
	summary := parseMetricsSummary(metricsData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetTimeSeriesData 获取时序数据
func GetTimeSeriesData(w http.ResponseWriter, r *http.Request) {
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "15m"
	}

	// 从 Prometheus 查询时序数据
	data, err := fetchPrometheusTimeSeries(timeRange)
	if err != nil {
		http.Error(w, "Failed to fetch time series data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// parseMetricsSummary 解析指标摘要
func parseMetricsSummary(metricsData string) MetricsSummary {
	// 如果没有数据，返回空指标
	if metricsData == "" {
		return MetricsSummary{
			Connections:       0,
			ConnectionsTrend:  0,
			APICalls:          0,
			APICallsTrend:     0,
			SuccessRate:       0,
			AvgResponseTime:   0,
			ResponseTimeTrend: 0,
			HeartbeatsSent:    0,
			HeartbeatsFailed:  0,
			CompressionRate:   0,
			BytesSaved:        0,
			DetailedMetrics:   []DetailedMetric{},
		}
	}
	
	lines := strings.Split(metricsData, "\n")
	metrics := make(map[string]float64)

	// 解析 Prometheus 格式的指标
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			value, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				metrics[name] = value
			}
		}
	}

	// 计算摘要指标
	connections := int(metrics["wx_channel_ws_connections_total"])
	
	// API 调用统计
	apiCallsSuccess := 0.0
	apiCallsFailed := 0.0
	for key, value := range metrics {
		if strings.Contains(key, "wx_channel_api_calls_total") {
			if strings.Contains(key, "success") {
				apiCallsSuccess += value
			} else {
				apiCallsFailed += value
			}
		}
	}
	totalAPICalls := apiCallsSuccess + apiCallsFailed
	successRate := 0.0
	if totalAPICalls > 0 {
		successRate = (apiCallsSuccess / totalAPICalls) * 100
	}

	// 压缩统计
	bytesIn := int64(metrics["wx_channel_compression_bytes_in_total"])
	bytesOut := int64(metrics["wx_channel_compression_bytes_out_total"])
	compressionRate := 0.0
	if bytesIn > 0 {
		compressionRate = float64(bytesIn-bytesOut) / float64(bytesIn) * 100
	}

	// 详细指标列表
	detailedMetrics := []DetailedMetric{
		{
			Name:        "WebSocket 连接总数",
			Value:       fmt.Sprintf("%d", connections),
			Description: "当前活跃的 WebSocket 连接数量",
		},
		{
			Name:        "API 调用总数",
			Value:       fmt.Sprintf("%.0f", totalAPICalls),
			Description: "所有 API 的累计调用次数",
		},
		{
			Name:        "API 成功率",
			Value:       fmt.Sprintf("%.2f%%", successRate),
			Description: "API 调用成功的百分比",
		},
		{
			Name:        "重连尝试次数",
			Value:       fmt.Sprintf("%.0f", metrics["wx_channel_reconnect_attempts_total"]),
			Description: "客户端尝试重连的总次数",
		},
		{
			Name:        "重连成功次数",
			Value:       fmt.Sprintf("%.0f", metrics["wx_channel_reconnect_success_total"]),
			Description: "客户端成功重连的次数",
		},
		{
			Name:        "心跳发送次数",
			Value:       fmt.Sprintf("%.0f", metrics["wx_channel_heartbeats_sent_total"]),
			Description: "发送的心跳消息总数",
		},
		{
			Name:        "心跳失败次数",
			Value:       fmt.Sprintf("%.0f", metrics["wx_channel_heartbeats_failed_total"]),
			Description: "失败的心跳消息总数",
		},
		{
			Name:        "压缩前数据量",
			Value:       formatBytes(bytesIn),
			Description: "压缩前传输的数据总量",
		},
		{
			Name:        "压缩后数据量",
			Value:       formatBytes(bytesOut),
			Description: "压缩后传输的数据总量",
		},
		{
			Name:        "压缩率",
			Value:       fmt.Sprintf("%.2f%%", compressionRate),
			Description: "数据压缩节省的百分比",
		},
	}

	return MetricsSummary{
		Connections:       connections,
		ConnectionsTrend:  0, // TODO: 计算趋势
		APICalls:          int(totalAPICalls),
		APICallsTrend:     0, // TODO: 计算趋势
		SuccessRate:       successRate,
		AvgResponseTime:   0, // TODO: 从 histogram 计算
		ResponseTimeTrend: 0, // TODO: 计算趋势
		HeartbeatsSent:    int(metrics["wx_channel_heartbeats_sent_total"]),
		HeartbeatsFailed:  int(metrics["wx_channel_heartbeats_failed_total"]),
		CompressionRate:   compressionRate,
		BytesSaved:        bytesIn - bytesOut,
		DetailedMetrics:   detailedMetrics,
	}
}

// fetchPrometheusTimeSeries 从 Prometheus 查询时序数据
func fetchPrometheusTimeSeries(timeRange string) (*TimeSeriesData, error) {
	// 这里需要使用 Prometheus Query API
	// 为了简化，我们生成模拟数据
	
	now := time.Now()
	points := 20
	interval := parseDuration(timeRange) / time.Duration(points)

	labels := make([]string, points)
	connectionsValues := make([]float64, points)
	apiSuccess := make([]float64, points)
	apiFailed := make([]float64, points)
	p50Values := make([]float64, points)
	p95Values := make([]float64, points)
	p99Values := make([]float64, points)

	for i := 0; i < points; i++ {
		t := now.Add(-time.Duration(points-i) * interval)
		labels[i] = t.Format("15:04")
		
		// 模拟数据（实际应该从 Prometheus 查询）
		connectionsValues[i] = float64(1 + i%3)
		apiSuccess[i] = float64(10 + i*2)
		apiFailed[i] = float64(i % 3)
		p50Values[i] = 500 + float64(i*10)
		p95Values[i] = 1000 + float64(i*20)
		p99Values[i] = 2000 + float64(i*30)
	}

	return &TimeSeriesData{
		Connections: TimeSeriesPoints{
			Labels: labels,
			Values: connectionsValues,
		},
		APICalls: APICallsPoints{
			Labels:  labels,
			Success: apiSuccess,
			Failed:  apiFailed,
		},
		ResponseTime: ResponseTimePoints{
			Labels: labels,
			P50:    p50Values,
			P95:    p95Values,
			P99:    p99Values,
		},
		LoadBalancer: LoadBalancerPoints{
			Labels: []string{"Client A", "Client B", "Client C"},
			Values: []float64{100, 85, 90},
		},
	}, nil
}

// parseDuration 解析时间范围
func parseDuration(timeRange string) time.Duration {
	switch timeRange {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	default:
		return 15 * time.Minute
	}
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
