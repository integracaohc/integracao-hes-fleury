package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"integracao-hes-fleury/db"
	"integracao-hes-fleury/mail"
	"integracao-hes-fleury/utils"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/godror/godror"
	"github.com/rabbitmq/amqp091-go"
)

// type DEPARA_INTEGRA struct {
// 	CD_DEPARA_INTEGRA string `json:"cd_depara_integra"`
// 	REPETICOES_FILA   int    `json:"repeticoes_fila"`
// }

type INSTRUCAO_FLEURY_MV struct {
	IdProduto        string `json:"idProduto"`
	TextoInstrucao   string `json:"textoInstrucao"`
	IdInstrucaoGeral string `json:"idInstrucaoGeral"`
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

func Consumer() {
	// _, err := acessoFleury("100")
	// if err != nil {
	// 	log.Println("Erro ao acessar Fleury:", err)
	// 	return
	// }
	label := "Consumer"
	conn, err := amqp091.Dial("amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@rabbitmq:" + os.Getenv("RABBITMQ_PORT") + "/")
	if err != nil {
		for i := 0; i < 5; i++ {
			conn, err = amqp091.Dial("amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@rabbitmq:" + os.Getenv("RABBITMQ_PORT") + "/")
			if err == nil {
				//log.Println("Conectou no RabbitMQ")
				utils.LogMonitor(utils.Debug, label, "Conectou no RabbitMQ")

				break
			}
			//log.Println("Tentativa falhou, tentando de novo...")
			utils.LogMonitor(utils.Debug, label, "Tentativa falhou, tentando de novo...")
			time.Sleep(3 * time.Second)
		}
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		//log.Println("Erro ao conectar no RabbitMQ: " + err.Error())
		utils.LogMonitor(utils.ErroConexao, label, "Erro ao conectar no RabbitMQ: "+err.Error())
		return
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
		//log.Println("Erro ao declarar fila: " + err.Error())
		utils.LogMonitor(utils.ErroConexao, label, "Erro ao declarar fila: "+err.Error())
		return
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
		//log.Println("Erro ao consumir fila: " + err.Error())
		utils.LogMonitor(utils.ErroConexao, label, "Erro ao consumir fila: "+err.Error())
		return
	}

	forever := make(chan bool)

	go func() {
		ticker := time.NewTicker(10 * time.Minute) // a cada 10 minutos
		defer ticker.Stop()

		listaProdutosErros := []ERROS_INTEGRA{}
		//listaProdutosSemInstrucao := []ERROS_INTEGRA{}

		for {
			select {
			case d := <-msgs:
				var cdDepara DEPARA_INTEGRA
				if err := json.Unmarshal(d.Body, &cdDepara); err != nil {
					utils.LogMonitor(utils.ErroGeral, label, "Erro ao decodificar: "+err.Error())
					continue
				}

				// instrucaoFleury, err := acessoFleury(cdDepara)
				// if err != nil {
				// 	utils.LogMonitor(utils.ErroFleury, label, "Erro ao acessar Fleury: "+err.Error())
				// 	if cdDepara.REPETICOES_FILA > 3 {
				// 		listaProdutosErros = append(listaProdutosErros, ERROS_INTEGRA{
				// 			CD_DEPARA_INTEGRA: cdDepara.CD_DEPARA_INTEGRA,
				// 			ERROS:             "Erro ao acessar Fleury: " + err.Error(),
				// 		})
				// 	}
				// 	continue
				// }

				accessToken, err := acessoFleury()
				if err != nil {
					utils.LogMonitor(utils.ErroFleury, label, "Erro ao acessar Fleury: "+err.Error())
					if cdDepara.REPETICOES_FILA > 3 {
						listaProdutosErros = append(listaProdutosErros, ERROS_INTEGRA{
							CD_DEPARA_MV:      cdDepara.CD_DEPARA_MV,
							CD_DEPARA_INTEGRA: cdDepara.CD_DEPARA_INTEGRA,
							ERROS:             "Erro ao acessar Fleury: " + err.Error(),
						})
					}
					continue
				}
				erroProcedure := buscaProcessaInstrucao(cdDepara.CD_DEPARA_MV, cdDepara.CD_DEPARA_INTEGRA, accessToken)

				if erroProcedure != "" {
					utils.LogMonitor(utils.ErroFleury, label, "Erro ao processar instrucao: "+erroProcedure)
					if cdDepara.REPETICOES_FILA > 3 {
						listaProdutosErros = append(listaProdutosErros, ERROS_INTEGRA{
							CD_DEPARA_MV:      cdDepara.CD_DEPARA_MV,
							CD_DEPARA_INTEGRA: cdDepara.CD_DEPARA_INTEGRA,
							ERROS:             "Erro ao processar instrucao: " + erroProcedure,
						})
					}
					continue
				}

				// if instrucaoFleury.IdProduto == "" {
				// 	utils.LogMonitor(utils.ErroFleury, label, "Produto "+cdDepara.CD_DEPARA_INTEGRA+" nao encontrado no Fleury")
				// 	listaProdutosSemInstrucao = append(listaProdutosSemInstrucao, ERROS_INTEGRA{
				// 		CD_DEPARA_INTEGRA: cdDepara.CD_DEPARA_INTEGRA,
				// 		ERROS:             "Produto nao encontrado no Fleury",
				// 	})
				// 	continue
				// }

				// if instrucaoFleury.TextoInstrucao == "" {
				// 	utils.LogMonitor(utils.ErroFleury, label, "Produto "+cdDepara.CD_DEPARA_INTEGRA+" sem instrução no Fleury")
				// 	listaProdutosSemInstrucao = append(listaProdutosSemInstrucao, ERROS_INTEGRA{
				// 		CD_DEPARA_INTEGRA: cdDepara.CD_DEPARA_INTEGRA,
				// 		ERROS:             "Produto sem instrução no Fleury",
				// 	})
				// 	continue
				// }

				// if err := procedureMV(instrucaoFleury); err != nil {
				// 	utils.LogMonitor(utils.ErroIntegracao, label, "Erro ao executar procedure para produto: "+cdDepara.CD_DEPARA_INTEGRA+" - "+err.Error())
				// 	listaProdutosErros = append(listaProdutosErros, ERROS_INTEGRA{
				// 		CD_DEPARA_INTEGRA: cdDepara.CD_DEPARA_INTEGRA,
				// 		ERROS:             "Erro ao executar procedure: " + err.Error(),
				// 	})
				// 	continue
				// }

			case <-ticker.C:
				mensagem := ""
				if len(listaProdutosErros) > 0 {
					mensagem = "Produtos com erros: " + strconv.Itoa(len(listaProdutosErros))
					for _, produto := range listaProdutosErros {
						mensagem += "\n" + produto.CD_DEPARA_INTEGRA + " - " + produto.ERROS
					}
				}
				// if len(listaProdutosSemInstrucao) > 0 {
				// 	mensagem += "\n\nProdutos sem instrucao: " + strconv.Itoa(len(listaProdutosSemInstrucao))
				// 	for _, produto := range listaProdutosSemInstrucao {
				// 		mensagem += "\n" + produto.CD_DEPARA_INTEGRA
				// 	}
				// }
				if len(mensagem) > 0 {
					mail.Send("Subject: Produtos com erros Fleury \n\n Abaixo os produtos com erros: " + mensagem)
					listaProdutosErros = nil
					//listaProdutosSemInstrucao = nil
				}
			}
		}
	}()
	utils.LogMonitor(utils.Debug, label, "Consumidor aguardando mensagens...")
	<-forever

}

func reenviaParaFila(mensagem DEPARA_INTEGRA, ch *amqp091.Channel, q amqp091.Queue) {
	label := "ReenviaParaFila"
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
		//log.Println("Erro ao publicar mensagem:", err)
		utils.LogMonitor(utils.ErroConexao, label, "Erro ao publicar mensagem: "+err.Error())
	} else {
		fmt.Printf("Mensagem enviada: %+v\n", mensagem)
		utils.LogMonitor(utils.Debug, label, "Mensagem enviada: "+mensagem.CD_DEPARA_INTEGRA)
	}
}

