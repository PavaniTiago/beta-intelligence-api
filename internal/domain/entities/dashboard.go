package entities

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"
)

// DashboardUnified representa a resposta consolidada do dashboard
type DashboardUnified struct {
	Summary        DashboardSummary        `json:"summary"`
	ChartData      []DashboardPeriodData   `json:"chartData"`
	PreviousPeriod DashboardPreviousPeriod `json:"previousPeriod"`
	Meta           DashboardMeta           `json:"meta"`
	ETag           string                  `json:"-"` // Campo interno para geração de ETag
}

// DashboardSummary contém dados resumidos de métricas principais
type DashboardSummary struct {
	Sessions    DashboardMetric     `json:"sessions"`
	Leads       DashboardMetric     `json:"leads"`
	Clients     DashboardMetric     `json:"clients"`
	Conversions DashboardConversion `json:"conversions"`
}

// DashboardMetric contém uma métrica com comparativo de período anterior
type DashboardMetric struct {
	Count          int64            `json:"count"`
	Active         int64            `json:"active,omitempty"` // Usado apenas para sessions
	PreviousPeriod MetricComparison `json:"previousPeriod"`
}

// DashboardConversion contém taxa de conversão com comparativo
type DashboardConversion struct {
	Rate           float64        `json:"rate"`
	PreviousPeriod RateComparison `json:"previousPeriod"`
}

// MetricComparison contém dados de comparação entre períodos para contagens
type MetricComparison struct {
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
	IsPositive bool    `json:"isPositive"`
}

// RateComparison contém dados de comparação entre períodos para taxas
type RateComparison struct {
	Rate       float64 `json:"rate"`
	Percentage float64 `json:"percentage"`
	IsPositive bool    `json:"isPositive"`
}

// DashboardPeriodData contém dados de um único período para visualização em gráficos
type DashboardPeriodData struct {
	Period        string  `json:"period"`
	DisplayPeriod string  `json:"displayPeriod"`
	Sessions      int64   `json:"sessions"`
	Leads         int64   `json:"leads"`
	Clients       int64   `json:"clients"`
	Conversions   float64 `json:"conversions"`
}

// DashboardPreviousPeriod contém metadados sobre o período anterior
type DashboardPreviousPeriod struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

// DashboardMeta contém metadados sobre os filtros aplicados
type DashboardMeta struct {
	Profession *MetaItem `json:"profession,omitempty"`
	Funnel     *MetaItem `json:"funnel,omitempty"`
}

// MetaItem representa um item de metadados com ID e nome
type MetaItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CalculateETag gera um hash único para identificar a versão dos dados
func (d *DashboardUnified) CalculateETag() string {
	data, _ := json.Marshal(d)
	hash := md5.Sum(data)
	d.ETag = fmt.Sprintf("%x", hash)
	return d.ETag
}

// GetPeriodLabel retorna a descrição textual do período comparativo
func GetPeriodLabel(timeFrame string) string {
	switch timeFrame {
	case "Daily":
		return "dia anterior"
	case "Weekly":
		return "semana anterior"
	case "Monthly":
		return "mês anterior"
	case "Yearly":
		return "ano anterior"
	default:
		return "período anterior"
	}
}

// FormatDisplayPeriod formata uma data para exibição de acordo com o timeFrame
func FormatDisplayPeriod(date time.Time, timeFrame string) string {
	switch timeFrame {
	case "Daily":
		return date.Format("02/01")
	case "Weekly":
		return fmt.Sprintf("Sem %d", date.Day()/7+1)
	case "Monthly":
		return date.Format("01/2006")
	case "Yearly":
		return date.Format("2006")
	default:
		return date.Format("02/01")
	}
}
