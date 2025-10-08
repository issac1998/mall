#!/bin/bash

# 本地Redis集群停止脚本

echo "停止Redis集群..."

# 停止所有Redis实例
for port in 7001 7002 7003 7004 7005 7006; do
    echo "停止Redis节点 $port..."
    redis-cli -h 127.0.0.1 -p $port shutdown nosave || true
done

echo "等待Redis节点停止..."
sleep 3

# 清理数据目录
echo "清理数据目录..."
rm -rf /tmp/redis-cluster

echo "Redis集群已停止！"