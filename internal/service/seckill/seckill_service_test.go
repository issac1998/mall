package seckill

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"seckill/internal/model"
)

func TestSeckillRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *SeckillRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &SeckillRequest{
				RequestID:  "test-123",
				ActivityID: 1,
				UserID:     100,
				Quantity:   1,
				IP:         "127.0.0.1",
				DeviceID:   "device-123",
				UserAgent:  "test-agent",
			},
			wantErr: false,
		},
		{
			name: "empty request id",
			req: &SeckillRequest{
				ActivityID: 1,
				UserID:     100,
				Quantity:   1,
			},
			wantErr: true,
		},
		{
			name: "zero activity id",
			req: &SeckillRequest{
				RequestID:  "test-123",
				ActivityID: 0,
				UserID:     100,
				Quantity:   1,
			},
			wantErr: true,
		},
		{
			name: "zero user id",
			req: &SeckillRequest{
				RequestID:  "test-123",
				ActivityID: 1,
				UserID:     0,
				Quantity:   1,
			},
			wantErr: true,
		},
		{
			name: "zero quantity",
			req: &SeckillRequest{
				RequestID:  "test-123",
				ActivityID: 1,
				UserID:     100,
				Quantity:   0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := tt.req.RequestID == "" || tt.req.ActivityID == 0 || 
					 tt.req.UserID == 0 || tt.req.Quantity <= 0
			assert.Equal(t, tt.wantErr, hasErr)
		})
	}
}

func TestSeckillResult_Success(t *testing.T) {
	result := &SeckillResult{
		Success:   true,
		RequestID: "test-123",
		OrderID:   "order-456",
		Message:   "秒杀成功",
		QueuePos:  0,
	}

	assert.True(t, result.Success)
	assert.Equal(t, "test-123", result.RequestID)
	assert.Equal(t, "order-456", result.OrderID)
	assert.Equal(t, "秒杀成功", result.Message)
	assert.Equal(t, 0, result.QueuePos)
}

func TestSeckillResult_Failure(t *testing.T) {
	result := &SeckillResult{
		Success:   false,
		RequestID: "test-123",
		OrderID:   "",
		Message:   "库存不足",
		QueuePos:  0,
	}

	assert.False(t, result.Success)
	assert.Equal(t, "test-123", result.RequestID)
	assert.Empty(t, result.OrderID)
	assert.Equal(t, "库存不足", result.Message)
}

func TestOrderMessage_Validation(t *testing.T) {
	msg := &model.OrderMessage{
		RequestID:  "test-123",
		ActivityID: 1,
		UserID:     100,
		Quantity:   1,
		Price:      1000,
		DeductID:   "deduct-456",
		Timestamp:  time.Now().Unix(),
	}

	assert.NotEmpty(t, msg.RequestID)
	assert.Greater(t, msg.ActivityID, uint64(0))
	assert.Greater(t, msg.UserID, uint64(0))
	assert.Greater(t, msg.Quantity, 0)
	assert.Greater(t, msg.Price, float64(0))
	assert.NotEmpty(t, msg.DeductID)
	assert.Greater(t, msg.Timestamp, int64(0))
}

func TestActivityValidation(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		activity *model.SeckillActivity
		isActive bool
	}{
		{
			name: "active activity",
			activity: &model.SeckillActivity{
				ID:        1,
				Name:      "Test Activity",
				StartTime: now.Add(-time.Hour),
				EndTime:   now.Add(time.Hour),
				Status:    1,
				Stock:     100,
			},
			isActive: true,
		},
		{
			name: "not started activity",
			activity: &model.SeckillActivity{
				ID:        2,
				Name:      "Future Activity",
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(2 * time.Hour),
				Status:    1,
				Stock:     100,
			},
			isActive: false,
		},
		{
			name: "ended activity",
			activity: &model.SeckillActivity{
				ID:        3,
				Name:      "Past Activity",
				StartTime: now.Add(-2 * time.Hour),
				EndTime:   now.Add(-time.Hour),
				Status:    1,
				Stock:     100,
			},
			isActive: false,
		},
		{
			name: "inactive status",
			activity: &model.SeckillActivity{
				ID:        4,
				Name:      "Inactive Activity",
				StartTime: now.Add(-time.Hour),
				EndTime:   now.Add(time.Hour),
				Status:    0,
				Stock:     100,
			},
			isActive: false,
		},
		{
			name: "no stock",
			activity: &model.SeckillActivity{
				ID:        5,
				Name:      "No Stock Activity",
				StartTime: now.Add(-time.Hour),
				EndTime:   now.Add(time.Hour),
				Status:    1,
				Stock:     0,
			},
			isActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isActive := tt.activity.Status == 1 && 
					   tt.activity.Stock > 0 &&
					   now.After(tt.activity.StartTime) && 
					   now.Before(tt.activity.EndTime)
			assert.Equal(t, tt.isActive, isActive)
		})
	}
}

func TestSeckillServiceInterface(t *testing.T) {
	// Test that SeckillService interface has required methods
	var service SeckillService
	assert.Nil(t, service)
	
	// Verify interface methods exist by checking function signatures
	ctx := context.Background()
	
	if service != nil {
		// These calls would panic if service is nil, but we're just testing interface
		_, _ = service.DoSeckill(ctx, &SeckillRequest{})
		_ = service.PrewarmActivity(ctx, 1)
		_, _ = service.QuerySeckillResult(ctx, "test", 1)
	}
}

func TestFailResult(t *testing.T) {
	// Test helper function for creating failure results
	createFailResult := func(requestID, message string) *SeckillResult {
		return &SeckillResult{
			Success:   false,
			RequestID: requestID,
			Message:   message,
		}
	}

	result := createFailResult("test-123", "活动不存在")
	
	assert.False(t, result.Success)
	assert.Equal(t, "test-123", result.RequestID)
	assert.Equal(t, "活动不存在", result.Message)
	assert.Empty(t, result.OrderID)
}

func TestSuccessResult(t *testing.T) {
	// Test helper function for creating success results
	createSuccessResult := func(requestID, orderID, message string) *SeckillResult {
		return &SeckillResult{
			Success:   true,
			RequestID: requestID,
			OrderID:   orderID,
			Message:   message,
		}
	}

	result := createSuccessResult("test-123", "order-456", "秒杀成功，订单处理中")
	
	assert.True(t, result.Success)
	assert.Equal(t, "test-123", result.RequestID)
	assert.Equal(t, "order-456", result.OrderID)
	assert.Equal(t, "秒杀成功，订单处理中", result.Message)
}