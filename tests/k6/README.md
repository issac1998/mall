# K6 秒杀系统性能测试

本目录包含了秒杀系统的K6性能测试脚本和配置文件。

## 文件结构

```
tests/k6/
├── README.md                    # 本文档
├── seckill_performance_test.js  # 主要的性能测试脚本
├── load_test_config.js          # 测试配置文件
├── utils.js                     # 测试工具函数
├── run_tests.sh                 # 测试运行脚本
└── test-results/                # 测试结果输出目录
```

## 快速开始

### 1. 安装K6

```bash
# macOS
brew install k6

# Ubuntu/Debian
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# CentOS/RHEL
sudo dnf install https://dl.k6.io/rpm/repo.rpm
sudo dnf install k6
```

### 2. 启动秒杀服务

确保秒杀服务在本地运行：

```bash
cd /Users/a/store
go run cmd/main.go
```

### 3. 运行测试

使用测试脚本运行不同场景的测试：

```bash
# 运行烟雾测试
./run_tests.sh --scenario smoke

# 运行负载测试
./run_tests.sh --scenario load

# 运行高QPS测试
./run_tests.sh --scenario high_qps --verbose

# 自定义URL和环境
./run_tests.sh --url http://localhost:8080 --env local --scenario stress
```

## 测试场景

### 1. 烟雾测试 (smoke)
- **目的**: 基本功能验证
- **配置**: 1个虚拟用户，持续30秒
- **阈值**: 95%请求响应时间 < 1000ms，错误率 < 1%

### 2. 负载测试 (load)
- **目的**: 正常负载下的性能测试
- **配置**: 逐步增加到200个虚拟用户
- **阈值**: 95%请求响应时间 < 800ms，错误率 < 5%

### 3. 压力测试 (stress)
- **目的**: 超出正常负载的性能测试
- **配置**: 逐步增加到400个虚拟用户
- **阈值**: 95%请求响应时间 < 1200ms，错误率 < 10%

### 4. 峰值测试 (spike)
- **目的**: 突发高负载测试
- **配置**: 快速增加到1400个虚拟用户
- **阈值**: 95%请求响应时间 < 2000ms，错误率 < 20%

### 5. 容量测试 (volume)
- **目的**: 最大容量测试
- **配置**: 逐步增加到2000个虚拟用户
- **阈值**: 95%请求响应时间 < 3000ms，错误率 < 30%

### 6. 浸泡测试 (soak)
- **目的**: 长时间稳定性测试
- **配置**: 400个虚拟用户，持续1小时
- **阈值**: 95%请求响应时间 < 1000ms，错误率 < 5%

### 7. 高QPS测试 (high_qps)
- **目的**: 50000 QPS目标测试
- **配置**: 50000请求/秒，持续5分钟
- **阈值**: 95%请求响应时间 < 500ms，错误率 < 10%，秒杀成功率 > 1%

## 测试指标

### 业务指标
- `seckill_success_rate`: 秒杀成功率
- `seckill_failure_rate`: 秒杀失败率
- `error_rate`: 总体错误率

### 系统指标
- `http_req_duration`: HTTP请求响应时间
- `http_req_failed`: HTTP请求失败率
- `http_reqs`: HTTP请求总数
- `response_time`: 自定义响应时间指标
- `request_count`: 自定义请求计数器

## 测试场景分布

测试脚本会模拟真实用户行为，按以下比例执行不同操作：

- **70%**: 执行秒杀操作
- **15%**: 查询活动信息
- **10%**: 查询库存信息
- **5%**: 预热活动

## 配置说明

### 环境配置

在 `load_test_config.js` 中定义了不同的测试环境：

```javascript
export const environments = {
  local: {
    baseUrl: 'http://localhost:8080',
    database: 'mysql://root:password@localhost:3306/seckill_test',
    redis: 'redis://localhost:6379/1'
  },
  dev: {
    baseUrl: 'http://dev.seckill.com',
    // ...
  },
  // ...
};
```

### 测试数据配置

```javascript
export const testData = {
  users: {
    count: 1000,           // 生成1000个测试用户
    usernamePrefix: 'testuser',
    phonePrefix: '13800',
    emailDomain: 'example.com',
    password: 'password123'
  },
  activities: {
    count: 10,             // 生成10个测试活动
    stockPerActivity: 1000, // 每个活动1000库存
    priceRange: [1000, 10000], // 价格范围（分）
    durationHours: 24      // 活动持续24小时
  }
};
```

## 结果分析

### 输出文件

测试完成后会生成以下文件：

- `seckill_test_{scenario}_{timestamp}.json`: JSON格式的详细结果
- `seckill_test_{scenario}_{timestamp}.csv`: CSV格式的时序数据
- `seckill_test_{scenario}_{timestamp}.html`: HTML格式的可视化报告（需要k6-reporter）

### 关键指标分析

1. **响应时间**
   - 平均响应时间应该在可接受范围内
   - P95和P99响应时间不应该过高
   - 响应时间分布应该相对稳定

2. **错误率**
   - HTTP错误率应该低于阈值
   - 业务错误（如库存不足）是正常的
   - 系统错误（如超时、连接失败）需要关注

3. **吞吐量**
   - 实际QPS与目标QPS的对比
   - 系统在不同负载下的吞吐量变化
   - 吞吐量的稳定性

4. **秒杀成功率**
   - 在高并发下的秒杀成功率
   - 成功率应该与库存量相匹配
   - 不应该出现超卖现象

## 故障排除

### 常见问题

1. **连接被拒绝**
   ```
   ERRO[0001] dial tcp 127.0.0.1:8080: connect: connection refused
   ```
   - 检查秒杀服务是否启动
   - 确认端口号是否正确

2. **认证失败**
   ```
   WARN[0001] Request Failed error="401 Unauthorized"
   ```
   - 检查用户注册和登录逻辑
   - 确认JWT token是否正确设置

3. **数据库连接失败**
   ```
   ERRO[0001] failed to connect to database
   ```
   - 检查数据库服务是否启动
   - 确认数据库连接配置

4. **Redis连接失败**
   ```
   ERRO[0001] failed to connect to redis
   ```
   - 检查Redis服务是否启动
   - 确认Redis连接配置

### 性能调优建议

1. **系统层面**
   - 调整数据库连接池大小
   - 优化Redis配置
   - 调整Go程序的GOMAXPROCS

2. **应用层面**
   - 优化数据库查询
   - 使用缓存减少数据库访问
   - 优化业务逻辑

3. **测试层面**
   - 调整虚拟用户数量
   - 优化测试脚本逻辑
   - 合理设置测试阈值

## 扩展测试

### 添加新的测试场景

1. 在 `load_test_config.js` 中添加新的场景配置
2. 在 `seckill_performance_test.js` 中添加对应的测试逻辑
3. 更新 `run_tests.sh` 脚本支持新场景

### 自定义指标

```javascript
import { Rate, Trend, Counter } from 'k6/metrics';

// 定义自定义指标
const customMetric = new Rate('custom_metric');

// 在测试中记录指标
export default function() {
  // ... 测试逻辑
  customMetric.add(success);
}
```

### 集成CI/CD

可以将性能测试集成到CI/CD流水线中：

```yaml
# .github/workflows/performance-test.yml
name: Performance Test
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  performance-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Install k6
        run: |
          sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      - name: Run performance tests
        run: |
          cd tests/k6
          ./run_tests.sh --scenario smoke --no-summary
```

## 参考资料

- [K6 官方文档](https://k6.io/docs/)
- [K6 性能测试最佳实践](https://k6.io/docs/testing-guides/test-types/)
- [秒杀系统设计文档](../../design/)