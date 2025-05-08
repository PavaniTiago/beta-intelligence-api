# Estrutura do Banco de Dados Supabase

Este documento mapeia a estrutura do banco de dados existente para o projeto BI Pesquisas, com foco nas relações entre tabelas e nos campos relevantes para a implementação do backend.

## Tabelas Principais

### Professions (Profissões)

```
professions {
  profession_id: bigint [PK]
  profession_name: text
  created_at: timestamp with time zone
  meta_pixel: text
  meta_token: text
  is_testing: boolean
}
```

- **Descrição**: Representa diferentes profissões de usuários
- **Relacionamentos**: Referenciada por Products e Events
- **Observações**: Contém configurações específicas de marketing (meta_pixel, meta_token)

### Products (Produtos)

```
products {
  product_id: bigint [PK]
  product_name: text
  profession_id: bigint [FK → professions.profession_id]
  created_at: timestamp with time zone
}
```

- **Descrição**: Produtos oferecidos por profissão
- **Relacionamentos**: 
  - N:1 com Professions (cada produto pertence a uma profissão)
  - 1:N com Funnels (um produto pode ter vários funis)

### Funnels (Funis)

```
funnels {
  funnel_id: bigint [PK]
  funnel_name: text
  funnel_tag: text
  product_id: bigint [FK → products.product_id]
  created_at: timestamp with time zone
  global: boolean
  is_active: boolean
  is_testing: boolean
  is_googleads: boolean
}
```

- **Descrição**: Funis de marketing e vendas
- **Relacionamentos**:
  - N:1 com Products (cada funil está associado a um produto)
  - 1:N com Surveys (um funil pode ter várias pesquisas)
- **Flags importantes**: Controle de status e tipo de funil

### Surveys (Pesquisas)

```
surveys {
  survey_id: bigint [PK]
  survey_name: text
  alt_survey_name: text
  funnel_id: bigint [FK → funnels.funnel_id]
  created_at: timestamp with time zone
  updated_at: timestamp with time zone
}
```

- **Descrição**: Pesquisas associadas a funis específicos
- **Relacionamentos**:
  - N:1 com Funnels (cada pesquisa pertence a um funil)
  - 1:N com Survey_Responses (uma pesquisa pode ter múltiplas respostas)

### Survey_Responses (Respostas de Pesquisas)

```
survey_responses {
  id: uuid [PK]
  survey_id: bigint [FK → surveys.survey_id]
  event_id: uuid [FK → events.event_id]
  total_score: bigint
  completed: boolean
  created_at: timestamp with time zone
  faixa: USER-DEFINED
}
```

- **Descrição**: Armazena as respostas completas às pesquisas (uma linha por pesquisa respondida)
- **Relacionamentos**:
  - N:1 com Surveys (cada resposta pertence a uma pesquisa)
  - N:1 com Events (cada resposta é gerada por um evento)
  - 1:N com Survey_Answers (uma resposta completa tem múltiplas respostas individuais)
- **Observações**: 
  - O campo `total_score` representa a pontuação total calculada para a resposta
  - O campo `faixa` (A, B, C...) é uma classificação atribuída à resposta, possivelmente para qualificação de leads

### Survey_Answers (Respostas Individuais)

```
survey_answers {
  id: uuid [PK]
  survey_response_id: uuid [FK → survey_responses.id]
  question_id: varchar [not null]
  question_text: text
  value: text [not null]
  score: bigint
  time_to_answer: double precision
  changed: boolean
  timestamp: timestamp with time zone [not null]
}
```

- **Descrição**: Armazena cada resposta individual a uma pergunta da pesquisa
- **Relacionamentos**:
  - N:1 com Survey_Responses (várias respostas individuais para uma única resposta de pesquisa)
- **Observações**: 
  - O campo `score` permite atribuir pontuação a respostas específicas (para scoring de leads)
  - O campo `time_to_answer` registra o tempo que o usuário levou para responder (indicador de engajamento)
  - Os `question_id` seguem um padrão de nomenclatura (ex: "question_0001", "question_0002")

### Events (Eventos)

```
events {
  event_id: uuid [PK]
  event_name: text
  event_time: timestamp with time zone
  user_id: uuid [not null]
  profession_id: bigint [FK → professions.profession_id]
  product_id: bigint [FK → products.product_id]
  funnel_id: bigint [FK → funnels.funnel_id]
  pageview_id: uuid
  session_id: uuid
  event_source: text
  event_type: USER-DEFINED
  event_propeties: jsonb
}
```