func acessoFleury() (string, error) {
	label := "AcessoFleury"
	//produto := depara.CD_DEPARA_INTEGRA
	url := os.Getenv("FLEURY_URL") + `/oauth/grant-code`
	method := "POST"

	payload := strings.NewReader(`{
	"client_id": "` + os.Getenv("FLEURY_CLIENT_ID_GRANT_CODE") + `",
	"redirect_uri": "http://localhost",
	"extra_info":{
		"idCliente":` + os.Getenv("FLEURY_BODY_CLIENTID_GRANT_CODE") + `,
		"tipoCliente":"` + os.Getenv("FLEURY_BODY_TIPOCLIENTE_GRANT_CODE") + `"
	}
  }
  `)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao criar request: "+err.Error())
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao fazer request: "+err.Error())
		return "", err
	}
	defer res.Body.Close()

	// Lê a resposta da outra API
	grantCode := GrantCode{}
	respJSON, _ := io.ReadAll(res.Body)
	json.Unmarshal(respJSON, &grantCode)

	code := strings.Replace(grantCode.RedirectUri, "http://localhost/?code=", "", 1)

	url = os.Getenv("FLEURY_URL") + `/oauth/access-token`
	method = "POST"

	payload = strings.NewReader("grant_type=authorization_code&code=" + code)

	client = &http.Client{}
	req, err = http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao criar request: "+err.Error())
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("client_id", os.Getenv("FLEURY_CLIENT_ID_GRANT_CODE"))
	req.Header.Add("Authorization", "Basic "+os.Getenv("FLEURY_ACCESS_TOKEN"))

	res, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao fazer request: "+err.Error())
		return "", err
	}
	defer res.Body.Close()

	accessToken := AccessToken{}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao ler resposta: "+err.Error())
		return "", err
	}
	fmt.Println(string(body))
	json.Unmarshal(body, &accessToken)

	return accessToken.AccessToken, nil

}

