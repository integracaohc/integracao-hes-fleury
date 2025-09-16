package functions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/godror/godror"
	"github.com/rabbitmq/amqp091-go"
)

type CD_DEPARA struct {
	CD_DEPARA_INTEGRA string `json:"cd_depara_integra"`
}

type INSTRUCAO_FLEURY_MV struct {
	IdProduto      string `json:"idProduto"`
	TextoInstrucao string `json:"textoInstrucao"`
}

func Consumer() {
	// _, err := acessoFleury("100")
	// if err != nil {
	// 	log.Println("Erro ao acessar Fleury:", err)
	// 	return
	// }
	conn, err := amqp091.Dial("amqp://admin:admin123@rabbitmq:5672/")
	if err != nil {
		for i := 0; i < 5; i++ {
			conn, err = amqp091.Dial("amqp://admin:admin123@rabbitmq:5672/")
			if err == nil {
				log.Println("Conectou no RabbitMQ")

				break
			}
			log.Println("Tentativa falhou, tentando de novo...")
			time.Sleep(3 * time.Second)
		}
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"fleury_instrucoes_mv",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusivo
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var cdDepara CD_DEPARA
			if err := json.Unmarshal(d.Body, &cdDepara); err != nil {
				log.Println("Erro ao decodificar:", err)
				continue
			}
			fmt.Printf("Recebido: %+v\n", cdDepara)
			mensagem, err := acessoFleury(cdDepara.CD_DEPARA_INTEGRA)
			if err != nil {
				log.Println("Erro ao acessar Fleury:", err)
				reenviaParaFila(mensagem, ch, q)
				continue
			}
			if mensagem.IdProduto == "" {
				continue
			}
			//log.Println(mensagem)
			log.Println("IdProduto:", mensagem.IdProduto)
			log.Println("TextoInstrucao:", mensagem.TextoInstrucao)

			err = procedureMV(mensagem)
			if err != nil {
				log.Println("Erro ao executar procedure:", err)
				continue
			}
		}
	}()
	log.Println("Consumidor aguardando mensagens...")
	fmt.Println("Consumidor aguardando mensagens...")
	<-forever

}

func reenviaParaFila(mensagem INSTRUCAO_FLEURY_MV, ch *amqp091.Channel, q amqp091.Queue) {
	// Serializar em JSON
	body, _ := json.Marshal(mensagem)

	// Publicar na fila
	err := ch.PublishWithContext(
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
		fmt.Printf("Mensagem enviada: %+v\n", mensagem)
	}
}

type GrantCode struct {
	RedirectUri string `json:"redirect_uri"`
}

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type InstrucoesGerais struct {
	IdInstrucaoGeral string `json:"idInstrucaoGeral"`
	TextoInstrucao   string `json:"textoInstrucao"`
}

type Mensagem struct {
	IdProduto        string             `json:"idProduto"`
	InstrucoesGerais []InstrucoesGerais `json:"instrucoesGerais"`
}

func acessoFleury(produto string) (INSTRUCAO_FLEURY_MV, error) {
	url := "https://api-hml.grupofleury.com.br/oauth/grant-code"
	method := "POST"

	payload := strings.NewReader(`{
	"client_id": "0119221f-1c0e-3189-8314-4515a9c95820",
	"redirect_uri": "http://localhost",
	"extra_info":{
		"idCliente":1234,
		"tipoCliente":"CD"
	}
  }
  `)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	defer res.Body.Close()

	// Lê a resposta da outra API
	grantCode := GrantCode{}
	respJSON, _ := io.ReadAll(res.Body)
	json.Unmarshal(respJSON, &grantCode)

	code := strings.Replace(grantCode.RedirectUri, "http://localhost/?code=", "", 1)

	url = "https://api-hml.grupofleury.com.br/oauth/access-token"
	method = "POST"

	payload = strings.NewReader("grant_type=authorization_code&code=" + code)

	client = &http.Client{}
	req, err = http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("client_id", "0119221f-1c0e-3189-8314-4515a9c95820")
	req.Header.Add("Authorization", "Basic MDExOTIyMWYtMWMwZS0zMTg5LTgzMTQtNDUxNWE5Yzk1ODIwOjZiNzdjNmJmLWVhY2MtM2JlZS1iOGY4LTQyMWUzOTdmYzY0ZA==")

	res, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	defer res.Body.Close()

	accessToken := AccessToken{}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	fmt.Println(string(body))
	json.Unmarshal(body, &accessToken)

	url = "https://api-hml.grupofleury.com.br/instrucoes-gerais-hospitais/v1/produtos?produtos=" + produto
	method = "GET"

	client = &http.Client{}
	req, err = http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	req.Header.Add("client_id", "0119221f-1c0e-3189-8314-4515a9c95820")
	req.Header.Add("access_token", accessToken.AccessToken)

	res, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return INSTRUCAO_FLEURY_MV{}, err
	}
	fmt.Println(string(body))
	log.Println(string(body))

	// mensagem := Mensagem{}
	// json.Unmarshal(body, &mensagem)
	// log.Println(mensagem)
	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return INSTRUCAO_FLEURY_MV{}, err
	}
	if len(result) == 0 {
		return INSTRUCAO_FLEURY_MV{}, nil
	}
	//fmt.Println("idProduto:", result[0]["idProduto"])
	instrucoes := result[0]["instrucoesGerais"].([]interface{})
	var instrucoesMV INSTRUCAO_FLEURY_MV
	instrucoesMV.IdProduto = produto
	for _, i := range instrucoes {

		instrucao := i.(map[string]interface{})
		//fmt.Println(instrucao["idInstrucaoGeral"])
		if instrucao["idInstrucaoGeral"] != float64(6) {
			continue
		}
		instrucoesMV.TextoInstrucao = instrucao["textoInstrucao"].(string)
		//fmt.Println("idInstrucaoGeral:", instrucao["idInstrucaoGeral"])
		//fmt.Println("textoInstrucao:", instrucao["textoInstrucao"])
	}
	log.Println(instrucoesMV)
	return instrucoesMV, nil
}

func procedureMV(instrucao INSTRUCAO_FLEURY_MV) error {
	db, err := sql.Open("godror", "INTEG_LAB/integ25#2025@2311db.cloudmv.com.br:1522/tst12311.db2311.mv2311vcn.oraclevcn.com")
	if err != nil {
		log.Println(err.Error() + " - Erro ao conectar no Oracle")
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("begin mvintegra.HC_IH_999(:1, :2); end;")
	if err != nil {
		log.Println(err.Error() + " - Erro ao preparar a procedure")
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(instrucao.IdProduto, instrucao.TextoInstrucao)
	if err != nil {
		log.Println(err.Error() + " - Erro ao executar a procedure")
		return err
	}
	log.Println("Procedure executada com sucesso. IdProduto: " + instrucao.IdProduto + " - TextoInstrucao: " + instrucao.TextoInstrucao)
	return nil
}
