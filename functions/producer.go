package functions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/godror/godror"
	"github.com/rabbitmq/amqp091-go"
)

type Pessoa struct {
	CD_DEPARA_INTEGRA string `json:"cd_depara_integra"`
}

func Producer() {
	// 1. Conectar no Oracle
	db, err := sql.Open("godror", "INTEG_LAB/integ25#2025@2311db.cloudmv.com.br:1522/tst12311.db2311.mv2311vcn.oraclevcn.com")
	if err != nil {
		log.Fatal(err.Error() + " - Erro ao conectar no Oracle")
	}
	defer db.Close()

	// 2. Conectar no RabbitMQ
	conn, err := amqp091.Dial("amqp://admin:admin123@rabbitmq:5672/")
	if err != nil {
		for i := 0; i < 5; i++ {
			conn, err = amqp091.Dial("amqp://admin:admin123@rabbitmq:5672/")
			if err == nil {
				log.Println("Conectou no RabbitMQ ✅")

				break
			}
			log.Println("Tentativa falhou, tentando de novo...")
			time.Sleep(3 * time.Second)
		}
	}
	defer conn.Close()

	log.Println("Conectou no RabbitMQ com sucesso ✅")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	// 3. Buscar registros no banco
	rows, err := db.Query("select CD_DEPARA_INTEGRA from mvintegra.depara where cd_sistema_integra like '%FLEURY%'")
	//rows, err := db.Query("select CD_DEPARA_INTEGRA from mvintegra.depara where cd_sistema_integra like '%OPMENEXO%' order by CD_DEPARA_INTEGRA FETCH FIRST 10 ROWS ONLY")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Pessoa
		if err := rows.Scan(&p.CD_DEPARA_INTEGRA); err != nil {
			log.Println("Erro ao ler linha:", err)
			continue
		}

		// Serializar em JSON
		body, _ := json.Marshal(p)

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
			log.Println("Erro ao publicar mensagem:", err)
		} else {
			fmt.Printf("Mensagem enviada: %+v\n", p)
		}
	}
}

func ProdutosFleury() gin.HandlerFunc {
	fn := func(c *gin.Context) {
		produto := c.Param("produto")
		log.Println("Produto:", produto)

		mensagem, err := acessoFleury(produto)
		if err != nil {
			log.Println("Erro ao acessar Fleury:", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"sucesso": false,
				"message": "Erro ao acessar Fleury " + err.Error(),
			})
			return

		}
		if mensagem.IdProduto == "" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"sucesso": false,
				"message": "Produto não encontrado no Fleury",
			})
			return
		}
		//log.Println(mensagem)
		log.Println("IdProduto:", mensagem.IdProduto)
		log.Println("TextoInstrucao:", mensagem.TextoInstrucao)

		err = procedureMV(mensagem)
		if err != nil {
			log.Println("Erro ao executar procedure:", err)
			mensagemErro := ""
			if err.Error() == "Dados não encontrados" {
				mensagemErro = "Erro ao executar procedure: Produto não encontrado no MV"
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"sucesso": false,
				"message": mensagemErro,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"sucesso": true,
			"message": "Produto processado com sucesso",
		})
	}
	return gin.HandlerFunc(fn)
}
