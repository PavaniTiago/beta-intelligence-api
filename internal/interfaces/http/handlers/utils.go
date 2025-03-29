package handlers

import "time"

// GenerateDateRange gera um array de strings de datas no formato "YYYY-MM-DD"
// para todas as datas no intervalo from até to (inclusive)
func GenerateDateRange(from, to time.Time) []string {
	if from.IsZero() || to.IsZero() || from.After(to) {
		return []string{}
	}

	// Normalizar as datas para início do dia
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())

	// Calcular o número de dias entre as datas
	duration := to.Sub(from)
	days := int(duration.Hours()/24) + 1 // +1 para incluir o dia final

	// Gerar array de datas
	result := make([]string, days)
	currentDate := from

	for i := 0; i < days; i++ {
		result[i] = currentDate.Format("2006-01-02")
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return result
}
