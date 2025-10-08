#!/bin/bash

# Docker Redis集群启动脚本
echo "启动Redis集群..."

# 启动Redis集群服务
docker-compose up -d redis-master-1 redis-master-2 redis-master-3 redis-slave-1 redis-slave-2 redis-slave-3

# 等待Redis节点启动
echo "等待Redis节点启动..."
sleep 15

# 创建Redis集群
echo "创建Redis集群..."
docker exec -it redis-master-1 redis-cli --cluster create \
  redis-master-1:7001 \
  redis-master-2:7002 \
  redis-master-3:7003 \
  redis-slave-1:7004 \
  redis-slave-2:7005 \
  redis-slave-3:7006 \
  --cluster-replicas 1 --cluster-yes

# 检查集群状态
echo "检查集群状态..."
docker exec -it redis-master-1 redis-cli -p 7001 cluster info
docker exec -it redis-master-1 redis-cli -p 7001 cluster nodes

echo "Redis集群启动完成！"