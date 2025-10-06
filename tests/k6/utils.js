// K6 测试工具函数

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
export const errorRate = new Rate('error_rate');
export const seckillSuccessRate = new Rate('seckill_success_rate');
export const seckillFailureRate = new Rate('seckill_failure_rate');
export const responseTime = new Trend('response_time');
export const requestCount = new Counter('request_count');

// HTTP 请求配置
export const httpConfig = {
  timeout: '30s',
  headers: {
    'Content-Type': 'application/json',
    'User-Agent': 'k6-load-test/1.0'
  }
};

// 基础HTTP请求封装
export class HttpClient {
  constructor(baseUrl, options = {}) {
    this.baseUrl = baseUrl;
    this.options = { ...httpConfig, ...options };
    this.token = null;
  }

  // 设置认证token
  setToken(token) {
    this.token = token;
    this.options.headers['Authorization'] = `Bearer ${token}`;
  }

  // GET请求
  get(path, params = {}) {
    const url = this.buildUrl(path, params);
    const response = http.get(url, this.options);
    this.recordMetrics(response);
    return response;
  }

  // POST请求
  post(path, data = {}, params = {}) {
    const url = this.buildUrl(path, params);
    const response = http.post(url, JSON.stringify(data), this.options);
    this.recordMetrics(response);
    return response;
  }

  // PUT请求
  put(path, data = {}, params = {}) {
    const url = this.buildUrl(path, params);
    const response = http.put(url, JSON.stringify(data), this.options);
    this.recordMetrics(response);
    return response;
  }

  // DELETE请求
  delete(path, params = {}) {
    const url = this.buildUrl(path, params);
    const response = http.del(url, null, this.options);
    this.recordMetrics(response);
    return response;
  }

  // 构建URL
  buildUrl(path, params = {}) {
    let url = `${this.baseUrl}${path}`;
    const queryString = Object.keys(params)
      .map(key => `${encodeURIComponent(key)}=${encodeURIComponent(params[key])}`)
      .join('&');
    
    if (queryString) {
      url += `?${queryString}`;
    }
    
    return url;
  }

  // 记录指标
  recordMetrics(response) {
    requestCount.add(1);
    responseTime.add(response.timings.duration);
    errorRate.add(response.status >= 400);
  }
}

// 用户认证类
export class AuthHelper {
  constructor(client) {
    this.client = client;
  }

  // 用户注册
  async register(userData) {
    const response = this.client.post('/api/auth/register', userData);
    
    const success = check(response, {
      'register status is 200': (r) => r.status === 200,
      'register response has user_id': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.user_id;
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        return body.data;
      } catch (e) {
        console.error('Failed to parse register response:', e);
        return null;
      }
    }

    return null;
  }

  // 用户登录
  async login(credentials) {
    const response = this.client.post('/api/auth/login', credentials);
    
    const success = check(response, {
      'login status is 200': (r) => r.status === 200,
      'login response has token': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.token;
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        const token = body.data.token;
        this.client.setToken(token);
        return body.data;
      } catch (e) {
        console.error('Failed to parse login response:', e);
        return null;
      }
    }

    return null;
  }

  // 获取用户信息
  async getUserInfo() {
    const response = this.client.get('/api/auth/me');
    
    const success = check(response, {
      'get user info status is 200': (r) => r.status === 200,
      'get user info response has user data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.id;
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        return body.data;
      } catch (e) {
        console.error('Failed to parse user info response:', e);
        return null;
      }
    }

    return null;
  }
}

// 秒杀助手类
export class SeckillHelper {
  constructor(client) {
    this.client = client;
  }

  // 获取活动列表
  async getActivities(params = {}) {
    const response = this.client.get('/api/seckill/activities', params);
    
    const success = check(response, {
      'get activities status is 200': (r) => r.status === 200,
      'get activities response has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && Array.isArray(body.data.list);
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        return body.data.list;
      } catch (e) {
        console.error('Failed to parse activities response:', e);
        return [];
      }
    }

    return [];
  }

