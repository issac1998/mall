package monitor

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	// 业务指标
	seckillRequestTotal    *prometheus.CounterVec
	seckillSuccessTotal    *prometheus.CounterVec
	seckillFailureTotal    *prometheus.CounterVec
	seckillDuration        *prometheus.HistogramVec
	stockDeductionTotal    *prometheus.CounterVec
	orderCreationTotal     *prometheus.CounterVec
	orderPaymentTotal      *prometheus.CounterVec
	userRegistrationTotal  *prometheus.CounterVec
	userLoginTotal         *prometheus.CounterVec
	activityPrewarmTotal   *prometheus.CounterVec
	
	// 系统指标
	httpRequestTotal       *prometheus.CounterVec
	httpRequestDuration    *prometheus.HistogramVec
	httpRequestSize        *prometheus.HistogramVec
	httpResponseSize       *prometheus.HistogramVec
	
	// 数据库指标
	dbConnectionsActive    prometheus.Gauge
	dbConnectionsIdle      prometheus.Gauge
	dbConnectionsTotal     prometheus.Gauge
	dbQueryTotal           *prometheus.CounterVec
	dbQueryDuration        *prometheus.HistogramVec
	
	// Redis指标
	redisConnectionsActive prometheus.Gauge
	redisCommandTotal      *prometheus.CounterVec
	redisCommandDuration   *prometheus.HistogramVec
	
	// 系统资源指标
	cpuUsage               prometheus.Gauge
	memoryUsage            prometheus.Gauge
	goroutineCount         prometheus.Gauge
	gcDuration             prometheus.Gauge
	
	// 队列指标
	queueMessageTotal      *prometheus.CounterVec
	queueMessageDuration   *prometheus.HistogramVec
	queueSize              *prometheus.GaugeVec
	
	mu sync.RWMutex
}

// NewMetricsCollector 创建新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{}
	mc.initMetrics()
	return mc
}

// initMetrics 初始化所有指标
func (mc *MetricsCollector) initMetrics() {
	// 业务指标
	mc.seckillRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "seckill_request_total",
			Help: "Total number of seckill requests",
		},
		[]string{"activity_id", "user_id", "status"},
	)
	
	mc.seckillSuccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "seckill_success_total",
			Help: "Total number of successful seckill requests",
		},
		[]string{"activity_id", "goods_id"},
	)
	
	mc.seckillFailureTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "seckill_failure_total",
			Help: "Total number of failed seckill requests",
		},
		[]string{"activity_id", "reason"},
	)
	
	mc.seckillDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "seckill_duration_seconds",
			Help:    "Duration of seckill requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"activity_id", "status"},
	)
	
	mc.stockDeductionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock_deduction_total",
			Help: "Total number of stock deductions",
		},
		[]string{"activity_id", "goods_id", "status"},
	)
	
	mc.orderCreationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_creation_total",
			Help: "Total number of order creations",
		},
		[]string{"activity_id", "status"},
	)
	
	mc.orderPaymentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_payment_total",
			Help: "Total number of order payments",
		},
		[]string{"payment_method", "status"},
	)
	
	mc.userRegistrationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_registration_total",
			Help: "Total number of user registrations",
		},
		[]string{"status"},
	)
	
	mc.userLoginTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_login_total",
			Help: "Total number of user logins",
		},
		[]string{"status"},
	)
	
	mc.activityPrewarmTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "activity_prewarm_total",
			Help: "Total number of activity prewarming",
		},
		[]string{"activity_id", "status"},
	)
	
	// 系统指标
	mc.httpRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	
	mc.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	
	mc.httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP requests",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)
	
	mc.httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)
	
	// 数据库指标
	mc.dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)
	
	mc.dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)
	
	mc.dbConnectionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_total",
			Help: "Total number of database connections",
		},
	)
	
	mc.dbQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table", "status"},
	)
	
	mc.dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)
	
	// Redis指标
	mc.redisConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_connections_active",
			Help: "Number of active Redis connections",
		},
	)
	
	mc.redisCommandTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_command_total",
			Help: "Total number of Redis commands",
		},
		[]string{"command", "status"},
	)
	
	mc.redisCommandDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_command_duration_seconds",
			Help:    "Duration of Redis commands",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command"},
	)
	
	// 系统资源指标
	mc.cpuUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "CPU usage percentage",
		},
	)
	
	mc.memoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
	)
	
	mc.goroutineCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutine_count",
			Help: "Number of goroutines",
		},
	)
	
	mc.gcDuration = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gc_duration_seconds",
			Help: "Duration of garbage collection",
		},
	)
	
	// 队列指标
	mc.queueMessageTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_message_total",
			Help: "Total number of queue messages",
		},
		[]string{"queue", "operation", "status"},
	)
	
	mc.queueMessageDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "queue_message_duration_seconds",
			Help:    "Duration of queue message processing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue", "operation"},
	)
	
	mc.queueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_size",
			Help: "Size of queue",
		},
		[]string{"queue"},
	)
}

// 业务指标记录方法

// RecordSeckillRequest 记录秒杀请求
func (mc *MetricsCollector) RecordSeckillRequest(activityID, userID, status string) {
	mc.seckillRequestTotal.WithLabelValues(activityID, userID, status).Inc()
}

// RecordSeckillSuccess 记录秒杀成功
func (mc *MetricsCollector) RecordSeckillSuccess(activityID, goodsID string) {
	mc.seckillSuccessTotal.WithLabelValues(activityID, goodsID).Inc()
}

