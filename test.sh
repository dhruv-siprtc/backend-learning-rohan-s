#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

echo "=========================================="
echo "  Paota Event System - Integration Tests"
echo "=========================================="
echo ""

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((TESTS_FAILED++))
    fi
}

# Function to wait for API
wait_for_api() {
    echo -e "${YELLOW}Waiting for API to be ready...${NC}"
    for i in {1..30}; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            echo -e "${GREEN}API is ready!${NC}"
            return 0
        fi
        sleep 2
        echo -n "."
    done
    echo -e "${RED}API failed to start${NC}"
    return 1
}

# Function to wait for consumers
wait_for_consumers() {
    echo -e "${YELLOW}Waiting for consumers to be ready...${NC}"
    sleep 10
    
    # Check if consumers are registered
    CREATED_CONSUMERS=$(docker compose exec -T rabbitmq rabbitmqctl list_queues name consumers 2>/dev/null | grep "user.created.queue" | awk '{print $2}')
    UPDATED_CONSUMERS=$(docker compose exec -T rabbitmq rabbitmqctl list_queues name consumers 2>/dev/null | grep "user.updated.queue" | awk '{print $2}')
    
    if [ "$CREATED_CONSUMERS" -ge 1 ] && [ "$UPDATED_CONSUMERS" -ge 1 ]; then
        echo -e "${GREEN}Consumers are ready!${NC}"
        return 0
    else
        echo -e "${RED}Consumers failed to register${NC}"
        return 1
    fi
}

# Test 1: Check Docker Compose Services
echo -e "\n${YELLOW}Test 1: Checking Docker services...${NC}"
if docker compose ps | grep -q "Up"; then
    print_result 0 "Docker services are running"
else
    print_result 1 "Docker services are not running"
    exit 1
fi

# Test 2: Wait for API
echo -e "\n${YELLOW}Test 2: Checking API health...${NC}"
if wait_for_api; then
    print_result 0 "API is healthy"
else
    print_result 1 "API is not responding"
    exit 1
fi

# Test 3: Check RabbitMQ queues exist
echo -e "\n${YELLOW}Test 3: Checking RabbitMQ queues...${NC}"
QUEUES=$(docker compose exec -T rabbitmq rabbitmqctl list_queues name 2>/dev/null)
if echo "$QUEUES" | grep -q "user.created.queue" && echo "$QUEUES" | grep -q "user.updated.queue"; then
    print_result 0 "RabbitMQ queues exist"
else
    print_result 1 "RabbitMQ queues not found"
fi

# Test 4: Check consumers are connected
echo -e "\n${YELLOW}Test 4: Checking consumer connections...${NC}"
if wait_for_consumers; then
    print_result 0 "Consumers are connected"
else
    print_result 1 "Consumers are not connected"
fi