func buscaProcessaInstrucao(produtoMV string, produtoFleury string, accessToken string) string {
	label := "BuscaProcessaInstrucao"
	url := os.Getenv("FLEURY_URL") + `/instrucoes-gerais-hospitais/v1/produtos?produtos=` + produtoFleury
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao criar request: "+err.Error())
		return "Erro ao criar request: " + err.Error()
	}
	req.Header.Add("client_id", os.Getenv("FLEURY_CLIENT_ID_GRANT_CODE"))
	req.Header.Add("access_token", accessToken)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao fazer request: "+err.Error())
		return "Erro ao fazer request: " + err.Error()
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		utils.LogMonitor(utils.ErroFleury, label, "Erro ao ler resposta: "+err.Error())
		return "Erro ao ler resposta: " + err.Error()
	}
	//fmt.Println(string(body))
	//log.Println(string(body))

	// mensagem := Mensagem{}
	// json.Unmarshal(body, &mensagem)
	// log.Println(mensagem)
	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "Erro ao deserializar resposta: " + err.Error()
	}
	if len(result) == 0 {
		return "Produto nao encontrado no Fleury"
	}
	//fmt.Println("idProduto:", result[0]["idProduto"])
	instrucoes := result[0]["instrucoesGerais"].([]interface{})
	var instrucoesMV INSTRUCAO_FLEURY_MV
	instrucoesMV.IdProduto = produtoMV

	erroProcedure := ""
	instrucaoFleury := ""

	for _, i := range instrucoes {

		instrucao := i.(map[string]interface{})
		//fmt.Println(instrucao["idInstrucaoGeral"])
		// if instrucao["idInstrucaoGeral"] != float64(6) {
		// 	continue
		// }
		nomeInstrucaoGeral := instrucao["nomeInstrucaoGeral"].(string)
		instrucoesMV.TextoInstrucao = nomeInstrucaoGeral + " \n " + instrucao["textoInstrucao"].(string)
		instrucoesMV.IdInstrucaoGeral = strconv.Itoa(int(instrucao["idInstrucaoGeral"].(float64)))
		//fmt.Println("idInstrucaoGeral:", instrucao["idInstrucaoGeral"])
		//fmt.Println("textoInstrucao:", instrucao["textoInstrucao"])

		//fmt.Println(instrucao["instrucoesFilho"])
		if instrucao["instrucoesFilho"] != nil {
			for _, i := range instrucao["instrucoesFilho"].([]interface{}) {
				instrucaoFilho := i.(map[string]interface{})
				//fmt.Println(instrucaoFilho)
				nomeInstrucaoFilho := instrucaoFilho["nomeInstrucaoGeralFilho"].(string)
				textoInstrucaoFilho := instrucaoFilho["textoInstrucaoFilho"].(string)
				//fmt.Println(nomeInstrucaoFilho)
				//fmt.Println(textoInstrucaoFilho)
				instrucoesMV.TextoInstrucao += "\n" + nomeInstrucaoFilho + "\n" + textoInstrucaoFilho
			}
		}

		utils.LogMonitor(utils.Debug, label, "IdProduto: "+produtoMV)
		utils.LogMonitor(utils.Debug, label, "TextoInstrucao: "+instrucoesMV.TextoInstrucao)
		utils.LogMonitor(utils.Debug, label, "IdInstrucaoGeral: "+instrucoesMV.IdInstrucaoGeral)

		if instrucoesMV.IdInstrucaoGeral == "6" {
			err = procedureMV(instrucoesMV)
			if err != nil {
				utils.LogMonitor(utils.ErroIntegracao, label, "Erro ao executar procedure: "+err.Error())
				erroProcedure += "Erro ao executar procedure para idInstrucao: " + instrucoesMV.IdInstrucaoGeral + " - " + err.Error() + "\n"
			}
		} else {
			instrucaoFleury += instrucoesMV.TextoInstrucao + "\n\n"
		}

		// err = procedureMV(instrucoesMV)
		// if err != nil {
		// 	utils.LogMonitor(utils.ErroIntegracao, label, "Erro ao executar procedure: "+err.Error())
		// 	erroProcedure += "Erro ao executar procedure para idInstrucao: " + instrucoesMV.IdInstrucaoGeral + " - " + err.Error() + "\n"
		// }
	}
	if instrucaoFleury != "" {
		instrucoesMV.TextoInstrucao = instrucaoFleury
		instrucoesMV.IdInstrucaoGeral = "9"
		err = procedureMV(instrucoesMV)
		if err != nil {
			utils.LogMonitor(utils.ErroIntegracao, label, "Erro ao executar procedure: "+err.Error())
			erroProcedure += "Erro ao executar procedure para idInstrucao: " + instrucoesMV.IdInstrucaoGeral + " - " + err.Error() + "\n"
		}
	}
	//log.Println(instrucoesMV)
	return erroProcedure
}

