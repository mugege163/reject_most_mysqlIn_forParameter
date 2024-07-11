package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
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

func initGo() {
	arr := []int{1, 2, 3}
	log.Printf("arr: %v", arr)
	log.Println(arr)
	var iface interface{} = []string{"a", "b", "c"}
	slice := []string{}
	if ifacCopy, ok := iface.([]string); ok {
		slice = append(slice, ifacCopy...)
	}
	log.Printf("slice: %v", slice)
	//cap 使用
	ifaceValue := iface.([]string) // 先进行类型断言，将iface转换为具体的类型
	if ifaceValue != nil {         // 检查断言后的值是否为空
		log.Println("ifaceValue的容量长度为：", cap(ifaceValue)) // 只有在断言成功后，才能使用cap函数
	} else {
		log.Println("iface is nil") // 接口变量为空
	}
	arr_ := make([]int, len(arr))
	copy(arr_, arr)

	log.Printf("arr_ copy的内容为: %v,", arr_)

	//创建复数
	var complexNumber complex64 = 1 + 2i
	conmpInt := complex(float64(2), float64(3))
	log.Printf("获取值拿到复数：complexNumber: %v, conmpInt: %v", complexNumber, conmpInt)

}

func main() {
	router := gin.Default()

	// 注册中间件
	router.Use(sqlchecker.SQLInjectionChecker)
	// withoutSQLCheckGroup := router.Group("/aab", func(c *gin.Context) {
	// 	c.Next() // 确保继续执行组内的其他处理器
	// })
	// withoutSQLCheckGroup.Use(gin.Logger()) // 示例：可以在这里添加针对这个组的局部中间件
	// withoutSQLCheckGroup.Use(gin.Recovery())

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
		// 由于标准的 GET 请求不应该有请求体，Gin 默认也不会读取 GET 请求的请求体。
		// 为了演示目的，我们尝试手动读取请求体，这在实际应用中可能不适用或需要特别配置。
		user := c.Query("user")
		pwd := c.Query("pwd")
		fmt.Printf("Received user: %s, password: %s\n", user, pwd)

		// 这里处理 user 和 pwd
		// ...

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

		// 打印UserCredentials的类型信息
		fmt.Printf("Type of creds: %s\n", reflect.TypeOf(creds))

		// 打印接收到的用户名和密码
		fmt.Printf("Received user: %s, password: %s\n", creds.User, creds.Pwd)

		// 同时，你也可以直接使用反射来访问和打印结构体字段的值
		val := reflect.ValueOf(creds)
		fmt.Printf("User via reflection: %s, Password via reflection: %s\n", val.FieldByName("User").String(), val.FieldByName("Pwd").String())

		c.JSON(http.StatusOK, gin.H{"message": "POST request successful with user:", "user": creds.User})

	})
	initGo()
	router.Run(":50269")
}
