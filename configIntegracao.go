package main

import (
	"fmt"
	"integracao-hes-fleury/functions"
	"integracao-hes-fleury/utils"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func setupControle() {

	//go producer()

	go functions.Consumer()

	//functions.Producer()

	go runScheduler(time.Tuesday, "15:00:00")

}

func runScheduler(weekday time.Weekday, tm string) {
	loc, err := time.LoadLocation("America/Sao_Paulo") // use "America/Sao_Paulo" em vez de "Brazil/East"
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// horário alvo (ex: "09:00:00")
	tm_, _ := time.ParseInLocation("15:04:05", tm, loc)

	// calcula o próximo disparo
	next := nextWeekdayTime(time.Now().In(loc), weekday, tm_, loc)

	duration := next.Sub(time.Now().In(loc))
	timer := time.NewTimer(duration)

	go func() {
		for {
			<-timer.C
			fmt.Println("Agenda executada em:", time.Now().In(loc))

			// sua função aqui
			functions.Producer()

			// calcula próxima execução (sempre +7 dias)
			next = nextWeekdayTime(time.Now().In(loc), weekday, tm_, loc)
			timer.Reset(next.Sub(time.Now().In(loc)))
		}
	}()

	select {} // trava a goroutine principal
}

// calcula a próxima data no dia da semana e hora desejada
func nextWeekdayTime(now time.Time, weekday time.Weekday, targetTime time.Time, loc *time.Location) time.Time {
	// cria a data de "hoje" com o horário alvo
	next := time.Date(now.Year(), now.Month(), now.Day(),
		targetTime.Hour(), targetTime.Minute(), targetTime.Second(), 0, loc)

	// quantos dias faltam até o próximo weekday
	offset := (int(weekday) - int(now.Weekday()) + 7) % 7

	// se já passou ou é exatamente agora → próxima semana
	if offset == 0 && !now.Before(next) {
		offset = 7
	}

	return next.AddDate(0, 0, offset)
}

func AuthMiddleware(expectedID, expectedSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetHeader("client_id")
		clientSecret := c.GetHeader("client_secret")

		if clientID != expectedID || clientSecret != expectedSecret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"sucesso": false,
				"message": "unauthorized",
			})
			return
		}

		c.Next()
	}
}

func setupRouter() *gin.Engine {

	r := gin.Default()
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	clientID := os.Getenv("client_id_ws")
	clientSecret := os.Getenv("client_secret_ws")

	r.GET("/check-status", AuthMiddleware(clientID, clientSecret), checkStatus())
	r.POST("/produtos-fleury/:produto", AuthMiddleware(clientID, clientSecret), functions.ProdutosFleury())
	r.NoRoute(AuthMiddleware(clientID, clientSecret), NotFound())

	return r
}

func NotFound() gin.HandlerFunc {
	fn := func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"sucesso": false,
			"message": "not found",
		})
		utils.LogMonitor(utils.Debug, "Error", "404 page not found. O recurso solicitado não foi encontrado. Request: "+c.Request.Method)

	}
	return gin.HandlerFunc(fn)
}

func checkStatus() gin.HandlerFunc {
	fn := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"sucesso": true,
			"message": "Aplicação online!",
		})
	}
	return gin.HandlerFunc(fn)
}
