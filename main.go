package main

import (
	"bytes"
	"fmt"
	"io"
	"main_gin_go/sqlchecker"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
)

type UserCredentials struct {
	User string `json:"user"`
	Pwd  string `json:"pwd"`
}
type FormData struct {
	User string `form:"user"`
	List string `form:"list"` // 注意：这里假设list是字符串数组，根据实际情况调整类型
}

func testFormDataHandler(c *gin.Context) {
	var formData FormData
	if err := c.ShouldBind(&formData); err == nil {
		fmt.Printf("User: %s, List: %v\n", formData.User, formData.List)
		c.JSON(http.StatusOK, gin.H{
			"user": formData.User,
			"list": formData.List,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

type RequestBody struct {
	List    []int       `json:"list"`
	StrList []string    `json:"strList"`
	Name    string      `json:"name"`
	Obj     InnerObject `json:"obj"`
}

type InnerObject struct {
	Age int   `json:"age"`
	Arr []int `json:"arr"`
}

func main() {
	router := gin.Default()

	// 注册中间件  防护示例
	router.Use(sqlchecker.SQLInjectionChecker)

	// 在此路由组内定义的路由将不受全局中间件的影响
	router.GET("/test-get1", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "GET request successful without SQL check sql"})
	})
	router.POST("/test-json", func(c *gin.Context) {
		var requestBody RequestBody
		if err := c.ShouldBindJSON(&requestBody); err == nil {
			fmt.Printf("Received Data: %+v\n", requestBody)
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data":   requestBody,
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
	})
	// 示例路由
	router.GET("/test-get", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{"message": "GET request successful"})
	})
	router.GET("/get/login", func(c *gin.Context) {

		user := c.Query("user")
		pwd := c.Query("pwd")
		fmt.Printf("Received user: %s, password: %s\n", user, pwd)

		c.JSON(http.StatusOK, gin.H{"message": "Received GET request with JSON body", "data": pwd})
	})

	router.POST("/test-form", testFormDataHandler)
	router.POST("/test-post", func(c *gin.Context) {
		// 备份原始请求体

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		defer c.Request.Body.Close()                         // 关闭备份的body，避免泄露资源
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body)) // 重置请求体以便后续使用

		// 打印原始请求体中的JSON数据
		fmt.Println("Raw JSON Request Body:", string(body))

		var creds UserCredentials

		if err := c.ShouldBindJSON(&creds); err != nil {
			fmt.Println("error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		val := reflect.ValueOf(creds)
		fmt.Printf("User via reflection: %s, Password via reflection: %s\n", val.FieldByName("User").String(), val.FieldByName("Pwd").String())

		c.JSON(http.StatusOK, gin.H{"message": "POST request successful with user:", "user": creds.User})

	})

	router.Run(":50269")
}
