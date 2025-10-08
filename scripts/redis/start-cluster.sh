#!/bin/bash

# 本地Redis集群启动脚本

set -e

echo "启动本地Redis集群..."

# 创建数据目录
mkdir -p /tmp/redis-cluster/{7001,7002,7003,7004,7005,7006}

# 启动6个Redis实例
for port in 7001 7002 7003 7004 7005 7006; do
    echo "启动Redis节点 $port..."
    redis-server --port $port \
                 --cluster-enabled yes \
                 --cluster-config-file /tmp/redis-cluster/$port/nodes.conf \
                 --cluster-node-timeout 5000 \
                 --appendonly yes \
                 --appendfilename appendonly-$port.aof \
                 --dbfilename dump-$port.rdb \
                 --logfile /tmp/redis-cluster/$port/redis.log \
                 --daemonize yes \
                 --protected-mode no \
                 --dir /tmp/redis-cluster/$port/
done

echo "等待Redis节点启动..."
sleep 5

# 检查所有Redis节点是否启动
for port in 7001 7002 7003 7004 7005 7006; do
    echo "检查Redis节点 127.0.0.1:$port..."
    while ! redis-cli -h 127.0.0.1 -p $port ping > /dev/null 2>&1; do
        echo "等待Redis节点 127.0.0.1:$port 启动..."
        sleep 2
    done
    echo "Redis节点 127.0.0.1:$port 已启动"
done

echo "所有Redis节点已启动，开始创建集群..."

# 创建Redis集群
redis-cli --cluster create \
    127.0.0.1:7001 \
    127.0.0.1:7002 \
    127.0.0.1:7003 \
    127.0.0.1:7004 \
    127.0.0.1:7005 \
    127.0.0.1:7006 \
    --cluster-replicas 1 \
    --cluster-yes

echo "Redis集群创建完成！"

# 检查集群状态
echo "检查集群状态..."
redis-cli -h 127.0.0.1 -p 7001 cluster info
redis-cli -h 127.0.0.1 -p 7001 cluster nodes

echo "Redis集群启动完成！"