#!/bin/bash

echo "Testing PostgreSQL..."
docker-compose exec -T postgres pg_isready -U postgres || exit 1

echo "Testing RabbitMQ..."
docker-compose exec -T rabbitmq rabbitmq-diagnostics ping || exit 1

echo "Testing API is running..."
curl -s http://localhost:8080 > /dev/null
if [ $? -eq 0 ] || [ $? -eq 22 ]; then
  echo "API is responding"
else
  echo "API is not responding"
  exit 1
fi

echo "All services are healthy!"