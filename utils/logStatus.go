package utils

import "log"

type StatusLog string

const (
	ErroGeral      StatusLog = "<0>"
	ErroBanco      StatusLog = "<1>"
	ErroConexao    StatusLog = "<2>"
	Debug          StatusLog = "<3>"
	ErroFleury     StatusLog = "<4>"
	SucessoFleury  StatusLog = "<5>"
	ErroIntegracao StatusLog = "<6>"
	Sucesso        StatusLog = "<7>"
)

func LogMonitor(status StatusLog, ih string, mensagem string) {
	/*
		log level:
		erroGeral: <0> erro inesperado
		erroBanco: <1> falha de banco
		erroConexao: <2> falha tolife
		debug: <3> debug
		erroStatus: <4> mensagem de erro da tolife
		sucessoStatus: <5> sucesso tolife
	*/

	var statusLog string = ""
	switch status {
	case ErroGeral: // erroGeral
		statusLog = "<0>"
	case ErroBanco: // erroBanco
		statusLog = "<1>"
	case ErroConexao: // erroConexao
		statusLog = "<2>"
	case Debug: // debug
		statusLog = "<3>"
	case ErroFleury: // erroStatus
		statusLog = "<4>"
	case SucessoFleury: // sucessoStatus
		statusLog = "<5>"
	case ErroIntegracao: // erroStatus
		statusLog = "<6>"
	case Sucesso: // sucessoStatus
		statusLog = "<7>"
	}
	log.Println(statusLog + " " + ih + " " + mensagem)

}
