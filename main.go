package main

import (
	"io"
	"log"
	"os"
	"time"

	_ "github.com/godror/godror"
)

var f, _ = os.OpenFile("logs/log_"+time.Now().Format("01-02-2006")+".log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

var PORT = "2000"

func main() {
	//setupVars()

	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

	setupControle()

	defer f.Close()

	r := setupRouter()
	r.Run(":" + PORT)
}

// func setupVars() {
// 	err := godotenv.Load(".env")
// 	if err != nil {
// 		//log.Println("Adicionar o arquivo de configuração .env \n", err.Error())
// 		utils.LogMonitor(utils.ErroGeral, "general", "Adicionar o arquivo de configuração .env: "+err.Error())
// 		panic(err)
// 	}

// 	PORT = os.Getenv("PORT")
// 	//vars.TablespaceName = os.Getenv("TABLESPACENAME")

// }
