package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/godror/godror"
)

/*InitDB funcao para iniciar a conexão do DB*/
func InitDB() (*sql.DB, error) {
	var dberr error

	dbURI := os.Getenv("dbStringSRC")

	log.Println(dbURI)
	db, dberr := sql.Open("godror", dbURI)
	//log.Println(dberr.Error())

	return db, dberr
}

/*PingBD funcao usada pelo endpoint para testar conexao com o banco*/
func PingBD() error {

	dbuser := os.Getenv("dbuser")
	dbpassword := os.Getenv("dbpassword")
	dbname := os.Getenv("dbname")
	dbhost := os.Getenv("dbhost")
	dbport := os.Getenv("dbport")

	dbURI := fmt.Sprintf("%s/%s@%s:%s/%s", dbuser, dbpassword, dbhost, dbport, dbname)
	log.Printf(dbURI)
	db, dberr := sql.Open("godror", dbURI)
	if dberr != nil {
		return dberr
	}

	err := db.Ping()

	return err

}

// CloseDB fecha a conexão com o db
func CloseDB(db *sql.DB) {
	db.Close()
}

// RowExists função auxiliar que retorna verdadeiro ou falso para o select enviado
func RowExists(query string, db *sql.DB, args ...interface{}) bool {

	var result int
	query = fmt.Sprintf("select case when exists (%s) then 1 else 0 end from dual", query)
	err := db.QueryRow(query, args...).Scan(&result)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("ERRO checando se existe '%s' %v", args, err)
	}
	if result == 1 {
		return true
	}

	return false
}

// NextSeq recebe o nome da sequencia e devolve o valor da mesma
// nextSeq("pessoa_juridica_estab_seq")
func NextSeq(sequencia string, db *sql.DB) string {

	log.Println("Executando nextSeq")

	sql := `select ` + sequencia + `.nextval from dual`

	row, err := db.Query(sql)
	if err != nil {
		log.Println("Erro ao executar nextval")
		log.Printf(err.Error())
	}
	defer row.Close()

	row.Next()

	var nrSeq string
	err = row.Scan(&nrSeq)
	if err != nil {
		log.Println("Erro ao obter nextval")
		log.Printf(err.Error())

	}

	return nrSeq
}