- **Descrição**: Eventos rastreados durante interação do usuário
- **Relacionamentos**:
  - Relaciona-se com múltiplas entidades: professions, products, funnels
- **Observações**: 
  - Campo `event_type` contém valores como 'LEAD', 'PESQUISA_LEAD', 'PURCHASE'
  - O `user_id` é usado para rastrear usuários através de diferentes eventos

## Estrutura Relacional

```
Professions 1---* Products 1---* Funnels 1---* Surveys 1---* Survey_Responses 1---* Survey_Answers
                     ^            ^             ^
                     |            |             |
                     +------------+-------------+
                                  |
                                Events
```

## Adaptações para os Modelos Go

### Nomenclatura e Tipos

1. **Tabelas em inglês vs. português**:
   - No banco: `funnels`, `surveys`, `products` (inglês)
   - Em nossos modelos: Usar `TableName()` para mapear corretamente

2. **Tipos de ID**:
   - Usar `uint` para IDs do tipo `bigint`
   - Usar `string` para IDs do tipo `uuid`

### Exemplo de Adaptação de Modelo

```go
// Funil representa um funil de vendas
type Funil struct {
    ID        uint      `json:"id" gorm:"primaryKey;column:funnel_id"`
    Nome      string    `json:"nome" gorm:"column:funnel_name"`
    Tag       string    `json:"tag" gorm:"column:funnel_tag"`
    ProdutoID uint      `json:"produto_id" gorm:"column:product_id"`
    CriadoEm  time.Time `json:"criado_em" gorm:"column:created_at"`
    Global    bool      `json:"global"`
    Ativo     bool      `json:"ativo" gorm:"column:is_active"`
    Teste     bool      `json:"teste" gorm:"column:is_testing"`
    GoogleAds bool      `json:"google_ads" gorm:"column:is_googleads"`
}

// TableName sobrescreve o nome da tabela para Funil
func (Funil) TableName() string {
    return "funnels"
}
```

## Consultas Importantes

### Métricas de Pesquisas (Implementação Atual)

A consulta a seguir é utilizada no projeto para obter métricas de pesquisas e vendas, baseando-se em eventos rastreados na tabela `events`. A consulta utiliza CTEs (Common Table Expressions) para organizar e filtrar os dados em etapas:

```sql
WITH pesquisa_users AS (
  SELECT user_id
  FROM events
  WHERE event_type = 'PESQUISA_LEAD'
),

eventos_filtrados AS (
  SELECT e.*
  FROM events e
  WHERE e.event_type IN ('LEAD', 'PESQUISA_LEAD', 'PURCHASE')
),

base AS (
  SELECT 
    s.alt_survey_name,
    f.funnel_name,
    e.user_id,
    e.event_type
  FROM surveys s
  JOIN funnels f ON s.funnel_id = f.funnel_id
  LEFT JOIN eventos_filtrados e ON f.funnel_id = e.funnel_id
  WHERE s.survey_id = 1 -- Filtro para pesquisa específica
)

SELECT 
  b.alt_survey_name,
  b.funnel_name,

  COUNT(*) FILTER (WHERE b.event_type = 'LEAD') AS total_leads,
  COUNT(*) FILTER (WHERE b.event_type = 'PESQUISA_LEAD') AS total_respostas,

  COUNT(*) FILTER (
    WHERE b.event_type = 'PURCHASE' 
      AND b.user_id IN (SELECT user_id FROM pesquisa_users)
  ) AS total_vendas_com_pesquisa,

  ROUND(
    (COUNT(*) FILTER (WHERE b.event_type = 'PESQUISA_LEAD')::numeric /
     NULLIF(COUNT(*) FILTER (WHERE b.event_type = 'LEAD'), 0)::numeric) * 100, 
    2
  ) / 100 AS taxa_resposta_calculada,

  ROUND(
    (COUNT(*) FILTER (
       WHERE b.event_type = 'PURCHASE' 
         AND b.user_id IN (SELECT user_id FROM pesquisa_users)
     )::numeric /
     NULLIF(COUNT(*) FILTER (WHERE b.event_type = 'PESQUISA_LEAD'), 0)::numeric) * 100,
    2
  ) / 100 AS conversao_vendas_calculada

FROM base b
GROUP BY 
  b.alt_survey_name,
  b.funnel_name;
```

Esta consulta produz os seguintes resultados:
- **alt_survey_name**: Nome alternativo da pesquisa
- **funnel_name**: Nome do funil
- **total_leads**: Número total de leads
- **total_respostas**: Número de respostas à pesquisa
- **total_vendas_com_pesquisa**: Número de vendas de usuários que responderam à pesquisa
- **taxa_resposta_calculada**: Porcentagem de leads que responderam à pesquisa
- **conversao_vendas_calculada**: Porcentagem de respostas que resultaram em vendas

