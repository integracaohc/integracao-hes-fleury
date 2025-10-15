package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"integracao-hes-fleury/db"
	"integracao-hes-fleury/utils"
	"os"
	"time"

	_ "github.com/godror/godror"
	"github.com/rabbitmq/amqp091-go"
)

type DEPARA_INTEGRA struct {
	CD_DEPARA_MV      string `json:"cd_depara_mv"`
	CD_DEPARA_INTEGRA string `json:"cd_depara_integra"`
	REPETICOES_FILA   int    `json:"repeticoes_fila"`
}

type ERROS_INTEGRA struct {
	CD_DEPARA_MV      string `json:"cd_depara_mv"`
	CD_DEPARA_INTEGRA string `json:"cd_depara_integra"`
	ERROS             string `json:"erros"`
}

func Producer() {
	label := "Producer"
	// 1. Conectar no Oracle
	//db, err := sql.Open("godror", "INTEG_LAB/integ25#2025@2311db.cloudmv.com.br:1522/tst12311.db2311.mv2311vcn.oraclevcn.com")
	db, err := db.InitDB()
	if err != nil {
		//log.Println("Erro ao conectar no Oracle: " + err.Error())
		utils.LogMonitor(utils.ErroBanco, label, "Erro ao conectar no Oracle: "+err.Error())
		return
	}
	defer db.Close()

	// 2. Conectar no RabbitMQ
	conn, err := amqp091.Dial("amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@rabbitmq:" + os.Getenv("RABBITMQ_PORT") + "/")
	if err != nil {
		for i := 0; i < 5; i++ {
			conn, err = amqp091.Dial("amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@rabbitmq:" + os.Getenv("RABBITMQ_PORT") + "/")
			if err == nil {
				//log.Println("Conectou no RabbitMQ ✅")
				utils.LogMonitor(utils.Debug, label, "Conectou no RabbitMQ ✅")

				break
			}
			//log.Println("Tentativa falhou, tentando de novo...")
			utils.LogMonitor(utils.Debug, label, "Tentativa falhou, tentando de novo...")
			time.Sleep(3 * time.Second)
		}
	}
	defer conn.Close()

	//log.Println("Conectou no RabbitMQ com sucesso ✅")
	utils.LogMonitor(utils.Debug, label, "Conectou no RabbitMQ com sucesso ✅")

	ch, err := conn.Channel()
	if err != nil {
		//log.Println("Erro ao conectar no RabbitMQ: " + err.Error())
		utils.LogMonitor(utils.ErroConexao, label, "Erro ao conectar no RabbitMQ: "+err.Error())
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"fleury_instrucoes_mv", // nome
		true,                   // durable
		false,                  // auto-delete
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // args
	)
	if err != nil {
		//log.Println("Erro ao declarar fila: " + err.Error())
		utils.LogMonitor(utils.ErroConexao, label, "Erro ao declarar fila: "+err.Error())
		return
	}

	// 3. Buscar registros no banco
	rows, err := db.Query("select CD_DEPARA_MV, CD_DEPARA_INTEGRA from mvintegra.depara where cd_sistema_integra like '%FLEURY%'")
	//rows, err := db.Query("select CD_DEPARA_INTEGRA from mvintegra.depara where cd_sistema_integra like '%OPMENEXO%' order by CD_DEPARA_INTEGRA FETCH FIRST 10 ROWS ONLY")
	if err != nil {
		//log.Println("Erro ao buscar registros no banco: " + err.Error())
		utils.LogMonitor(utils.ErroBanco, label, "Erro ao buscar registros no banco: "+err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var depara DEPARA_INTEGRA
		if err := rows.Scan(&depara.CD_DEPARA_MV, &depara.CD_DEPARA_INTEGRA); err != nil {
			//log.Println("Erro ao ler linha:", err)
			utils.LogMonitor(utils.ErroGeral, label, "Erro ao ler linha: "+err.Error())
			continue
		}

		depara.REPETICOES_FILA = 0

		// Serializar em JSON
		body, _ := json.Marshal(depara)

		// Publicar na fila
		err = ch.PublishWithContext(
			context.Background(),
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp091.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		if err != nil {
			//log.Println("Erro ao publicar mensagem:", err)
			utils.LogMonitor(utils.ErroConexao, label, "Erro ao publicar mensagem: "+err.Error())
		} else {
			fmt.Printf("Mensagem enviada: %+v\n", depara)
			utils.LogMonitor(utils.Debug, label, "Mensagem enviada: "+depara.CD_DEPARA_INTEGRA)
		}
	}
}
