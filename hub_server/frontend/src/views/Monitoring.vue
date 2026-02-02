<template>
  <div class="monitoring-container">
    <header class="header">
      <h1 class="title">系统监控</h1>
      <div class="header-actions">
        <button @click="refreshData" class="btn-refresh" :disabled="loading">
          <span class="icon">🔄</span>
          {{ loading ? '刷新中...' : '刷新数据' }}
        </button>
        <select v-model="timeRange" @change="refreshData" class="time-select">
          <option value="5m">最近 5 分钟</option>
          <option value="15m">最近 15 分钟</option>
          <option value="1h">最近 1 小时</option>
          <option value="6h">最近 6 小时</option>
          <option value="24h">最近 24 小时</option>
        </select>
      </div>
    </header>

    <!-- 关键指标卡片 -->
    <div class="metrics-grid">
      <div class="metric-card">
        <div class="metric-icon">🔌</div>
        <div class="metric-content">
          <div class="metric-label">WebSocket 连接</div>
          <div class="metric-value">{{ metrics.connections }}</div>
          <div class="metric-trend" :class="getTrendClass(metrics.connectionsTrend)">
            {{ formatTrend(metrics.connectionsTrend) }}
          </div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon">📡</div>
        <div class="metric-content">
          <div class="metric-label">API 调用总数</div>
          <div class="metric-value">{{ formatNumber(metrics.apiCalls) }}</div>
          <div class="metric-trend" :class="getTrendClass(metrics.apiCallsTrend)">
            {{ formatTrend(metrics.apiCallsTrend) }}
          </div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon">✅</div>
        <div class="metric-content">
          <div class="metric-label">API 成功率</div>
          <div class="metric-value">{{ metrics.successRate }}%</div>
          <div class="metric-status" :class="getStatusClass(metrics.successRate)">
            {{ getStatusText(metrics.successRate) }}
          </div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon">⚡</div>
        <div class="metric-content">
          <div class="metric-label">平均响应时间</div>
          <div class="metric-value">{{ metrics.avgResponseTime }}ms</div>
          <div class="metric-trend" :class="getTrendClass(-metrics.responseTimeTrend)">
            {{ formatTrend(metrics.responseTimeTrend) }}
          </div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon">💓</div>
        <div class="metric-content">
          <div class="metric-label">心跳状态</div>
          <div class="metric-value">{{ metrics.heartbeatsSent }}</div>
          <div class="metric-status success">
            失败: {{ metrics.heartbeatsFailed }}
          </div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon">📦</div>
        <div class="metric-content">
          <div class="metric-label">压缩率</div>
          <div class="metric-value">{{ metrics.compressionRate }}%</div>
          <div class="metric-status success">
            节省 {{ formatBytes(metrics.bytesSaved) }}
          </div>
        </div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-section">
      <!-- 连接数趋势 -->
      <div class="chart-card">
        <h3 class="chart-title">WebSocket 连接数趋势</h3>
        <div class="chart-container">
          <canvas ref="connectionsChart"></canvas>
        </div>
      </div>

      <!-- API 调用趋势 -->
      <div class="chart-card">
        <h3 class="chart-title">API 调用趋势</h3>
        <div class="chart-container">
          <canvas ref="apiCallsChart"></canvas>
        </div>
      </div>

      <!-- 响应时间分布 -->
      <div class="chart-card">
        <h3 class="chart-title">API 响应时间</h3>
        <div class="chart-container">
          <canvas ref="responseTimeChart"></canvas>
        </div>
      </div>

      <!-- 负载均衡分布 -->
      <div class="chart-card">
        <h3 class="chart-title">负载均衡分布</h3>
        <div class="chart-container">
          <canvas ref="loadBalancerChart"></canvas>
        </div>
      </div>
    </div>

    <!-- 详细指标表格 -->
    <div class="details-section">
      <h3 class="section-title">详细指标</h3>
      <div class="metrics-table">
        <table>
          <thead>
            <tr>
              <th>指标名称</th>
              <th>当前值</th>
              <th>说明</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="metric in detailedMetrics" :key="metric.name">
              <td>{{ metric.name }}</td>
              <td class="value">{{ metric.value }}</td>
              <td class="description">{{ metric.description }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import Chart from 'chart.js/auto'

const loading = ref(false)
const timeRange = ref('15m')
const metrics = ref({
  connections: 0,
  connectionsTrend: 0,
  apiCalls: 0,
  apiCallsTrend: 0,
  successRate: 0,
  avgResponseTime: 0,
  responseTimeTrend: 0,
  heartbeatsSent: 0,
  heartbeatsFailed: 0,
  compressionRate: 0,
  bytesSaved: 0
})

const detailedMetrics = ref([])

// Chart 实例
const connectionsChart = ref(null)
const apiCallsChart = ref(null)
const responseTimeChart = ref(null)
const loadBalancerChart = ref(null)

let charts = {}
let refreshInterval = null

// 获取监控数据
async function fetchMetrics() {
  try {
    const token = localStorage.getItem('token')
    const response = await fetch('/api/metrics/summary', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    const data = await response.json()
    
    metrics.value = {
      connections: data.connections || 0,
      connectionsTrend: data.connectionsTrend || 0,
      apiCalls: data.apiCalls || 0,
      apiCallsTrend: data.apiCallsTrend || 0,
      successRate: data.successRate || 0,
      avgResponseTime: data.avgResponseTime || 0,
      responseTimeTrend: data.responseTimeTrend || 0,
      heartbeatsSent: data.heartbeatsSent || 0,
      heartbeatsFailed: data.heartbeatsFailed || 0,
      compressionRate: data.compressionRate || 0,
      bytesSaved: data.bytesSaved || 0
    }

    detailedMetrics.value = data.detailedMetrics || []
    
    return data
  } catch (error) {
    console.error('获取监控数据失败:', error)
    return null
  }
}

// 获取时序数据
async function fetchTimeSeriesData() {
  try {
    const token = localStorage.getItem('token')
    const response = await fetch(`/api/metrics/timeseries?range=${timeRange.value}`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    return await response.json()
  } catch (error) {
    console.error('获取时序数据失败:', error)
    return null
  }
}

// 刷新数据
async function refreshData() {
  loading.value = true
  try {
    await fetchMetrics()
    const timeSeriesData = await fetchTimeSeriesData()
    if (timeSeriesData) {
      updateCharts(timeSeriesData)
    }
  } finally {
    loading.value = false
  }
}

// 初始化图表
function initCharts() {
  // 连接数趋势图
  if (connectionsChart.value) {
    charts.connections = new Chart(connectionsChart.value, {
      type: 'line',
      data: {
        labels: [],
        datasets: [{
          label: '连接数',
          data: [],
          borderColor: '#3b82f6',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          tension: 0.4,
          fill: true
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false }
        },
        scales: {
          y: { beginAtZero: true }
        }
      }
    })
  }

  // API 调用趋势图
  if (apiCallsChart.value) {
    charts.apiCalls = new Chart(apiCallsChart.value, {
      type: 'line',
      data: {
        labels: [],
        datasets: [
          {
            label: '成功',
            data: [],
            borderColor: '#10b981',
            backgroundColor: 'rgba(16, 185, 129, 0.1)',
            tension: 0.4,
            fill: true
          },
          {
            label: '失败',
            data: [],
            borderColor: '#ef4444',
            backgroundColor: 'rgba(239, 68, 68, 0.1)',
            tension: 0.4,
            fill: true
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: { beginAtZero: true }
        }
      }
    })
  }

  // 响应时间图
  if (responseTimeChart.value) {
    charts.responseTime = new Chart(responseTimeChart.value, {
      type: 'line',
      data: {
        labels: [],
        datasets: [
          {
            label: 'P50',
            data: [],
            borderColor: '#3b82f6',
            tension: 0.4
          },
          {
            label: 'P95',
            data: [],
            borderColor: '#f59e0b',
            tension: 0.4
          },
          {
            label: 'P99',
            data: [],
            borderColor: '#ef4444',
            tension: 0.4
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: { beginAtZero: true }
        }
      }
    })
  }

  // 负载均衡分布图
  if (loadBalancerChart.value) {
    charts.loadBalancer = new Chart(loadBalancerChart.value, {
      type: 'bar',
      data: {
        labels: [],
        datasets: [{
          label: '请求数',
          data: [],
          backgroundColor: [
            'rgba(59, 130, 246, 0.8)',
            'rgba(16, 185, 129, 0.8)',
            'rgba(245, 158, 11, 0.8)',
            'rgba(139, 92, 246, 0.8)'
          ]
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: { beginAtZero: true }
        }
      }
    })
  }
}

// 更新图表
function updateCharts(data) {
  if (charts.connections && data.connections) {
    charts.connections.data.labels = data.connections.labels
    charts.connections.data.datasets[0].data = data.connections.values
    charts.connections.update()
  }

  if (charts.apiCalls && data.apiCalls) {
    charts.apiCalls.data.labels = data.apiCalls.labels
    charts.apiCalls.data.datasets[0].data = data.apiCalls.success
    charts.apiCalls.data.datasets[1].data = data.apiCalls.failed
    charts.apiCalls.update()
  }

  if (charts.responseTime && data.responseTime) {
    charts.responseTime.data.labels = data.responseTime.labels
    charts.responseTime.data.datasets[0].data = data.responseTime.p50
    charts.responseTime.data.datasets[1].data = data.responseTime.p95
    charts.responseTime.data.datasets[2].data = data.responseTime.p99
    charts.responseTime.update()
  }

  if (charts.loadBalancer && data.loadBalancer) {
    charts.loadBalancer.data.labels = data.loadBalancer.labels
    charts.loadBalancer.data.datasets[0].data = data.loadBalancer.values
    charts.loadBalancer.update()
  }
}

// 格式化函数
function formatNumber(num) {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

function formatBytes(bytes) {
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(2) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(2) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return bytes + ' B'
}

function formatTrend(trend) {
  if (trend > 0) return `↑ ${trend.toFixed(1)}%`
  if (trend < 0) return `↓ ${Math.abs(trend).toFixed(1)}%`
  return '→ 0%'
}

function getTrendClass(trend) {
  if (trend > 0) return 'trend-up'
  if (trend < 0) return 'trend-down'
  return 'trend-neutral'
}

function getStatusClass(rate) {
  if (rate >= 95) return 'success'
  if (rate >= 90) return 'warning'
  return 'danger'
}

function getStatusText(rate) {
  if (rate >= 95) return '优秀'
  if (rate >= 90) return '良好'
  return '需关注'
}

onMounted(async () => {
  await refreshData()
  initCharts()
  
  // 每 10 秒自动刷新
  refreshInterval = setInterval(refreshData, 10000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
  
  // 销毁图表
  Object.values(charts).forEach(chart => {
    if (chart) chart.destroy()
  })
})
</script>

<style scoped>
.monitoring-container {
  padding: 2rem;
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.title {
  font-size: 2rem;
  font-weight: 700;
  color: #1e293b;
}

.header-actions {
  display: flex;
  gap: 1rem;
}

.btn-refresh {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s;
}

.btn-refresh:hover:not(:disabled) {
  background: #2563eb;
  transform: translateY(-1px);
}

.btn-refresh:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.time-select {
  padding: 0.75rem 1rem;
  border: 1px solid #e2e8f0;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
}

/* 指标卡片网格 */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.metric-card {
  background: white;
  border-radius: 1rem;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  gap: 1rem;
  transition: all 0.2s;
}

.metric-card:hover {
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.metric-icon {
  font-size: 2.5rem;
  line-height: 1;
}

.metric-content {
  flex: 1;
}

.metric-label {
  font-size: 0.875rem;
  color: #64748b;
  margin-bottom: 0.5rem;
}

.metric-value {
  font-size: 2rem;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 0.25rem;
}

.metric-trend {
  font-size: 0.875rem;
  font-weight: 500;
}

.trend-up {
  color: #10b981;
}

.trend-down {
  color: #ef4444;
}

.trend-neutral {
  color: #64748b;
}

.metric-status {
  font-size: 0.875rem;
  font-weight: 500;
}

.metric-status.success {
  color: #10b981;
}

.metric-status.warning {
  color: #f59e0b;
}

.metric-status.danger {
  color: #ef4444;
}

/* 图表区域 */
.charts-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.chart-card {
  background: white;
  border-radius: 1rem;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.chart-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 1rem;
}

.chart-container {
  height: 300px;
  position: relative;
}

/* 详细指标表格 */
.details-section {
  background: white;
  border-radius: 1rem;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.section-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 1rem;
}

.metrics-table {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

thead {
  background: #f8fafc;
}

th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.875rem;
  font-weight: 600;
  color: #475569;
  border-bottom: 2px solid #e2e8f0;
}

td {
  padding: 0.75rem 1rem;
  font-size: 0.875rem;
  color: #64748b;
  border-bottom: 1px solid #f1f5f9;
}

td.value {
  font-weight: 600;
  color: #1e293b;
}

td.description {
  color: #94a3b8;
}

tbody tr:hover {
  background: #f8fafc;
}

@media (max-width: 768px) {
  .monitoring-container {
    padding: 1rem;
  }

  .header {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }

  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .charts-section {
    grid-template-columns: 1fr;
  }
}
</style>
