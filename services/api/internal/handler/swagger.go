package handler

import (
	"embed"
	"io/fs"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var swaggerFS embed.FS

// SwaggerHandler 返回 Swagger UI + OpenAPI 文档处理器
func SwaggerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("any")
		if path == "" || path == "/" || path == "/index.html" {
			serveSwaggerUI(c)
			return
		}
		if path == "/openapi.yaml" || path == "/openapi.json" {
			data, err := fs.ReadFile(swaggerFS, "openapi.yaml")
			if err != nil {
				c.String(500, "openapi.yaml read failed: %s", err.Error())
				return
			}
			c.Header("Content-Type", "application/yaml; charset=utf-8")
			c.Data(200, "application/yaml; charset=utf-8", data)
			return
		}
		c.String(404, "not found")
	}
}

func serveSwaggerUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, swaggerUIHTML)
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>MediaHub API - Swagger UI</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
<style>body{margin:0;}</style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
window.onload = () => {
  window.ui = SwaggerUIBundle({
    url: "/swagger/openapi.yaml",
    dom_id: "#swagger-ui",
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout"
  });
};
</script>
</body>
</html>`