### Métricas de Pesquisas (Alternativa Genérica)

```sql
SELECT 
    s.survey_id as pesquisa_id,
    s.survey_name as nome_pesquisa,
    p.profession_name as profissao,
    f.funnel_name as funil,
    COUNT(DISTINCT sr.id) as total_respostas,
    COUNT(DISTINCT e.event_id) FILTER (WHERE e.event_name = 'purchase') as total_vendas,
    COUNT(DISTINCT s.survey_id) as total_pesquisas
FROM 
    surveys s
    JOIN funnels f ON s.funnel_id = f.funnel_id
    JOIN products pr ON f.product_id = pr.product_id
    JOIN professions p ON pr.profession_id = p.profession_id
    LEFT JOIN survey_responses sr ON s.survey_id = sr.survey_id
    LEFT JOIN events e ON s.survey_id = e.survey_id AND e.event_name = 'purchase'
GROUP BY 
    s.survey_id, s.survey_name, p.profession_name, f.funnel_name
```

## Considerações para Implementação

1. **Migrações**:
   - Não utilizar auto-migração, pois as tabelas já existem
   - Focar apenas na conexão e mapeamento correto

2. **Consultas**:
   - Adaptar queries nos repositórios para usar os nomes corretos de tabelas e colunas
   - Implementar filtros por data para o timezone de Brasília

3. **DTOs**:
   - Manter os DTOs em português para a API, realizando a tradução/mapeamento internamente

## Análise Detalhada por Pergunta e Resposta

Além das métricas agregadas, o sistema permite uma análise aprofundada das respostas individuais e sua correlação com as conversas, fornecendo insights valiosos para otimização do funil e estratégias de vendas.

### Consulta Análitica de Conversão por Resposta

A consulta a seguir realiza uma análise avançada que correlaciona as respostas específicas de cada pergunta com taxas de conversão:

