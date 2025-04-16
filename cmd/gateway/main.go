package main

import (
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()

	r.Any("/inventory/*action", proxyInventory)

	r.Any("/orders/*action", proxyOrders)

	r.Run(":8080")
}

func proxyInventory(c *gin.Context) {
	targetURL := "http://localhost:8081" + c.Param("action")
	proxyRequest(c, targetURL)
}

func proxyOrders(c *gin.Context) {
	targetURL := "http://localhost:8082" + c.Param("action")
	proxyRequest(c, targetURL)
}

func proxyRequest(c *gin.Context, targetURL string) {
	client := &http.Client{}

	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header = c.Request.Header

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
