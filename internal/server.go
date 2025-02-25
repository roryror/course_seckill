package internal

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func RunServer() {
	config := GlobalConfig.Server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	}))
	r.Use(gin.Recovery())
	r.LoadHTMLGlob("web/*")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	r.GET("/api/courses", tableCourses)
	r.GET("/api/orders", tableOrders)
	r.GET("/api/redis", tableRedis)
	r.GET("/api/seckill/:cid/:uid", seckillCourse)
	r.GET("/api/checkOrderStatus/:cids/:uid", checkOrderStatus)
	server := &http.Server{
		Addr:           config.Port,
		Handler:        r,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
		IdleTimeout:    config.IdleTimeout,
	}
	fmt.Println("Server started on http://localhost:8080\nplease wait for reader activation...")
	server.ListenAndServe()
}

func tableCourses(c *gin.Context) {
	courses := getAllCourses()
	c.JSON(http.StatusOK, courses)
}

func tableOrders(c *gin.Context) {
	orders := getAllOrders()
	c.JSON(http.StatusOK, orders)
}

func tableRedis(c *gin.Context) {
	redisHash := getRedisHash()
	c.JSON(http.StatusOK, redisHash)
}

func seckillCourse(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))
	uid, _ := strconv.Atoi(c.Param("uid"))
	var test bool = true
	if test {
		if cid == 0 {
			cid = rand.Intn(6) + 1
		}
		if uid == 0 {
			uid = rand.Intn(1000) + 1
		}
	}

	err := cacheAside(uid, cid)
	// give quick response to the front end
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "seckill failed", "error": err.Error(), "courseID": cid})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "seckill order pending", "error": nil, "courseID": cid})
}

func checkOrderStatus(c *gin.Context) {
    cids := c.Param("cids")
    uid, _ := strconv.Atoi(c.Param("uid"))
    cidList := strings.Split(cids, ":")
	
    statusMap := make(map[string]string)

    for _, cidStr := range cidList {
        cid, _ := strconv.Atoi(cidStr)
        
        status, err := redisClient.HGet(ctx, "order:status", fmt.Sprintf("%d:%d", uid, cid)).Result()
        if err != nil {
            statusMap[cidStr] = "0"
        } else {
            statusMap[cidStr] = status
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "status": statusMap,
    })
}