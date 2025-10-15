package functions

import (
	"context"
	"integracao-hes-fleury/db"
	"integracao-hes-fleury/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func DeparaMV() gin.HandlerFunc {
	fn := func(c *gin.Context) {
		label := "DeparaMV"
		depara := c.Param("cdDeparaMV")
		if depara == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"sucesso": false,
				"message": "Parâmetro 'depara' é obrigatório.",
			})
			utils.LogMonitor(utils.ErroGeral, label, "Parâmetro 'depara' é obrigatório.")
			return
		}
		utils.LogMonitor(utils.Debug, label, "Depara MV: "+depara)

		db, err := db.InitDB()
		if err != nil {
			utils.LogMonitor(utils.ErroBanco, label, "Erro ao conectar no Oracle: "+err.Error())
			return
		}
		defer db.Close()
		ctx := context.Background()

		var cdDeparaFleury CD_DEPARA
		err = db.QueryRowContext(ctx, `select CD_DEPARA_INTEGRA from mvintegra.depara where cd_depara_mv = :1`, depara).Scan(&cdDeparaFleury.CD_DEPARA_INTEGRA)
		if err != nil {
			//log.Println(err.Error() + " - Erro ao buscar registros no banco")
			utils.LogMonitor(utils.ErroBanco, label, "Erro ao buscar registros no banco: "+err.Error())
			return
		}
		utils.LogMonitor(utils.Debug, label, "Depara Fleury: "+cdDeparaFleury.CD_DEPARA_INTEGRA)
		c.JSON(http.StatusOK, gin.H{
			"sucesso":        true,
			"cdDeparaMV":     depara,
			"cdDeparaFleury": cdDeparaFleury.CD_DEPARA_INTEGRA,
		})
	}
	return gin.HandlerFunc(fn)
}