  // 活动预热
  async prewarmActivity(activityId) {
    const response = this.client.post(`/api/seckill/activities/${activityId}/prewarm`);
    
    const success = check(response, {
      'prewarm activity status is 200': (r) => r.status === 200,
      'prewarm activity response success': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.code === 0;
        } catch (e) {
          return false;
        }
      }
    });

    return success;
  }

  // 执行秒杀
  async executeSeckill(activityId, quantity = 1) {
    const startTime = Date.now();
    const response = this.client.post('/api/seckill/execute', {
      activity_id: activityId,
      quantity: quantity
    });
    const endTime = Date.now();

    const isSuccess = response.status === 200;
    let seckillSuccess = false;

    if (isSuccess) {
      try {
        const body = JSON.parse(response.body);
        seckillSuccess = body.code === 0 && body.data && body.data.order_id;
      } catch (e) {
        console.error('Failed to parse seckill response:', e);
      }
    }

    // 记录秒杀指标
    seckillSuccessRate.add(seckillSuccess);
    seckillFailureRate.add(!seckillSuccess);

    const result = check(response, {
      'seckill request completed': (r) => r.status !== 0,
      'seckill response time < 500ms': () => (endTime - startTime) < 500,
    });

    return {
      success: seckillSuccess,
      response: response,
      responseTime: endTime - startTime,
      orderId: seckillSuccess ? JSON.parse(response.body).data.order_id : null
    };
  }

  // 查询秒杀结果
  async getSeckillResult(requestId) {
    const response = this.client.get(`/api/seckill/result/${requestId}`);
    
    const success = check(response, {
      'get seckill result status is 200': (r) => r.status === 200,
      'get seckill result response has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data !== undefined;
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        return body.data;
      } catch (e) {
        console.error('Failed to parse seckill result response:', e);
        return null;
      }
    }

    return null;
  }

  // 获取库存信息
  async getStock(activityId) {
    const response = this.client.get(`/api/seckill/activities/${activityId}/stock`);
    
    const success = check(response, {
      'get stock status is 200': (r) => r.status === 200,
      'get stock response has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && typeof body.data.stock === 'number';
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        return body.data.stock;
      } catch (e) {
        console.error('Failed to parse stock response:', e);
        return 0;
      }
    }

    return 0;
  }
}

// 订单助手类
export class OrderHelper {
  constructor(client) {
    this.client = client;
  }

  // 获取订单列表
  async getOrders(params = {}) {
    const response = this.client.get('/api/orders', params);
    
    const success = check(response, {
      'get orders status is 200': (r) => r.status === 200,
      'get orders response has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && Array.isArray(body.data.list);
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        return body.data.list;
      } catch (e) {
        console.error('Failed to parse orders response:', e);
        return [];
      }
    }

    return [];
  }

  // 获取订单详情
  async getOrderDetail(orderId) {
    const response = this.client.get(`/api/orders/${orderId}`);
    
    const success = check(response, {
      'get order detail status is 200': (r) => r.status === 200,
      'get order detail response has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.id;
        } catch (e) {
          return false;
        }
      }
    });

    if (success && response.status === 200) {
      try {
        const body = JSON.parse(response.body);
        return body.data;
      } catch (e) {
        console.error('Failed to parse order detail response:', e);
        return null;
      }
    }

    return null;
  }

  // 支付订单
  async payOrder(orderId, paymentData = {}) {
    const response = this.client.post(`/api/orders/${orderId}/pay`, paymentData);
    
    const success = check(response, {
      'pay order status is 200': (r) => r.status === 200,
      'pay order response success': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.code === 0;
        } catch (e) {
          return false;
        }
      }
    });

    return success;
  }
}

// 工具函数
export const utils = {
  // 随机延迟
  randomSleep(min = 1, max = 3) {
    const delay = Math.random() * (max - min) + min;
    sleep(delay);
  },

  // 生成随机字符串
  randomString(length = 8) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  },

  // 生成随机数字
  randomNumber(min = 0, max = 100) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  },

  // 生成随机手机号
  randomPhone() {
    const prefixes = ['138', '139', '150', '151', '152', '158', '159', '188', '189'];
    const prefix = prefixes[Math.floor(Math.random() * prefixes.length)];
    const suffix = String(Math.floor(Math.random() * 100000000)).padStart(8, '0');
    return prefix + suffix;
  },

  // 生成随机邮箱
  randomEmail() {
    const domains = ['example.com', 'test.com', 'demo.com'];
    const domain = domains[Math.floor(Math.random() * domains.length)];
    const username = this.randomString(8);
    return `${username}@${domain}`;
  },

  // 格式化时间
  formatTime(timestamp) {
    return new Date(timestamp).toISOString();
  },

  // 计算成功率
  calculateSuccessRate(successCount, totalCount) {
    return totalCount > 0 ? (successCount / totalCount * 100).toFixed(2) : 0;
  },

  // 等待条件满足
  async waitFor(condition, timeout = 30000, interval = 1000) {
    const startTime = Date.now();
    
    while (Date.now() - startTime < timeout) {
      if (await condition()) {
        return true;
      }
      sleep(interval / 1000);
    }
    
    return false;
  }
};

// 导出所有工具
export default {
  HttpClient,
  AuthHelper,
  SeckillHelper,
  OrderHelper,
  utils,
  errorRate,
  seckillSuccessRate,
  seckillFailureRate,
  responseTime,
  requestCount
};