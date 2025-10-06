// K6 秒杀系统性能测试脚本
import { HttpClient, AuthHelper, SeckillHelper, utils } from './utils.js';
import { getEnvironment, getScenario, generateTestUsers, generateTestActivities } from './load_test_config.js';

// 获取测试环境和场景配置
const environment = getEnvironment();
const scenario = getScenario();

// 测试选项
export let options = {
  scenarios: {
    [__ENV.TEST_SCENARIO || 'high_qps']: scenario
  },
  
  thresholds: {
    // HTTP请求失败率小于10%
    http_req_failed: ['rate<0.1'],
    // 95%的请求响应时间小于500ms
    http_req_duration: ['p(95)<500'],
    // 99%的请求响应时间小于1000ms
    'http_req_duration{expected_response:true}': ['p(99)<1000'],
    // 秒杀成功率大于1%
    seckill_success_rate: ['rate>0.01'],
    // 错误率小于15%
    error_rate: ['rate<0.15'],
  },
};

// 测试数据
let testUsers = [];
let testActivities = [];
let userTokens = new Map();

// 初始化测试数据
export function setup() {
  console.log('开始初始化测试数据...');
  
  // 生成测试用户和活动
  testUsers = generateTestUsers(1000);
  testActivities = generateTestActivities(10);
  
  console.log(`初始化完成: ${testUsers.length} 个用户, ${testActivities.length} 个活动`);
  
  return {
    users: testUsers,
    activities: testActivities
  };
}

// 主测试函数
export default function(data) {
  // 创建HTTP客户端
  const client = new HttpClient(environment.baseUrl);
  const authHelper = new AuthHelper(client);
  const seckillHelper = new SeckillHelper(client);
  
  // 随机选择用户和活动
  const user = data.users[Math.floor(Math.random() * data.users.length)];
  const activity = data.activities[Math.floor(Math.random() * data.activities.length)];
  
  // 用户登录（如果还没有token）
  let token = userTokens.get(user.username);
  if (!token) {
    const loginResult = authHelper.login({
      account: user.username,
      password: user.password
    });
    
    if (loginResult && loginResult.token) {
      token = loginResult.token;
      userTokens.set(user.username, token);
    } else {
      // 如果登录失败，尝试注册
      const registerResult = authHelper.register(user);
      if (registerResult) {
        const loginResult2 = authHelper.login({
          account: user.username,
          password: user.password
        });
        if (loginResult2 && loginResult2.token) {
          token = loginResult2.token;
          userTokens.set(user.username, token);
        }
      }
    }
  }
  
  if (!token) {
    console.error(`用户 ${user.username} 登录失败`);
    return;
  }
  
  // 执行不同的测试场景
  const scenario = Math.random();
  
  if (scenario < 0.7) {
    // 70% 概率执行秒杀
    executeSeckillScenario(seckillHelper, activity.id);
  } else if (scenario < 0.85) {
    // 15% 概率查询活动信息
    queryActivityScenario(seckillHelper);
  } else if (scenario < 0.95) {
    // 10% 概率查询库存
    queryStockScenario(seckillHelper, activity.id);
  } else {
    // 5% 概率预热活动
    prewarmActivityScenario(seckillHelper, activity.id);
  }
  
  // 随机延迟
  utils.randomSleep(0.1, 0.5);
}

// 秒杀场景
function executeSeckillScenario(seckillHelper, activityId) {
  const result = seckillHelper.executeSeckill(activityId, 1);
  
  if (result.success) {
    console.log(`秒杀成功，订单ID: ${result.orderId}, 响应时间: ${result.responseTime}ms`);
  } else {
    console.log(`秒杀失败，响应时间: ${result.responseTime}ms`);
  }
}

// 查询活动场景
function queryActivityScenario(seckillHelper) {
  const activities = seckillHelper.getActivities({
    page: 1,
    page_size: 10,
    status: 1
  });
  
  console.log(`查询到 ${activities.length} 个活动`);
}

// 查询库存场景
function queryStockScenario(seckillHelper, activityId) {
  const stock = seckillHelper.getStock(activityId);
  console.log(`活动 ${activityId} 剩余库存: ${stock}`);
}

// 预热活动场景
function prewarmActivityScenario(seckillHelper, activityId) {
  const success = seckillHelper.prewarmActivity(activityId);
  console.log(`活动 ${activityId} 预热${success ? '成功' : '失败'}`);
}

// 测试结束后的清理
export function teardown(data) {
  console.log('性能测试完成，开始清理...');
  
  // 清理用户tokens
  userTokens.clear();
  
  console.log('清理完成');
}