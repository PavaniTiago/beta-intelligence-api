# Documentação da API BI Pesquisas

Este documento descreve os endpoints disponíveis na API de BI Pesquisas, incluindo os filtros suportados, formatos de dados e cenários de uso comum.

## Tabela de Conteúdo
- [Visão Geral](#visão-geral)
- [Endpoints](#endpoints)
  - [Verificação de Saúde](#verificação-de-saúde)
  - [Métricas de Pesquisas](#métricas-de-pesquisas)
  - [Detalhes de Pesquisa](#detalhes-de-pesquisa)
- [Tipos de Eventos](#tipos-de-eventos)
- [Filtros de Data](#filtros-de-data)
  - [Filtros de Data Padrão](#filtros-de-data-padrão)
  - [Filtros de Data Específicos por Tipo de Evento](#filtros-de-data-específicos-por-tipo-de-evento)
- [Cenários de Uso](#cenários-de-uso)
  - [Monitoramento de Campanhas](#monitoramento-de-campanhas)
  - [Análise de Conversão por Período](#análise-de-conversão-por-período)
  - [Comparação Entre Fases de Lançamento](#comparação-entre-fases-de-lançamento)
  - [Segmentação por Faixas de Renda e Faixa Etária](#segmentação-por-faixas-de-renda-e-faixa-etária)
  - [Análise de Score por Período](#análise-de-score-por-período)
- [Considerações sobre Fusos Horários](#considerações-sobre-fusos-horários)

## Visão Geral

A API do BI Pesquisas fornece acesso a métricas e análises detalhadas de pesquisas e vendas. Todos os endpoints retornam dados em formato JSON e suportam vários filtros para análises personalizadas.

A base URL para todos os endpoints é: `/api/v1`

## Endpoints

### Verificação de Saúde

```
GET /health
```

Retorna o status da API para verificação de funcionamento.

**Resposta de Exemplo:**
```json
{
  "status": "ok",
  "message": "API de BI Pesquisas funcionando corretamente"
}
```

### Métricas de Pesquisas

```
GET /api/v1/metricas/pesquisas
```

Retorna métricas agregadas de todas as pesquisas disponíveis ou filtradas pelos parâmetros fornecidos.

**Parâmetros de Query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| data_inicio | string | Data de início geral (ISO8601 com timezone) |
| data_fim | string | Data de fim geral (ISO8601 com timezone) |
| lead_inicio | string | Data de início para leads (ISO8601 com timezone) |
| lead_fim | string | Data de fim para leads (ISO8601 com timezone) |
| pesquisa_inicio | string | Data de início para respostas de pesquisa (ISO8601 com timezone) |
| pesquisa_fim | string | Data de fim para respostas de pesquisa (ISO8601 com timezone) |
| venda_inicio | string | Data de início para vendas (ISO8601 com timezone) |
| venda_fim | string | Data de fim para vendas (ISO8601 com timezone) |
| profissao | string | Filtrar por profissão |
| funil | string | Filtrar por funil |
| pesquisa_id | integer | Filtrar por ID da pesquisa |

**Resposta de Exemplo:**
```json
[
  {
    "nome_pesquisa": "Lead Scoring",
    "funil": "Semanal - Lead Scoring - Cadastro",
    "total_leads": 10839,
    "total_respostas": 8251,
    "total_vendas": 82,
    "taxa_resposta": 0.7612,
    "conversao_vendas": 0.0099
  },
  {
    "nome_pesquisa": "Qualificação Inicial",
    "funil": "Mensal - Produto X",
    "total_leads": 5200,
    "total_respostas": 4100,
    "total_vendas": 45,
    "taxa_resposta": 0.7885,
    "conversao_vendas": 0.0110
  }
]
```

### Detalhes de Pesquisa

```
GET /api/v1/metricas/pesquisas/{id}
```

Retorna análise detalhada de uma pesquisa específica, com drill-down por pergunta e opção de resposta, incluindo métricas de conversão.

**Parâmetros de Path:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| id | integer | ID da pesquisa (obrigatório) |

**Parâmetros de Query:**
Os mesmos parâmetros de filtro de data disponíveis no endpoint de métricas.

**Resposta de Exemplo:**
```json
[
  {
    "pergunta_id": "1",
    "texto_pergunta": "Qual sua faixa etária?",
    "respostas": [
      {
        "opcao_id": "1",
        "texto_opcao": "18 a 25 anos",
        "score_peso": 5,
        "num_respostas": 2450,
        "percentual_participacao": 29.63,
        "num_vendas": 12,
        "taxa_conversao_percentual": 0.49,
        "percentual_vendas": 0.76
      },
      {
        "opcao_id": "2",
        "texto_opcao": "26 a 35 anos",
        "score_peso": 10,
        "num_respostas": 3680,
        "percentual_participacao": 44.48,
        "num_vendas": 38,
        "taxa_conversao_percentual": 1.03,
        "percentual_vendas": 2.40
      }
      // Mais opções...
    ]
  },
  // Mais perguntas...
]
```

## Tipos de Eventos

O sistema rastreia três tipos principais de eventos:

1. **LEAD** - Quando um usuário é registrado como lead potencial
2. **PESQUISA_LEAD** - Quando um lead responde a uma pesquisa
3. **PURCHASE** - Quando um lead realiza uma compra

Cada tipo de evento pode ser filtrado independentemente usando os filtros de data específicos.

## Filtros de Data

### Filtros de Data Padrão

Os filtros `data_inicio` e `data_fim` são aplicados a todos os tipos de eventos, a menos que filtros específicos por tipo sejam fornecidos.

**Exemplo:**
```
GET /api/v1/metricas/pesquisas?data_inicio=2025-04-01T00:00:00-03:00&data_fim=2025-04-30T23:59:59-03:00
```

Este exemplo retornará métricas para todos os eventos (leads, pesquisas e vendas) que ocorreram em abril de 2025.

### Filtros de Data Específicos por Tipo de Evento

Os filtros específicos por tipo de evento permitem análises mais granulares:

- `lead_inicio` e `lead_fim` - Filtram apenas eventos de leads
- `pesquisa_inicio` e `pesquisa_fim` - Filtram apenas eventos de resposta a pesquisas
- `venda_inicio` e `venda_fim` - Filtram apenas eventos de vendas

**Exemplo:**
```
GET /api/v1/metricas/pesquisas?pesquisa_inicio=2025-04-15T00:00:00-03:00&venda_inicio=2025-04-22T00:00:00-03:00
```

Este exemplo analisará respostas de pesquisa a partir de 15 de abril e vendas a partir de 22 de abril.

## Cenários de Uso

### Monitoramento de Campanhas

**Cenário:** Você lançou uma campanha de marketing em 15 de abril e quer analisar as respostas das pesquisas e vendas resultantes.

```
GET /api/v1/metricas/pesquisas?lead_inicio=2025-04-15T00:00:00-03:00&lead_fim=2025-04-30T23:59:59-03:00
```

### Análise de Conversão por Período

**Cenário:** Você quer identificar se alterações recentes no script de vendas melhoraram a conversão de pesquisas para vendas após 22 de abril.

```
GET /api/v1/metricas/pesquisas?pesquisa_inicio=2025-04-01T00:00:00-03:00&venda_inicio=2025-04-22T00:00:00-03:00
```

### Comparação Entre Fases de Lançamento

**Cenário:** Você quer comparar o desempenho de uma pesquisa específica antes e depois de uma atualização de produto em 20 de abril.

```
GET /api/v1/metricas/pesquisas/1?venda_inicio=2025-04-20T00:00:00-03:00

// Comparar com:
GET /api/v1/metricas/pesquisas/1?venda_fim=2025-04-19T23:59:59-03:00
```

### Segmentação por Faixas de Renda e Faixa Etária

**Cenário:** Você quer analisar como diferentes faixas etárias e de renda respondem às suas ofertas, através das perguntas específicas na pesquisa.

```
GET /api/v1/metricas/pesquisas/1?data_inicio=2025-04-01T00:00:00-03:00&data_fim=2025-04-30T23:59:59-03:00
```

Este exemplo retornará detalhes de conversão por pergunta, permitindo analisar perguntas sobre idade e renda.

### Análise de Score por Período

**Cenário:** Você quer verificar se o sistema de pontuação (score) está corretamente identificando leads de alta qualidade que resultam em conversões.

```
GET /api/v1/metricas/pesquisas/1?pesquisa_inicio=2025-03-01T00:00:00-03:00&venda_inicio=2025-04-01T00:00:00-03:00
```

Este exemplo permite analisar se os leads com maior pontuação em março converteram em vendas em abril.

## Considerações sobre Fusos Horários

Todos os parâmetros de data aceitam datas no formato ISO8601 com informação de timezone. Para especificar o fuso horário de Brasília (UTC-3), use o sufixo `-03:00`.

**Exemplo:**
```
2025-04-15T00:00:00-03:00
```

As datas são processadas respeitando o timezone fornecido, garantindo consistência nas análises temporais independentemente do timezone do servidor.
