# Atualização da API de Pesquisa - Ciclo de Webinar

## Introdução

A API de Pesquisa foi atualizada para suportar o conceito de ciclo de webinar, que consiste em duas fases principais:

1. **Fase de Captação**: Período em que os usuários podem responder à pesquisa
2. **Fase de Vendas**: Período em que as vendas são realizadas após o término da pesquisa

## Regras do Ciclo de Webinar

### Cronograma Fixo

O ciclo de webinar segue um cronograma fixo, sempre baseado no horário de Brasília:

- **Início da captação**: Terça-feira às 20:00
- **Fim da captação**: Terça-feira seguinte às 20:00
- **Início das vendas**: Terça-feira (mesmo dia do fim da captação) às 20:30
- **Fim das vendas**: Terça-feira (mesmo dia do início das vendas) às 23:59:59

### Exemplo

```
Início da captação: 29/04/2025 às 20:00
Fim da captação: 06/05/2025 às 20:00
Início das vendas: 06/05/2025 às 20:30
Fim das vendas: 06/05/2025 às 23:59:59
```

## Filtros Disponíveis

A API agora aceita quatro filtros de data específicos:

1. `pesquisa_inicio`: Data de início da fase de captação (Terça-feira às 20:00)
2. `pesquisa_fim`: Data de fim da fase de captação (Terça-feira às 20:00)
3. `venda_inicio`: Data de início da fase de vendas (Terça-feira às 20:30)
4. `venda_fim`: Data de fim da fase de vendas (Terça-feira às 23:59:59)

### Validações e Restrições

Todos os filtros de data estão sujeitos a validações rígidas:

- **Dia da semana**: Todos os filtros devem ser aplicados a uma terça-feira
- **Horários específicos**:
  - `pesquisa_inicio`: Deve ser às 20:00
  - `pesquisa_fim`: Deve ser às 20:00
  - `venda_inicio`: Deve ser às 20:30
  - `venda_fim`: Deve ser às 23:59:59
- **Consistência**: Se `pesquisa_fim` e `venda_inicio` forem fornecidos, devem ser do mesmo dia

## Exemplos de Uso da API

### Requisição Válida

```http
GET /metrics/surveys?pesquisa_inicio=2025-04-29T20:00:00-03:00&pesquisa_fim=2025-05-06T20:00:00-03:00&venda_inicio=2025-05-06T20:30:00-03:00&venda_fim=2025-05-06T23:59:59-03:00
```

### Requisições Inválidas

#### Dia da semana incorreto
```http
GET /metrics/surveys?pesquisa_inicio=2025-04-30T20:00:00-03:00
```

Resposta:
```json
{
  "error": "Data de início de pesquisa inválida. As pesquisas sempre iniciam às terças-feiras às 20:00 do horário de Brasília."
}
```

#### Horário incorreto
```http
GET /metrics/surveys?venda_inicio=2025-05-06T21:00:00-03:00
```

Resposta:
```json
{
  "error": "Data de início de vendas inválida. As vendas sempre iniciam às terças-feiras às 20:30 do horário de Brasília."
}
```

#### Inconsistência entre datas
```http
GET /metrics/surveys?pesquisa_fim=2025-05-06T20:00:00-03:00&venda_inicio=2025-05-07T20:30:00-03:00
```

Resposta:
```json
{
  "error": "Inconsistência nos filtros de data. O fim da pesquisa e o início das vendas devem ocorrer no mesmo dia (terça-feira)."
}
```

## Implementação no Frontend

### Componentes de Data/Hora Sugeridos

1. **Seletor de Ciclo de Webinar**:
   - Implementar um componente que permita selecionar apenas terças-feiras válidas
   - Ao selecionar uma terça-feira, preencher automaticamente os quatro filtros com os horários corretos

2. **Adaptação dos Datepickers existentes**:
   - Restringir a seleção para apenas terças-feiras
   - Fixar os horários conforme as regras (20:00, 20:30, 23:59:59)
   - Adicionar validação para garantir consistência entre fim da pesquisa e início das vendas

### Tratamento de Erros

É importante implementar o tratamento dos novos erros de validação:

```javascript
// Exemplo de função para tratamento de erros
function handleApiError(error) {
  if (error.response && error.response.data && error.response.data.error) {
    const errorMessage = error.response.data.error;
    
    // Identificar tipo de erro baseado na mensagem
    if (errorMessage.includes('terças-feiras')) {
      // Erro de dia da semana
      showDateSelectionError('Selecione apenas terças-feiras para os filtros de data.');
    } else if (errorMessage.includes('Inconsistência nos filtros')) {
      // Erro de inconsistência entre datas
      showDateConsistencyError('O fim da pesquisa e início das vendas devem ser no mesmo dia.');
    } else {
      // Outros erros
      showGenericError(errorMessage);
    }
  } else {
    showGenericError('Ocorreu um erro ao processar sua solicitação.');
  }
}
```

## Dicas de Implementação

1. **Pré-preencher horários**: Ao selecionar uma data, preencha automaticamente o horário adequado.
2. **Validação client-side**: Implemente validações no frontend para evitar chamadas à API com datas inválidas.
3. **Feedback visual**: Forneça feedback claro sobre as restrições de data ao usuário durante a seleção.
4. **Seleção simplificada**: Considere implementar um seletor simples de "semana de webinar" que configure automaticamente os quatro filtros.

## Endpoints Afetados

Esta atualização afeta os seguintes endpoints:

- `/metrics/surveys` - Métricas gerais de pesquisas
- `/metrics/surveys/{id}` - Detalhes de uma pesquisa específica

## Exemplos de Integração

### React com date-fns

```jsx
import { format, isTuesday, setHours, setMinutes, setSeconds } from 'date-fns';
import { ptBR } from 'date-fns/locale';

function WebinarDatePicker({ onChange }) {
  const handleDateChange = (date) => {
    if (!isTuesday(date)) {
      alert('Por favor, selecione uma terça-feira');
      return;
    }
    
    // Configurar datas do ciclo de webinar
    const pesquisaInicio = setSeconds(setMinutes(setHours(date, 20), 0), 0);
    const pesquisaFim = setSeconds(setMinutes(setHours(date, 20), 0), 0);
    const vendaInicio = setSeconds(setMinutes(setHours(date, 20), 30), 0);
    const vendaFim = setSeconds(setMinutes(setHours(date, 23), 59), 59);
    
    onChange({
      pesquisaInicio,
      pesquisaFim,
      vendaInicio,
      vendaFim
    });
  };
  
  return (
    <div>
      <label>Selecione a terça-feira do webinar:</label>
      <DatePicker
        selected={selectedDate}
        onChange={handleDateChange}
        filterDate={isTuesday}
        dateFormat="dd/MM/yyyy"
        locale={ptBR}
      />
    </div>
  );
}
```

## Suporte

Em caso de dúvidas sobre a implementação ou problemas com a integração, entre em contato com a equipe de desenvolvimento da API.
