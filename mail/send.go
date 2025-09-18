package mail

import (
	"errors"
	"log"
	smtp "net/smtp"
	"os"
	"strings"
)

type smtpServer struct {
	host string
	port string
}

// serverName URI to smtp server
func (s *smtpServer) serverName() string {
	return s.host + ":" + s.port
}

/*Send funcao para envio de email em casos de erros*/
func Send(dados string) error {

	// Receiver email address.
	emails := os.Getenv("emails")
	to := []string{
		"emailreport@hospitalcare.com.br",
	}

	if len(emails) != 0 {
		to = strings.Split(strings.TrimSpace(emails), ";")
	} else {
		log.Println("Variável de emails não definda, usando valor default: emailreport@hospitalcare.com.br")
	}

	hospital := os.Getenv("hospital")
	if len(hospital) == 0 {
		log.Println("Variável de ambiente com o nome do hospital não foi definda")
		hospital = "Variável de ambiente com o nome do hospital não foi definda"
	}

	dados += "\n Hospital: " + hospital

	// smtp server configuration.
	smtpServer := smtpServer{host: "smtp.office365.com", port: "587"}
	// Message.
	message := []byte(dados)

	// Authentication.

	loginEmail := os.Getenv("loginEmail")
	passwordEmail := os.Getenv("passwordEmail")

	auth := LoginAuth(loginEmail, passwordEmail)
	// Sending email.
	err := smtp.SendMail(smtpServer.serverName(), auth, loginEmail, to, message)
	if err != nil {
		log.Println("Erro ao enviar e-mail " + err.Error() + dados)
		return err
	}

	log.Println("Email enviado", dados)
	return nil
}

type loginAuth struct {
	username, password string
}

// LoginAuth funcao para autenticacao no office365
func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unknown from server")
		}
	}
	return nil, nil
}