func procedureMV(instrucao INSTRUCAO_FLEURY_MV) error {
	label := "ProcedureMV"
	db, err := db.InitDB()
	if err != nil {
		//log.Println(err.Error() + " - Erro ao conectar no Oracle")
		utils.LogMonitor(utils.ErroBanco, label, "Erro ao conectar no Oracle: "+err.Error())
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("begin mvintegra.HC_IH_999(:1, :2, :3); end;")
	if err != nil {
		//log.Println(err.Error() + " - Erro ao preparar a procedure")
		utils.LogMonitor(utils.ErroBanco, label, "Erro ao preparar a procedure: "+err.Error())
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(instrucao.IdProduto, instrucao.TextoInstrucao, instrucao.IdInstrucaoGeral)
	if err != nil {
		//log.Println(err.Error() + " - Erro ao executar a procedure")
		utils.LogMonitor(utils.ErroBanco, label, "Erro ao executar a procedure: "+err.Error())
		return err
	}
	//log.Println("Procedure executada com sucesso. IdProduto: " + instrucao.IdProduto + " - TextoInstrucao: " + instrucao.TextoInstrucao)
	utils.LogMonitor(utils.Debug, label, "Procedure executada com sucesso. IdProduto: "+instrucao.IdProduto+" - TextoInstrucao: "+instrucao.TextoInstrucao)
	return nil
}