```sql
-- ALTERE APENAS ESTES PARÂMETROS
WITH parametros AS (
    SELECT 
        9 AS funnel_id,                           -- ID do funil a ser analisado
        '2025-04-22 20:00:00-03:00'::timestamptz AS pesquisa_inicio,  -- Data/hora inicial das pesquisas
        '2025-04-29 20:00:00-03:00'::timestamptz AS pesquisa_fim,     -- Data/hora final das pesquisas
        '2025-04-29 20:45:00-03:00'::timestamptz AS venda_inicio,     -- Data/hora inicial das vendas
        '2025-04-29 23:59:59-03:00'::timestamptz AS venda_fim,        -- Data/hora final das vendas
        'question_0007' AS filtro_pergunta                  -- ID da pergunta para filtrar (deixe NULL para todas)
),

-- Identificar usuários que compraram no período especificado
compradores AS (
    SELECT DISTINCT
        v.user_id
    FROM
        events v
    WHERE
        v.funnel_id = (SELECT funnel_id FROM parametros)
        AND v.event_type = 'PURCHASE'
        AND v.event_time BETWEEN (SELECT venda_inicio FROM parametros) AND (SELECT venda_fim FROM parametros)
        AND v.event_propeties->>'product_type' = 'main'
        AND EXISTS (
            SELECT 1
            FROM events pesq
            WHERE pesq.user_id = v.user_id
            AND pesq.event_type = 'PESQUISA_LEAD'
        )
),

-- Mapeamento direto de cada comprador para sua resposta na pesquisa
respostas_compradores AS (
    SELECT
        c.user_id,
        sa.question_id,
        sa.question_text,
        sa.value AS resposta,
        sa.score
    FROM
        compradores c
        CROSS JOIN LATERAL (
            SELECT 
                sa.question_id,
                sa.question_text,
                sa.value,
                sa.score,
                e.event_time
            FROM 
                events e
                JOIN survey_responses sr ON sr.event_id = e.event_id
                JOIN survey_answers sa ON sa.survey_response_id = sr.id
            WHERE 
                e.user_id = c.user_id
                AND e.event_type = 'PESQUISA_LEAD'
                AND (sa.question_id = (SELECT filtro_pergunta FROM parametros) OR (SELECT filtro_pergunta FROM parametros) IS NULL)
            ORDER BY 
                e.event_time DESC
            LIMIT 1
        ) sa
),

-- Todas as respostas de pesquisa no período especificado
todas_respostas AS (
    SELECT
        sa.question_id,
        sa.question_text,
        sa.value AS resposta,
        sa.score,
        e.user_id
    FROM
        survey_answers sa
        JOIN survey_responses sr ON sa.survey_response_id = sr.id
        JOIN events e ON sr.event_id = e.event_id
    WHERE
        e.event_type = 'PESQUISA_LEAD'
        AND e.funnel_id = (SELECT funnel_id FROM parametros)
        AND e.event_time BETWEEN (SELECT pesquisa_inicio FROM parametros) AND (SELECT pesquisa_fim FROM parametros)
        AND (sa.question_id = (SELECT filtro_pergunta FROM parametros) OR (SELECT filtro_pergunta FROM parametros) IS NULL)
),

-- Agrupamento de respostas por pergunta e alternativa
respostas_agrupadas AS (
    SELECT
        question_id,
        question_text,
        resposta,
        score,
        COUNT(DISTINCT user_id) AS total_respondentes
    FROM
        todas_respostas
    GROUP BY
        question_id, question_text, resposta, score
),

-- Contagem de vendas por resposta (usando o mapeamento direto)
vendas_por_resposta AS (
    SELECT
        question_id,
        question_text,
        resposta,
        score,
        COUNT(DISTINCT user_id) AS total_compradores
    FROM
        respostas_compradores
    GROUP BY
        question_id, question_text, resposta, score
),

-- Totais por pergunta para cálculo de percentuais
totais_pergunta AS (
    SELECT
        question_id,
        SUM(total_respondentes) AS total_respostas_pergunta
    FROM
        respostas_agrupadas
    GROUP BY
        question_id
),

-- Total geral de vendas para referência
total_vendas AS (
    SELECT COUNT(DISTINCT user_id) AS total FROM compradores
)

-- Resultado final
SELECT
    ra.question_id,
    ra.question_text,
    ra.score AS score_peso,
    ra.resposta,
    ra.total_respondentes AS num_respostas,
    ROUND((ra.total_respondentes * 100.0 / tp.total_respostas_pergunta)::numeric, 2) AS participacao_percentual,
    COALESCE(vpr.total_compradores, 0) AS num_vendas,
    CASE 
        WHEN ra.total_respondentes > 0 THEN 
            ROUND((COALESCE(vpr.total_compradores, 0) * 100.0 / ra.total_respondentes)::numeric, 2)
        ELSE 0 
    END AS taxa_conversao_percentual,
    -- Percentual em relação ao total de vendas
    ROUND((COALESCE(vpr.total_compradores, 0) * 100.0 / (SELECT total FROM total_vendas))::numeric, 2) AS percentual_do_total_vendas
FROM
    respostas_agrupadas ra
    JOIN totais_pergunta tp ON ra.question_id = tp.question_id
    LEFT JOIN vendas_por_resposta vpr ON 
        ra.question_id = vpr.question_id AND 
        ra.resposta = vpr.resposta
    CROSS JOIN total_vendas
ORDER BY
    ra.question_id, 
    num_vendas DESC, 
    participacao_percentual DESC;
```

### Estrutura JSON de Resposta

A consulta acima produz dados que são formatados em uma estrutura JSON hierárquica, agrupando respostas por pergunta para facilitar a análise:

```json
[
  {
    "pergunta_id": "question_0007",
    "texto_pergunta": "Você tem OAB?",
    "respostas": [
      {
        "opcao_id": "opcao_1", 
        "texto_opcao": "Não",
        "score_peso": 0,
        "num_respostas": 1042,
        "percentual_participacao": 63.38,
        "num_vendas": 16,
        "taxa_conversao_percentual": 1.54,
        "percentual_vendas": 64.00
      },
      {
        "opcao_id": "opcao_2",
        "texto_opcao": "Sim",
        "score_peso": 0,
        "num_respostas": 602,
        "percentual_participacao": 36.62,
        "num_vendas": 9,
        "taxa_conversao_percentual": 1.50,
        "percentual_vendas": 36.00
      }
    ]
  }
]
```

### Valor Analítico

Esta análise oferece diversos insights valiosos:

1. **Segmentação de leads** - Identifica grupos de alto valor com base nas respostas
2. **Otimização de perguntas** - Determina quais perguntas são mais preditivas de conversão
3. **Direcionamento de marketing** - Permite mensagens personalizadas com base em padrões de resposta
4. **Refino de produto** - Identifica características valorizadas pelos compradores

Estes insights podem ser usados para melhorar significativamente taxas de conversão e a eficiência do funil de vendas.
