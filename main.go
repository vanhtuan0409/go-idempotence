package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/redsync.v1"
)

const (
	IdempotenceHeader = "Idempotence-Key"
	KeyExpireTime     = 60 * 60 * 24 // 1day
	LockExpireTime    = 10 * time.Second
)

var (
	CurrentBidValue = 0
	rPool           *redis.Pool
	rSync           *redsync.Redsync
)

type (
	BidRequest struct {
		Value int `json:"value"`
	}

	BidReponse struct {
		Ok      bool   `json:"ok"`
		Message string `json:"message"`
	}
)

// Handlers
func postBidHandler(c echo.Context) error {
	key := c.Request().Header.Get(IdempotenceHeader)
	if key == "" {
		return c.JSON(http.StatusUnauthorized, &BidReponse{
			Ok:      false,
			Message: "Require Idempotence Key",
		})
	}

	return runWithLock(fmt.Sprintf("lock:%s", key), func() error {
		request := new(BidRequest)
		if err := c.Bind(request); err != nil {
			return c.JSON(http.StatusBadRequest, &BidReponse{
				Ok:      false,
				Message: fmt.Sprintf("Invalid request. ERR: %v", err),
			})
		}

		existingBid, err := redis.Int(rPool.Get().Do("GET", key))
		if err != nil && err != redis.ErrNil {
			return c.JSON(http.StatusUnauthorized, &BidReponse{
				Ok:      false,
				Message: fmt.Sprintf("ERR: %v", err),
			})
		} else if err == nil {
			return c.JSON(http.StatusOK, &BidReponse{
				Ok:      false,
				Message: fmt.Sprintf("Bid with value %d (cached)", existingBid),
			})
		}

		if request.Value <= CurrentBidValue {
			return c.JSON(http.StatusBadRequest, &BidReponse{
				Ok:      false,
				Message: "Bidding value must be greater than current value",
			})
		}

		// Simulate long running process
		time.Sleep(3 * time.Second)

		_, err = rPool.Get().Do("SET", key, request.Value, "EX", KeyExpireTime)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, &BidReponse{
				Ok:      false,
				Message: fmt.Sprintf("Cannot save value. ERR: %v", err),
			})
		}

		CurrentBidValue = request.Value

		return c.JSON(http.StatusOK, &BidReponse{
			Ok:      true,
			Message: fmt.Sprintf("Bid with value %d (saved)", CurrentBidValue),
		})
	})
}

func getBidHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"bid": CurrentBidValue,
	})
}

// Main
func main() {
	rPool = newRedisPool(":6379")
	defer rPool.Close()

	rSync = redsync.New([]redsync.Pool{rPool})

	e := echo.New()
	e.Logger.SetLevel(1)
	e.Use(middleware.Logger())
	e.POST("/bid", postBidHandler)
	e.GET("/bid", getBidHandler)
	e.Start(":8080")
}

// Utility function
func newRedisPool(address string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 60 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", address)
		},
	}
}

func runWithLock(resource string, f func() error) error {
	lock := rSync.NewMutex(resource, redsync.SetExpiry(LockExpireTime))
	if err := lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()
	return f()
}
