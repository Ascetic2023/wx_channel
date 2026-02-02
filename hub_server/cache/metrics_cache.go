package cache

import (
	"sync"
	"time"
)

// metricsCache 缓存客户端推送的监控数据
var (
	metricsCache      = make(map[string]string) // clientID -> metrics
	metricsCacheMutex sync.RWMutex
	metricsCacheTTL   = 60 * time.Second // 缓存 60 秒
	metricsTimestamp  = make(map[string]time.Time)
)

// UpdateClientMetrics 更新客户端的监控数据
func UpdateClientMetrics(clientID string, metricsData string) {
	metricsCacheMutex.Lock()
	defer metricsCacheMutex.Unlock()
	
	metricsCache[clientID] = metricsData
	metricsTimestamp[clientID] = time.Now()
}

// GetClientMetrics 获取客户端的监控数据
func GetClientMetrics() (string, error) {
	metricsCacheMutex.RLock()
	defer metricsCacheMutex.RUnlock()
	
	// 清理过期数据
	now := time.Now()
	for clientID, timestamp := range metricsTimestamp {
		if now.Sub(timestamp) > metricsCacheTTL {
			delete(metricsCache, clientID)
			delete(metricsTimestamp, clientID)
		}
	}
	
	// 如果没有任何客户端数据，返回空字符串（不是错误）
	if len(metricsCache) == 0 {
		return "", nil
	}
	
	// 合并所有客户端的指标（简单起见，使用第一个客户端的数据）
	// 实际应该聚合所有客户端的数据
	for _, metrics := range metricsCache {
		return metrics, nil
	}
	
	return "", nil
}

// GetAllClientMetrics 获取所有客户端的监控数据
func GetAllClientMetrics() map[string]string {
	metricsCacheMutex.RLock()
	defer metricsCacheMutex.RUnlock()
	
	// 清理过期数据
	now := time.Now()
	for clientID, timestamp := range metricsTimestamp {
		if now.Sub(timestamp) > metricsCacheTTL {
			delete(metricsCache, clientID)
			delete(metricsTimestamp, clientID)
		}
	}
	
	// 返回所有客户端的数据副本
	result := make(map[string]string)
	for clientID, metrics := range metricsCache {
		result[clientID] = metrics
	}
	
	return result
}
