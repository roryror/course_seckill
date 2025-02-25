package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client
var ctx = context.Background()

func InitRedis() {
	config := GlobalConfig.Redis
	
	redisClient = redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		PoolTimeout:  config.PoolTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})
	
	redisClient.Del(ctx, "course:stock")    
	redisClient.Del(ctx, "order:requests") 
	// coures stock cache
	warmUp()
	// order request cache, record the previous order request in 5 seconds
	redisClient.SAdd(ctx, "order:requests", "u:c")
	redisClient.Expire(ctx, "order:requests", time.Second)
	// order status cache, for front end polling
	redisClient.HSet(ctx, "order:status", "u:c", 0)
	fmt.Println("Redis initialized")
}

// warm up the redis cache
// preload the stock of all courses to redis to avoid cache miss
func warmUp() {
	var courses []Course
	db.Find(&courses)
	for _, course := range courses {
		redisClient.HSet(ctx, "course:stock", course.ID, course.Stock)
	}
}

// cache aside pattern
// check if the order request is in the redis cache
// if not, add the request to the redis cache
// if yes, return 0
// if the stock is 0, return -1
// if the stock is not 0, decrease the stock and return 1
func cacheAside(uid int, cid int) error {
	luaScript := redis.NewScript(`
		local uid = ARGV[1] 
		local cid = ARGV[2]
		local requestKey = ARGV[1] .. ":" .. ARGV[2]
		
		if redis.call("SISMEMBER", "order:requests", requestKey) == 1 then
			return 0
		end

		redis.call("SADD", "order:requests", requestKey)
		redis.call("EXPIRE", "order:requests", 5) 
		
		local stock = redis.call("HGET", "course:stock", cid)
		if not stock then
			return -1
		end
		
		stock = tonumber(stock)
		if stock <= 0 then
			return -1
		end
		
		local new_stock = redis.call("HINCRBY", "course:stock", cid, -1)
		if new_stock >= 0 then
			return 1
		else
			redis.call("HINCRBY", "course:stock", cid, 1)
			return -1
		end
	`)

	result, _ := luaScript.Run(ctx, redisClient, []string{}, strconv.Itoa(uid), strconv.Itoa(cid)).Result()
	intResult := result.(int64)
	switch intResult {
	case 1:
		select {
		// send the order request to the channel buffer, 
		// minimize the delay of http request
		case messageChan <- orderMessage{UserID: uid, CourseID: cid}:
			// messageChan is a channel buffer set up in kafka.go
			fmt.Printf("order message sent -> %d:%d\n", uid, cid)
		default:
			rollbackRedis(cid)
			return errors.New("system busy")
		}
		return nil
	case 0:
		return errors.New("repeat order")
	case -1:
		return errors.New("out of stock")
	default:
		return fmt.Errorf("unexpected result: %v", result)
	}
}

// rollback the stock of the course when later operation fails
func rollbackRedis(cid int) {
	redisClient.HIncrBy(ctx, "course:stock", strconv.Itoa(cid), 1)
}

// change the status of the order in the redis cache for front end polling
func changeOrderStatus(uid int, cid int, status int) {
	redisClient.HSet(ctx, "order:status", fmt.Sprintf("%d:%d", uid, cid), status)
}

func getRedisHash() map[string]string {
	hash := redisClient.HGetAll(ctx, "course:stock").Val()
	return hash
}

func CloseRedis() {
	redisClient.Close()
}