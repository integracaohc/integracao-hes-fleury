package functions

import (
	"context"
	"integracao-hes-fleury/db"
	"integracao-hes-fleury/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/godror/godror"
)

type CD_DEPARA struct {
	CD_DEPARA_INTEGRA string `json:"cd_depara_integra"`
}

func ProcessaProdutosFleury() gin.HandlerFunc {
	fn := func(c *gin.Context) {
		Producer()
		c.JSON(http.StatusOK, gin.H{
			"sucesso": true,
			"message": "Produtos enviados para fila com sucesso",
		})
	}
	return gin.HandlerFunc(fn)
}

func ProdutosFleury() gin.HandlerFunc {
	fn := func(c *gin.Context) {
		label := "ProdutosFleury"
		produto := c.Param("produto")
		//log.Println("Produto:", produto)
		utils.LogMonitor(utils.Debug, label, "Produto: "+produto)

		db, err := db.InitDB()
		if err != nil {
			//log.Println(err.Error() + " - Erro ao conectar no Oracle")
			utils.LogMonitor(utils.ErroBanco, label, "Erro ao conectar no Oracle: "+err.Error())
			return
		}
		defer db.Close()
		ctx := context.Background()

		var produtoFleury CD_DEPARA
		// row, err := db.Query("select CD_DEPARA_INTEGRA from mvintegra.depara where cd_depara_mv = :1", produto)
		// if err != nil {
		// 	//log.Println(err.Error() + " - Erro ao buscar registros no banco")
		// 	utils.LogMonitor(utils.ErroBanco, label, "Erro ao buscar registros no banco: "+err.Error())
		// 	return
		// }
		// defer row.Close()

		// row.Scan(&produtoFleury.CD_DEPARA_INTEGRA)

		err = db.QueryRowContext(ctx, `select CD_DEPARA_INTEGRA from mvintegra.depara where cd_depara_mv = :1`, produto).Scan(&produtoFleury.CD_DEPARA_INTEGRA)
		if err != nil {
			//log.Println(err.Error() + " - Erro ao buscar registros no banco")
			utils.LogMonitor(utils.ErroBanco, label, "Erro ao buscar registros no banco: "+err.Error())
			return
		}

		depara := DEPARA_INTEGRA{
			CD_DEPARA_INTEGRA: produtoFleury.CD_DEPARA_INTEGRA,
			REPETICOES_FILA:   0,
		}
		utils.LogMonitor(utils.Debug, label, "Produto encontrado no banco: "+depara.CD_DEPARA_INTEGRA)
		// accessToken, err := acessoFleury()
		// if err != nil {
		// 	//log.Println("Erro ao acessar Fleury:", err)
		// 	utils.LogMonitor(utils.ErroFleury, label, "Erro ao acessar Fleury: "+err.Error())
		// 	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		// 		"sucesso": false,
		// 		"message": "Erro ao acessar Fleury " + err.Error(),
		// 	})
		// 	return

		// }
		// //utils.LogMonitor(utils.Debug, label, "Produto encontrado no Fleury: "+instrucaoFleury.IdProduto)
		// if instrucaoFleury.IdProduto == "" {
		// 	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
		// 		"sucesso": false,
		// 		"message": "Produto não encontrado no Fleury",
		// 	})
		// 	return
		// }

		accessToken, err := acessoFleury()
		if err != nil {
			utils.LogMonitor(utils.ErroFleury, label, "Erro ao acessar Fleury: "+err.Error())
			// if cdDepara.REPETICOES_FILA > 3 {
			// 	listaProdutosErros = append(listaProdutosErros, ERROS_INTEGRA{
			// 		CD_DEPARA_INTEGRA: cdDepara.CD_DEPARA_INTEGRA,
			// 		ERROS:             "Erro ao acessar Fleury: " + err.Error(),
			// 	})
			// }
			//continue
		}
		erroProcedure := buscaProcessaInstrucao(produto, depara.CD_DEPARA_INTEGRA, accessToken)

		if erroProcedure != "" {
			utils.LogMonitor(utils.ErroFleury, label, "Erro ao processar instrucao: "+erroProcedure)
			mensagemErro := ""
			if err.Error() == "Dados não encontrados" {
				mensagemErro = "Erro ao executar procedure: Produto não encontrado no MV"
			} else {
				mensagemErro = "Erro ao executar procedure: " + err.Error()
			}
			utils.LogMonitor(utils.ErroIntegracao, label, "Erro ao executar procedure: "+mensagemErro)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"sucesso": false,
				"message": mensagemErro,
			})
			return
		}

		//log.Println(mensagem)
		//log.Println("IdProduto:", instrucaoFleury.IdProduto)
		//log.Println("TextoInstrucao:", instrucaoFleury.TextoInstrucao)
		// utils.LogMonitor(utils.Debug, label, "IdProduto: "+instrucaoFleury.IdProduto)
		// utils.LogMonitor(utils.Debug, label, "TextoInstrucao: "+instrucaoFleury.TextoInstrucao)
		// utils.LogMonitor(utils.Debug, label, "IdInstrucaoGeral: "+instrucaoFleury.IdInstrucaoGeral)

		utils.LogMonitor(utils.Sucesso, label, "Produto processado com sucesso")
		c.JSON(http.StatusOK, gin.H{
			"sucesso": true,
			"message": "Produto processado com sucesso",
		})
	}
	return gin.HandlerFunc(fn)
}