# Test 5: Create a user and verify USER_CREATED event
echo -e "\n${YELLOW}Test 5: Testing USER_CREATED event...${NC}"
CREATE_RESPONSE=$(curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User 1","email":"test1@example.com","password":"password123"}')

if echo "$CREATE_RESPONSE" | grep -q "test1@example.com"; then
    print_result 0 "User created successfully"
    USER_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
    
    # Wait for event processing
    sleep 3
    
    # Check consumer logs
    if docker compose logs consumer 2>/dev/null | grep -q "USER_CREATED.*test1@example.com"; then
        print_result 0 "USER_CREATED event consumed successfully"
    else
        print_result 1 "USER_CREATED event not found in consumer logs"
    fi
else
    print_result 1 "Failed to create user"
fi

# Test 6: Update user and verify USER_UPDATED event
if [ ! -z "$USER_ID" ]; then
    echo -e "\n${YELLOW}Test 6: Testing USER_UPDATED event...${NC}"
    UPDATE_RESPONSE=$(curl -s -X PUT http://localhost:8080/users/$USER_ID \
      -H "Content-Type: application/json" \
      -d '{"name":"Test User Updated","email":"test1.updated@example.com","password":"newpassword"}')
    
    if echo "$UPDATE_RESPONSE" | grep -q "test1.updated@example.com"; then
        print_result 0 "User updated successfully"
        
        # Wait for event processing
        sleep 3
        
        # Check consumer logs
        if docker compose logs consumer 2>/dev/null | grep -q "USER_UPDATED.*test1.updated@example.com"; then
            print_result 0 "USER_UPDATED event consumed successfully"
        else
            print_result 1 "USER_UPDATED event not found in consumer logs"
        fi
    else
        print_result 1 "Failed to update user"
    fi
fi

# Test 7: Multiple concurrent events
echo -e "\n${YELLOW}Test 7: Testing multiple concurrent events...${NC}"
for i in {1..5}; do
    curl -s -X POST http://localhost:8080/users \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"Bulk User $i\",\"email\":\"bulk$i@example.com\",\"password\":\"pass$i\"}" > /dev/null &
done
wait

sleep 5

# Count events in logs
CREATED_COUNT=$(docker compose logs consumer 2>/dev/null | grep "USER_CREATED.*bulk.*@example.com" | wc -l)
if [ "$CREATED_COUNT" -ge 5 ]; then
    print_result 0 "Multiple concurrent events processed (found $CREATED_COUNT)"
else
    print_result 1 "Not all concurrent events processed (found $CREATED_COUNT, expected 5)"
fi

# Test 8: Check queue depths
echo -e "\n${YELLOW}Test 8: Checking queue depths...${NC}"
QUEUE_DEPTHS=$(docker compose exec -T rabbitmq rabbitmqctl list_queues name messages 2>/dev/null)
CREATED_DEPTH=$(echo "$QUEUE_DEPTHS" | grep "user.created.queue" | awk '{print $2}')
UPDATED_DEPTH=$(echo "$QUEUE_DEPTHS" | grep "user.updated.queue" | awk '{print $2}')

if [ "$CREATED_DEPTH" -eq 0 ] && [ "$UPDATED_DEPTH" -eq 0 ]; then
    print_result 0 "All messages processed, queues are empty"
else
    print_result 1 "Messages still in queues (created: $CREATED_DEPTH, updated: $UPDATED_DEPTH)"
fi

# Test 9: Consumer resilience test
echo -e "\n${YELLOW}Test 9: Testing consumer resilience...${NC}"
docker compose stop consumer > /dev/null 2>&1
sleep 2

# Send message while consumer is down
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Resilience Test","email":"resilience@example.com","password":"test123"}' > /dev/null

sleep 2

# Check message is queued
QUEUED=$(docker compose exec -T rabbitmq rabbitmqctl list_queues name messages 2>/dev/null | grep "user.created.queue" | awk '{print $2}')
if [ "$QUEUED" -ge 1 ]; then
    print_result 0 "Message queued while consumer was down"
    
    # Restart consumer
    docker compose start consumer > /dev/null 2>&1
    sleep 10
    
    # Check if message was processed
    if docker compose logs consumer 2>/dev/null | grep -q "resilience@example.com"; then
        print_result 0 "Message processed after consumer restart"
    else
        print_result 1 "Message not processed after consumer restart"
    fi
else
    print_result 1 "Message not queued while consumer was down"
fi

# Test 10: Check for errors in logs
echo -e "\n${YELLOW}Test 10: Checking for errors in logs...${NC}"
ERROR_COUNT=$(docker compose logs 2>/dev/null | grep -i "error\|failed\|fatal" | grep -v "Failed to" | wc -l)
if [ "$ERROR_COUNT" -eq 0 ]; then
    print_result 0 "No errors found in logs"
else
    print_result 1 "Found $ERROR_COUNT errors in logs"
    echo -e "${RED}Sample errors:${NC}"
    docker compose logs 2>/dev/null | grep -i "error\|failed\|fatal" | grep -v "Failed to" | head -n 5
fi

# Summary
echo ""
echo "=========================================="
echo "           Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo "=========================================="

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed! ✗${NC}"
    echo ""
    echo "To debug failures:"
    echo "  - Check API logs: docker compose logs api"
    echo "  - Check consumer logs: docker compose logs consumer"
    echo "  - Check RabbitMQ: docker compose logs rabbitmq"
    echo "  - Check queues: docker compose exec rabbitmq rabbitmqctl list_queues name messages consumers"
    exit 1
fi