// RecordSeckillFailure 记录秒杀失败
func (mc *MetricsCollector) RecordSeckillFailure(activityID, reason string) {
	mc.seckillFailureTotal.WithLabelValues(activityID, reason).Inc()
}

// RecordSeckillDuration 记录秒杀耗时
func (mc *MetricsCollector) RecordSeckillDuration(activityID, status string, duration time.Duration) {
	mc.seckillDuration.WithLabelValues(activityID, status).Observe(duration.Seconds())
}

// RecordStockDeduction 记录库存扣减
func (mc *MetricsCollector) RecordStockDeduction(activityID, goodsID, status string) {
	mc.stockDeductionTotal.WithLabelValues(activityID, goodsID, status).Inc()
}

// RecordOrderCreation 记录订单创建
func (mc *MetricsCollector) RecordOrderCreation(activityID, status string) {
	mc.orderCreationTotal.WithLabelValues(activityID, status).Inc()
}

// RecordOrderPayment 记录订单支付
func (mc *MetricsCollector) RecordOrderPayment(paymentMethod, status string) {
	mc.orderPaymentTotal.WithLabelValues(paymentMethod, status).Inc()
}

// RecordUserRegistration 记录用户注册
func (mc *MetricsCollector) RecordUserRegistration(status string) {
	mc.userRegistrationTotal.WithLabelValues(status).Inc()
}

// RecordUserLogin 记录用户登录
func (mc *MetricsCollector) RecordUserLogin(status string) {
	mc.userLoginTotal.WithLabelValues(status).Inc()
}

// RecordActivityPrewarm 记录活动预热
func (mc *MetricsCollector) RecordActivityPrewarm(activityID, status string) {
	mc.activityPrewarmTotal.WithLabelValues(activityID, status).Inc()
}

// 系统指标记录方法

// RecordHTTPRequest 记录HTTP请求
func (mc *MetricsCollector) RecordHTTPRequest(method, path, status string) {
	mc.httpRequestTotal.WithLabelValues(method, path, status).Inc()
}

// RecordHTTPDuration 记录HTTP请求耗时
func (mc *MetricsCollector) RecordHTTPDuration(method, path string, duration time.Duration) {
	mc.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordHTTPRequestSize 记录HTTP请求大小
func (mc *MetricsCollector) RecordHTTPRequestSize(method, path string, size float64) {
	mc.httpRequestSize.WithLabelValues(method, path).Observe(size)
}

// RecordHTTPResponseSize 记录HTTP响应大小
func (mc *MetricsCollector) RecordHTTPResponseSize(method, path string, size float64) {
	mc.httpResponseSize.WithLabelValues(method, path).Observe(size)
}

// 数据库指标记录方法

// UpdateDBConnections 更新数据库连接数
func (mc *MetricsCollector) UpdateDBConnections(active, idle, total int) {
	mc.dbConnectionsActive.Set(float64(active))
	mc.dbConnectionsIdle.Set(float64(idle))
	mc.dbConnectionsTotal.Set(float64(total))
}

// RecordDBQuery 记录数据库查询
func (mc *MetricsCollector) RecordDBQuery(operation, table, status string) {
	mc.dbQueryTotal.WithLabelValues(operation, table, status).Inc()
}

// RecordDBQueryDuration 记录数据库查询耗时
func (mc *MetricsCollector) RecordDBQueryDuration(operation, table string, duration time.Duration) {
	mc.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// Redis指标记录方法

// UpdateRedisConnections 更新Redis连接数
func (mc *MetricsCollector) UpdateRedisConnections(active int) {
	mc.redisConnectionsActive.Set(float64(active))
}

// RecordRedisCommand 记录Redis命令
func (mc *MetricsCollector) RecordRedisCommand(command, status string) {
	mc.redisCommandTotal.WithLabelValues(command, status).Inc()
}

// RecordRedisCommandDuration 记录Redis命令耗时
func (mc *MetricsCollector) RecordRedisCommandDuration(command string, duration time.Duration) {
	mc.redisCommandDuration.WithLabelValues(command).Observe(duration.Seconds())
}

// 系统资源指标更新方法

// UpdateSystemMetrics 更新系统指标
func (mc *MetricsCollector) UpdateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	mc.memoryUsage.Set(float64(m.Alloc))
	mc.goroutineCount.Set(float64(runtime.NumGoroutine()))
	mc.gcDuration.Set(float64(m.PauseTotalNs) / 1e9)
}

// 队列指标记录方法

// RecordQueueMessage 记录队列消息
func (mc *MetricsCollector) RecordQueueMessage(queue, operation, status string) {
	mc.queueMessageTotal.WithLabelValues(queue, operation, status).Inc()
}

// RecordQueueMessageDuration 记录队列消息处理耗时
func (mc *MetricsCollector) RecordQueueMessageDuration(queue, operation string, duration time.Duration) {
	mc.queueMessageDuration.WithLabelValues(queue, operation).Observe(duration.Seconds())
}

// UpdateQueueSize 更新队列大小
func (mc *MetricsCollector) UpdateQueueSize(queue string, size int) {
	mc.queueSize.WithLabelValues(queue).Set(float64(size))
}

// StartSystemMetricsCollection 启动系统指标收集
func (mc *MetricsCollector) StartSystemMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.UpdateSystemMetrics()
		}
	}
}

// GetRegistry 获取Prometheus注册器
func (mc *MetricsCollector) GetRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}