package sqlchecker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

func checkSQLInjection(input string) bool {
	sqlConstructs := []string{
		`(?i)\s*union\s+select\s*`, `(?i)\bselect\b`, `(?i)\bupdate\b`,
		`(?i)\bdelete\b`, `(?i)\binsert\b`, `(?i)\bcreate\b`, `(?i)\bdrop\b`, `(?i)\bwhere\b`, `(?i)\bfrom\b`, `(?i)\bset\b`, `(?i)\btable\b`, `(?i)\bdatabase\b`, `(?i)\binto\b`, `(?i)\bvalues\b`, `(?i)\border\b`, `(?i)\bby\b`, `(?i)\blimit\b`, `(?i)\basc\b`, `(?i)\bdesc\b`, `(?i)\bjoin\b`, `(?i)\binner\b`, `(?i)\bleft\b`, `(?i)\bright\b`, `(?i)\bfull\b`, `(?i)\bouter\b`, `(?i)\bgroup\b`, `(?i)\bhaving\b`, `(?i)\bcase\b`, `(?i)\bwhen\b`, `(?i)\bthen\b`, `(?i)\belse\b`, `(?i)\bend\b`, `(?i)\blike\b`, `(?i)\bnot\b`, `(?i)\band\b`, `(?i)\bor\b`, `(?i)\bunion\b`,
		`(?i)\balter\b`, `(?i)\btruncate\b`, `(?i)\bexec\b`, `(?i)\bdeclare\b`,
		`\%`, `\bunion\s+select\b|\bUNION\s+ALL\s+SELECT\b`, `(?i)(\s*=\s*|\s*!\s*=\s*|\s*<>\s*|\s*>\s*|\s*<\s*|\s*>=\s*|\s*<=\s*)`, `(?i)[#%&\(\)=+\n\r\t*]`,
	}
	// 百分号不需要不区分大小写

	passKeys := []string{
		`(?i)^select$`, `(?i)^update$`, `(?i)^insert$`, `(?i)^delete$`, `(?i)^create$`, `(?i)^drop$`, `(?i)^alter$`, `(?i)^truncate$`,
		`(?i)^exec$`, `(?i)^declare$`, `(?i)^like$`, `(?i)^and$`, `(?i)^or$`, `(?i)^not$`, `(?i)^where$`, `(?i)^values$`,
		`(?i)^set$`, `(?i)^table$`, `(?i)^database$`, `(?i)^into$`, `(?i)^order$`, `(?i)^by$`, `(?i)^limit$`, `(?i)^asc$`,
		`(?i)^desc$`, `(?i)^join$`, `(?i)^inner$`, `(?i)^left$`, `(?i)^right$`, `(?i)^full$`, `(?i)^outer$`, `(?i)^group$`,
		`(?i)^having$`, `(?i)^case$`, `(?i)^when$`, `(?i)^then$`, `(?i)^else$`, `(?i)^end$`, `(?i)^union$`, `^[0-9]+%`,
	}
	for _, key := range passKeys {
		re := regexp.MustCompile(key)
		if re.MatchString(input) {
			fmt.Println("Passed", key, input)
			return false // 如果匹配到任何关键词，则返回false，表示不安全
		}
	}

	for _, construct := range sqlConstructs {
		re := regexp.MustCompile(construct)
		if re.MatchString(input) {
			return true
		}
	}

	return false
}
func replaceSemicolonWithEncoded(urlStr string) string {
	return strings.ReplaceAll(urlStr, ";", "%3B")
}
func SQLInjectionChecker(c *gin.Context) {
	//增加白名单，跳过检查的路径列表
	skipPaths := []string{"/test-get1"}

	isSkippedPath := false
	for _, path := range skipPaths {
		if strings.HasPrefix(c.Request.URL.Path, path) {
			isSkippedPath = true
			break
		}
	}
	if isSkippedPath {
		c.Next()
		return
	}

	requestMethod := c.Request.Method
	var params1 map[string][]string

	switch requestMethod {
	case http.MethodGet:
		params1 = handleGetRequest(c)
	case http.MethodPost:
		params1 = handlePostRequest(c)
	default:
		c.Next()
		return
	}

	if params1 != nil {
		checkParamsForSQLInjection(c, params1)
	}
}
func handleGetRequest(c *gin.Context) map[string][]string {
	originalURL := c.Request.URL.String()
	modifiedURL := replaceSemicolonWithEncoded(originalURL)
	parsedURL, err := url.Parse(modifiedURL)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to parse URL sql"})
		return nil
	}
	fmt.Println("http.MethodGet", parsedURL)
	return parsedURL.Query()
}

func handlePostRequest(c *gin.Context) map[string][]string {
	contentType := c.Request.Header.Get("Content-Type")
	isMultipartFormData := strings.HasPrefix(contentType, "multipart/form-data")
	isHasJsonType := strings.Contains(contentType, "application/json")
	isHasWww_form_urlencoded := strings.Contains(contentType, "application/x-www-form-urlencoded")
	if isMultipartFormData || isHasWww_form_urlencoded {
		return handleFormRequest(c)
	} else if isHasJsonType {
		return handleJSONRequest(c)
	} else {
		c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{"error": "Unsupported Content-Type"})
		return nil
	}

}
func handleJSONRequest(c *gin.Context) map[string][]string {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body in sql"})
		return nil
	}
	defer c.Request.Body.Close()
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var jsonData map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON data in sql"})
		return nil
	}

	params := make(map[string][]string)
	for key, value := range jsonData {
		var formattedValue string
		switch v := value.(type) {
		case float64: // 处理浮点数
			formattedValue = fmt.Sprintf("%.f", v)
		case int, int32, int64: // 处理整数类型
			formattedValue = fmt.Sprintf("%d", v)
		case string:
			formattedValue = v
		case nil:
			formattedValue = ""
		default:
			formattedValue = fmt.Sprintf("%v", value) // 对于其他类型，仍按原始方式处理
		}
		params[key] = []string{formattedValue}
	}
	return params
}

func handleFormRequest(c *gin.Context) map[string][]string {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil && err != http.ErrNotMultipart {
		if err := c.Request.ParseForm(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
			return nil
		}
	}
	return c.Request.Form
}
func checkParamsForSQLInjection(c *gin.Context, params map[string][]string) {

	for key, values := range params {
		for _, value := range values {

			if checkSQLInjection(value) {
				log.Println("检测到SQL注入攻击", "key:", key, "value:", value)
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"code":    401,
					"message": "您的请求包含非法字符或格式不正确，请检查后重新提交。",
					"data":    nil,
				})
				return
			}
		}
	}
}
