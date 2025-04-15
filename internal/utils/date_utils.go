package utils

import "time"

// GetBrasilLocation retorna a localização de São Paulo (UTC-3)
// Esta função deve ser usada em todo o projeto para obter o fuso horário padrão brasileiro,
// garantindo consistência em todas as operações relacionadas a data e hora.
func GetBrasilLocation() *time.Location {
	brazilLocation, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Fallback para UTC-3 se não conseguir carregar a localização
		brazilLocation = time.FixedZone("BRT", -3*60*60)
	}
	return brazilLocation
}
