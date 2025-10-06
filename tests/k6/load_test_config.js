// K6 负载测试配置文件

// 测试环境配置
export const environments = {
  local: {
    baseUrl: 'http://localhost:8080',
    database: 'mysql://root:password@localhost:3306/seckill_test',
    redis: 'redis://localhost:6379/1'
  },
  dev: {
    baseUrl: 'http://dev.seckill.com',
    database: 'mysql://user:pass@dev-db:3306/seckill',
    redis: 'redis://dev-redis:6379/0'
  },
  staging: {
    baseUrl: 'http://staging.seckill.com',
    database: 'mysql://user:pass@staging-db:3306/seckill',
    redis: 'redis://staging-redis:6379/0'
  }
};

// 测试场景配置
export const scenarios = {
  // 烟雾测试 - 基本功能验证
  smoke: {
    executor: 'constant-vus',
    vus: 1,
    duration: '30s',
    thresholds: {
      http_req_duration: ['p(95)<1000'],
      http_req_failed: ['rate<0.01'],
    }
  },
  
  // 负载测试 - 正常负载下的性能
  load: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m', target: 100 },
      { duration: '5m', target: 100 },
      { duration: '2m', target: 200 },
      { duration: '5m', target: 200 },
      { duration: '2m', target: 0 },
    ],
    thresholds: {
      http_req_duration: ['p(95)<800'],
      http_req_failed: ['rate<0.05'],
    }
  },
  
  // 压力测试 - 超出正常负载
  stress: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m', target: 100 },
      { duration: '5m', target: 100 },
      { duration: '2m', target: 200 },
      { duration: '5m', target: 200 },
      { duration: '2m', target: 300 },
      { duration: '5m', target: 300 },
      { duration: '2m', target: 400 },
      { duration: '5m', target: 400 },
      { duration: '10m', target: 0 },
    ],
    thresholds: {
      http_req_duration: ['p(95)<1200'],
      http_req_failed: ['rate<0.1'],
    }
  },
  
  // 峰值测试 - 突发高负载
  spike: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '10s', target: 100 },
      { duration: '1m', target: 100 },
      { duration: '10s', target: 1400 },
      { duration: '3m', target: 1400 },
      { duration: '10s', target: 100 },
      { duration: '3m', target: 100 },
      { duration: '10s', target: 0 },
    ],
    thresholds: {
      http_req_duration: ['p(95)<2000'],
      http_req_failed: ['rate<0.2'],
    }
  },
  
  // 容量测试 - 最大容量测试
  volume: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m', target: 500 },
      { duration: '5m', target: 500 },
      { duration: '2m', target: 1000 },
      { duration: '5m', target: 1000 },
      { duration: '2m', target: 1500 },
      { duration: '5m', target: 1500 },
      { duration: '2m', target: 2000 },
      { duration: '10m', target: 2000 },
      { duration: '5m', target: 0 },
    ],
    thresholds: {
      http_req_duration: ['p(95)<3000'],
      http_req_failed: ['rate<0.3'],
    }
  },
  
  // 浸泡测试 - 长时间稳定性测试
  soak: {
    executor: 'constant-vus',
    vus: 400,
    duration: '1h',
    thresholds: {
      http_req_duration: ['p(95)<1000'],
      http_req_failed: ['rate<0.05'],
    }
  },
  
  // 高QPS测试 - 50000 QPS目标
  high_qps: {
    executor: 'constant-arrival-rate',
    rate: 50000,
    timeUnit: '1s',
    duration: '5m',
    preAllocatedVUs: 5000,
    maxVUs: 10000,
    thresholds: {
      http_req_duration: ['p(95)<500'],
      http_req_failed: ['rate<0.1'],
      seckill_success: ['rate>0.01'],
    }
  }
};

// 测试数据配置
export const testData = {
  // 用户数据
  users: {
    count: 1000,
    usernamePrefix: 'testuser',
    phonePrefix: '13800',
    emailDomain: 'example.com',
    password: 'password123'
  },
  
  // 活动数据
  activities: {
    count: 10,
    stockPerActivity: 1000,
    priceRange: [1000, 10000], // 价格范围（分）
    durationHours: 24
  },
  
  // 商品数据
  goods: {
    count: 50,
    namePrefix: '测试商品',
    priceRange: [5000, 50000], // 原价范围（分）
    categories: ['电子产品', '服装', '食品', '图书', '家居']
  }
};

// 监控指标配置
export const metrics = {
  // 业务指标
  business: [
    'seckill_success_rate',
    'seckill_failure_rate',
    'order_creation_rate',
    'payment_success_rate',
    'stock_deduction_accuracy'
  ],
  
  // 系统指标
  system: [
    'http_req_duration',
    'http_req_failed',
    'http_reqs',
    'vus',
    'vus_max',
    'data_received',
    'data_sent'
  ],
  
  // 自定义指标
  custom: [
    'response_time_p95',
    'response_time_p99',
    'error_rate',
    'throughput',
    'concurrent_users'
  ]
};

// 报告配置
export const reporting = {
  // 输出格式
  formats: ['json', 'html', 'csv'],
  
  // 报告路径
  outputDir: './test-results',
  
  // 实时监控
  realtime: {
    enabled: true,
    interval: '10s',
    dashboard: 'http://localhost:3000/dashboard'
  },
  
  // 告警配置
  alerts: {
    errorRateThreshold: 0.1,
    responseTimeThreshold: 1000,
    throughputThreshold: 1000
  }
};

// 获取当前环境配置
export function getEnvironment() {
  const env = __ENV.TEST_ENV || 'local';
  return environments[env] || environments.local;
}

// 获取测试场景配置
export function getScenario() {
  const scenario = __ENV.TEST_SCENARIO || 'load';
  return scenarios[scenario] || scenarios.load;
}

// 生成测试用户
export function generateTestUsers(count = testData.users.count) {
  const users = [];
  for (let i = 0; i < count; i++) {
    users.push({
      username: `${testData.users.usernamePrefix}${i}`,
      phone: `${testData.users.phonePrefix}${String(i).padStart(6, '0')}`,
      email: `${testData.users.usernamePrefix}${i}@${testData.users.emailDomain}`,
      password: testData.users.password
    });
  }
  return users;
}

// 生成测试活动
export function generateTestActivities(count = testData.activities.count) {
  const activities = [];
  const now = new Date();
  
  for (let i = 0; i < count; i++) {
    const startTime = new Date(now.getTime() + i * 60000); // 每个活动间隔1分钟
    const endTime = new Date(startTime.getTime() + testData.activities.durationHours * 3600000);
    
    activities.push({
      id: i + 1,
      name: `测试秒杀活动${i + 1}`,
      goods_id: (i % testData.goods.count) + 1,
      price: Math.floor(Math.random() * (testData.activities.priceRange[1] - testData.activities.priceRange[0])) + testData.activities.priceRange[0],
      stock: testData.activities.stockPerActivity,
      start_time: startTime.toISOString(),
      end_time: endTime.toISOString(),
      limit_per_user: 1,
      status: 1
    });
  }
  
  return activities;
}

// 导出默认配置
export default {
  environments,
  scenarios,
  testData,
  metrics,
  reporting,
  getEnvironment,
  getScenario,
  generateTestUsers,
  generateTestActivities
};