#!/bin/bash

# K6 性能测试运行脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
DEFAULT_BASE_URL="http://localhost:8080"
DEFAULT_TEST_ENV="local"
DEFAULT_SCENARIO="load"
DEFAULT_OUTPUT_DIR="./test-results"

# 帮助信息
show_help() {
    echo "K6 秒杀系统性能测试脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help              显示帮助信息"
    echo "  -u, --url URL           设置基础URL (默认: $DEFAULT_BASE_URL)"
    echo "  -e, --env ENV           设置测试环境 (local|dev|staging, 默认: $DEFAULT_TEST_ENV)"
    echo "  -s, --scenario SCENARIO 设置测试场景 (smoke|load|stress|spike|volume|soak|high_qps, 默认: $DEFAULT_SCENARIO)"
    echo "  -o, --output DIR        设置输出目录 (默认: $DEFAULT_OUTPUT_DIR)"
    echo "  -v, --verbose           详细输出"
    echo "  --no-summary           不显示测试摘要"
    echo "  --no-thresholds        忽略阈值检查"
    echo ""
    echo "测试场景说明:"
    echo "  smoke     - 烟雾测试，基本功能验证"
    echo "  load      - 负载测试，正常负载下的性能"
    echo "  stress    - 压力测试，超出正常负载"
    echo "  spike     - 峰值测试，突发高负载"
    echo "  volume    - 容量测试，最大容量测试"
    echo "  soak      - 浸泡测试，长时间稳定性测试"
    echo "  high_qps  - 高QPS测试，50000 QPS目标"
    echo ""
    echo "示例:"
    echo "  $0 --scenario smoke                    # 运行烟雾测试"
    echo "  $0 --scenario load --env dev           # 在开发环境运行负载测试"
    echo "  $0 --scenario high_qps --verbose       # 运行高QPS测试并显示详细输出"
}

# 解析命令行参数
BASE_URL="$DEFAULT_BASE_URL"
TEST_ENV="$DEFAULT_TEST_ENV"
SCENARIO="$DEFAULT_SCENARIO"
OUTPUT_DIR="$DEFAULT_OUTPUT_DIR"
VERBOSE=false
NO_SUMMARY=false
NO_THRESHOLDS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -u|--url)
            BASE_URL="$2"
            shift 2
            ;;
        -e|--env)
            TEST_ENV="$2"
            shift 2
            ;;
        -s|--scenario)
            SCENARIO="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --no-summary)
            NO_SUMMARY=true
            shift
            ;;
        --no-thresholds)
            NO_THRESHOLDS=true
            shift
            ;;
        *)
            echo -e "${RED}错误: 未知选项 $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 检查k6是否安装
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}错误: k6 未安装${NC}"
    echo "请访问 https://k6.io/docs/getting-started/installation/ 安装k6"
    exit 1
fi

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 生成时间戳
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULT_FILE="$OUTPUT_DIR/seckill_test_${SCENARIO}_${TIMESTAMP}"

# 构建k6命令
K6_CMD="k6 run"

# 添加环境变量
K6_CMD="$K6_CMD --env BASE_URL=$BASE_URL"
K6_CMD="$K6_CMD --env TEST_ENV=$TEST_ENV"
K6_CMD="$K6_CMD --env TEST_SCENARIO=$SCENARIO"

# 添加输出选项
K6_CMD="$K6_CMD --out json=${RESULT_FILE}.json"
K6_CMD="$K6_CMD --out csv=${RESULT_FILE}.csv"

# 添加其他选项
if [ "$VERBOSE" = true ]; then
    K6_CMD="$K6_CMD --verbose"
fi

if [ "$NO_SUMMARY" = true ]; then
    K6_CMD="$K6_CMD --no-summary"
fi

if [ "$NO_THRESHOLDS" = true ]; then
    K6_CMD="$K6_CMD --no-thresholds"
fi

# 添加测试脚本
K6_CMD="$K6_CMD tests/k6/seckill_performance_test.js"

