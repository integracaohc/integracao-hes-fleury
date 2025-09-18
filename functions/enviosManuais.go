package functions

import (
	"integracao-hes-fleury/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/godror/godror"
)

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

		depara := DEPARA_INTEGRA{
			CD_DEPARA_INTEGRA: produto,
			REPETICOES_FILA:   0,
		}

		instrucaoFleury, err := acessoFleury(depara)
		if err != nil {
			//log.Println("Erro ao acessar Fleury:", err)
			utils.LogMonitor(utils.ErroFleury, label, "Erro ao acessar Fleury: "+err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"sucesso": false,
				"message": "Erro ao acessar Fleury " + err.Error(),
			})
			return

		}
		if instrucaoFleury.IdProduto == "" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"sucesso": false,
				"message": "Produto não encontrado no Fleury",
			})
			return
		}
		//log.Println(mensagem)
		//log.Println("IdProduto:", instrucaoFleury.IdProduto)
		//log.Println("TextoInstrucao:", instrucaoFleury.TextoInstrucao)
		utils.LogMonitor(utils.Debug, label, "IdProduto: "+instrucaoFleury.IdProduto)
		utils.LogMonitor(utils.Debug, label, "TextoInstrucao: "+instrucaoFleury.TextoInstrucao)

		err = procedureMV(instrucaoFleury)
		if err != nil {
			//log.Println("Erro ao executar procedure:", err)
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
		utils.LogMonitor(utils.Sucesso, label, "Produto processado com sucesso")
		c.JSON(http.StatusOK, gin.H{
			"sucesso": true,
			"message": "Produto processado com sucesso",
		})
	}
	return gin.HandlerFunc(fn)
}
