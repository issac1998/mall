// K6 秒杀系统性能测试脚本
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
export const errorRate = new Rate('error_rate');
export const seckillSuccessRate = new Rate('seckill_success_rate');
export const responseTime = new Trend('response_time');

// 测试选项
export let options = {
  scenarios: {
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '10s',
    },
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },
        { duration: '1m', target: 10 },
        { duration: '30s', target: 0 },
      ],
    },
    high_qps: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 100,
      maxVUs: 200,
    }
  }
};

// 测试数据
const baseUrl = 'http://localhost:8080';
const testUsers = [];
const testActivities = [];

// 生成测试用户
function generateTestUsers(count = 20) {
  const users = [];
  for (let i = 0; i < count; i++) {
    users.push({
      username: `testuser${i}`,
      password: 'password123',
      phone: `1380000${String(i).padStart(4, '0')}`,
      email: `testuser${i}@example.com`
    });
  }
  return users;
}

// 生成测试活动
function generateTestActivities(count = 3) {
  const activities = [];
  for (let i = 0; i < count; i++) {
    activities.push({
      id: i + 1,
      name: `测试活动${i + 1}`,
      stock: 100,
      price: 1000 + i * 100
    });
  }
  return activities;
}

// 初始化测试数据
export function setup() {
  console.log('开始初始化测试数据...');
  
  const users = generateTestUsers(20);
  const activities = generateTestActivities(3);
  
  console.log(`初始化完成: ${users.length} 个用户, ${activities.length} 个活动`);
  
  return {
    users: users,
    activities: activities
  };
}

// 主测试函数
export default function(data) {
  const user = data.users[Math.floor(Math.random() * data.users.length)];
  const activity = data.activities[Math.floor(Math.random() * data.activities.length)];
  
  // 用户注册
  const registerResponse = http.post(`${baseUrl}/api/v1/auth/register`, JSON.stringify({
    username: user.username,
    password: user.password,
    phone: user.phone,
    email: user.email
  }), {
    headers: { 'Content-Type': 'application/json' }
  });
  
  const registerSuccess = check(registerResponse, {
    '注册成功或用户已存在': (r) => r.status === 200 || r.status === 400,
  });
  
  errorRate.add(!registerSuccess);
  responseTime.add(registerResponse.timings.duration);
  
  // 用户登录
  const loginResponse = http.post(`${baseUrl}/api/v1/auth/login`, JSON.stringify({
    account: user.username,
    password: user.password
  }), {
    headers: { 'Content-Type': 'application/json' }
  });
  
  const loginSuccess = check(loginResponse, {
    '登录成功': (r) => r.status === 200,
    '返回token': (r) => r.json('data.access_token') !== undefined,
  });
  
  errorRate.add(!loginSuccess);
  responseTime.add(loginResponse.timings.duration);
  
  if (!loginSuccess) {
    return;
  }
  
  const token = loginResponse.json('data.access_token');
  
  // 查询活动列表
  const activitiesResponse = http.get(`${baseUrl}/api/v1/activities`, {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    }
  });
  
  check(activitiesResponse, {
    '查询活动成功': (r) => r.status === 200,
  });
  
  responseTime.add(activitiesResponse.timings.duration);
  
  // 执行秒杀
  const requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  const seckillResponse = http.post(`${baseUrl}/api/v1/seckill/execute`, JSON.stringify({
    request_id: requestId,
    activity_id: activity.id,
    quantity: 1
  }), {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    }
  });
  
  const seckillSuccess = check(seckillResponse, {
    '秒杀请求成功': (r) => r.status === 200 || r.status === 400,
  });
  
  const isRealSuccess = seckillResponse.status === 200;
  seckillSuccessRate.add(isRealSuccess);
  errorRate.add(!seckillSuccess);
  responseTime.add(seckillResponse.timings.duration);
  
  sleep(1);
}

// 清理函数
export function teardown(data) {
  console.log('测试完成，开始清理...');
}