# 显示测试信息
echo -e "${BLUE}==================== K6 秒杀系统性能测试 ====================${NC}"
echo -e "${YELLOW}测试环境:${NC} $TEST_ENV"
echo -e "${YELLOW}基础URL:${NC} $BASE_URL"
echo -e "${YELLOW}测试场景:${NC} $SCENARIO"
echo -e "${YELLOW}输出目录:${NC} $OUTPUT_DIR"
echo -e "${YELLOW}结果文件:${NC} ${RESULT_FILE}.{json,csv}"
echo -e "${BLUE}=========================================================${NC}"
echo ""

# 检查服务是否可用
echo -e "${YELLOW}检查服务可用性...${NC}"
if curl -s --connect-timeout 5 "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 服务可用${NC}"
else
    echo -e "${RED}✗ 服务不可用，请检查服务是否启动${NC}"
    echo -e "${YELLOW}提示: 请确保秒杀服务在 $BASE_URL 上运行${NC}"
    exit 1
fi

# 运行测试
echo -e "${YELLOW}开始运行性能测试...${NC}"
echo -e "${BLUE}命令: $K6_CMD${NC}"
echo ""

# 记录开始时间
START_TIME=$(date +%s)

# 执行k6测试
if eval "$K6_CMD"; then
    # 记录结束时间
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    echo ""
    echo -e "${GREEN}==================== 测试完成 ====================${NC}"
    echo -e "${GREEN}✓ 测试成功完成${NC}"
    echo -e "${YELLOW}测试时长:${NC} ${DURATION}秒"
    echo -e "${YELLOW}结果文件:${NC}"
    echo -e "  - JSON: ${RESULT_FILE}.json"
    echo -e "  - CSV:  ${RESULT_FILE}.csv"
    
    # 生成HTML报告
    if command -v k6-reporter &> /dev/null; then
        echo -e "${YELLOW}生成HTML报告...${NC}"
        k6-reporter "${RESULT_FILE}.json" --output "${RESULT_FILE}.html"
        echo -e "  - HTML: ${RESULT_FILE}.html"
    fi
    
    echo -e "${BLUE}=================================================${NC}"
    
    # 显示简要统计
    if [ -f "${RESULT_FILE}.json" ] && command -v jq &> /dev/null; then
        echo ""
        echo -e "${YELLOW}测试摘要:${NC}"
        
        # 提取关键指标
        TOTAL_REQUESTS=$(jq -r '.metrics.http_reqs.values.count // 0' "${RESULT_FILE}.json")
        FAILED_REQUESTS=$(jq -r '.metrics.http_req_failed.values.fails // 0' "${RESULT_FILE}.json")
        AVG_DURATION=$(jq -r '.metrics.http_req_duration.values.avg // 0' "${RESULT_FILE}.json")
        P95_DURATION=$(jq -r '.metrics.http_req_duration.values["p(95)"] // 0' "${RESULT_FILE}.json")
        
        if [ "$TOTAL_REQUESTS" != "0" ]; then
            FAILURE_RATE=$(echo "scale=2; $FAILED_REQUESTS * 100 / $TOTAL_REQUESTS" | bc -l 2>/dev/null || echo "0")
        else
            FAILURE_RATE="0"
        fi
        
        echo -e "  总请求数: ${TOTAL_REQUESTS}"
        echo -e "  失败请求: ${FAILED_REQUESTS}"
        echo -e "  失败率: ${FAILURE_RATE}%"
        echo -e "  平均响应时间: ${AVG_DURATION}ms"
        echo -e "  P95响应时间: ${P95_DURATION}ms"
    fi
    
else
    echo ""
    echo -e "${RED}==================== 测试失败 ====================${NC}"
    echo -e "${RED}✗ 测试执行失败${NC}"
    echo -e "${YELLOW}请检查错误信息并重试${NC}"
    echo -e "${BLUE}=================================================${NC}"
    exit 1
fi