package mail

// ErroStatusIH erro padrão para casos em que ocorra um erro durante envio do JSON
var ErroStatusIH string = "Subject: Erro durante envio \n\n"

// ErroEnvioIH erro padrão para casos em que ocorra um erro durante envio do JSON
var ErroEnvioIH string = "Subject: Erro ao enviar registro \n\n"

// ErroRecebimentoIH erro padrão para casos em que ocorra um erro durante a recepção/adicao de um JSON
var ErroRecebimentoIH string = "Subject: Erro no endpoint ao processar JSON " //aqui não foi colocado \n\n pois será concatenada a label da interface

// ErroTabelaControle erro padrão para casos em que ocorra erro ao inserir na tabela de controle
var ErroTabelaControle string = "Subject: Erro ao inserir na tabela de controle \n\n"
