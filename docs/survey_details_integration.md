# Survey Details API Integration Guide

This document provides practical examples for integrating with the survey details endpoint (`/metrics/surveys/:id`), which offers in-depth analysis of survey responses and conversion metrics for frontend applications.

## Endpoint Overview

```
GET /metrics/surveys/:id
```

This endpoint provides detailed analysis of a specific survey, including:
- Question-by-question breakdown
- Response statistics per question
- Conversion rates per answer option
- Profession segmentation

## Filter Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `data_inicio` | string | General start date (ISO8601 with timezone) |
| `data_fim` | string | General end date (ISO8601 with timezone) |
| `pesquisa_inicio` | string | Start date for survey responses (ISO8601 with timezone) |
| `pesquisa_fim` | string | End date for survey responses (ISO8601 with timezone) |
| `venda_inicio` | string | Start date for sales (ISO8601 with timezone) |
| `venda_fim` | string | End date for sales (ISO8601 with timezone) |

## Example Requests

### Basic Request (No Filters)

```bash
curl -X GET "http://localhost:8080/metrics/surveys/1" -H "Content-Type: application/json"
```

### Filtering by Date Range

```bash
curl -X GET "http://localhost:8080/metrics/surveys/1?data_inicio=2025-04-01T00:00:00-03:00&data_fim=2025-04-30T23:59:59-03:00" -H "Content-Type: application/json"
```

### Analyzing Recent Responses Only

```bash
curl -X GET "http://localhost:8080/metrics/surveys/1?pesquisa_inicio=2025-04-15T00:00:00-03:00" -H "Content-Type: application/json"
```

### Analyzing Conversion for Specific Sales Period

```bash
curl -X GET "http://localhost:8080/metrics/surveys/1?venda_inicio=2025-04-22T00:00:00-03:00&venda_fim=2025-04-30T23:59:59-03:00" -H "Content-Type: application/json"
```

## Example Response

```json
[
  {
    "pergunta_id": "question_0001",
    "texto_pergunta": "Qual sua faixa etária?",
    "profissao": "Advogado",
    "respostas": [
      {
        "texto_opcao": "26 a 35 anos",
        "score_peso": 10,
        "num_respostas": 3680,
        "percentual_participacao": 44.48,
        "num_vendas": 38,
        "taxa_conversao_percentual": 1.03,
        "percentual_vendas": 46.34
      },
      {
        "texto_opcao": "18 a 25 anos",
        "score_peso": 5,
        "num_respostas": 2450,
        "percentual_participacao": 29.63,
        "num_vendas": 12,
        "taxa_conversao_percentual": 0.49,
        "percentual_vendas": 14.63
      },
      {
        "texto_opcao": "36 a 45 anos",
        "score_peso": 8,
        "num_respostas": 1652,
        "percentual_participacao": 19.96,
        "num_vendas": 25,
        "taxa_conversao_percentual": 1.51,
        "percentual_vendas": 30.49
      },
      {
        "texto_opcao": "Acima de 45 anos",
        "score_peso": 6,
        "num_respostas": 469,
        "percentual_participacao": 5.93,
        "num_vendas": 7,
        "taxa_conversao_percentual": 1.49,
        "percentual_vendas": 8.54
      }
    ]
  },
  {
    "pergunta_id": "question_0002",
    "texto_pergunta": "Qual seu objetivo ao responder esta pesquisa?",
    "profissao": "Advogado",
    "respostas": [
      {
        "texto_opcao": "Conhecer mais sobre o produto",
        "score_peso": 3,
        "num_respostas": 4125,
        "percentual_participacao": 49.86,
        "num_vendas": 42,
        "taxa_conversao_percentual": 1.02,
        "percentual_vendas": 51.22
      },
      {
        "texto_opcao": "Estou procurando uma solução para meu problema",
        "score_peso": 10,
        "num_respostas": 3251,
        "percentual_participacao": 39.30,
        "num_vendas": 35,
        "taxa_conversao_percentual": 1.08,
        "percentual_vendas": 42.68
      },
      {
        "texto_opcao": "Apenas curiosidade",
        "score_peso": 1,
        "num_respostas": 875,
        "percentual_participacao": 10.84,
        "num_vendas": 5,
        "taxa_conversao_percentual": 0.57,
        "percentual_vendas": 6.10
      }
    ]
  }
  // More questions...
]
```

## Response Fields Explained

Each question contains:

- **pergunta_id**: Unique identifier for the question (e.g., "question_0001")
- **texto_pergunta**: The text of the question
- **profissao**: Profession associated with this data
- **respostas**: Array of response options with metrics

Each response option contains:

| Field | Description |
|-------|-------------|
| **texto_opcao** | The text of the answer option |
| **score_peso** | The score weight assigned to this option (for lead scoring) |
| **num_respostas** | Number of respondents who chose this option |
| **percentual_participacao** | Percentage of total respondents who chose this option |
| **num_vendas** | Number of sales from people who chose this option |
| **taxa_conversao_percentual** | Conversion rate: percentage of people who chose this option and made a purchase |
| **percentual_vendas** | Percentage of total sales from people who chose this option |

## Frontend Integration Examples

### Displaying Questions and Answers with React

```jsx
function SurveyAnalysis({ surveyId }) {
  const [surveyData, setSurveyData] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  
  useEffect(() => {
    async function fetchSurveyDetails() {
      try {
        setIsLoading(true);
        const response = await fetch(`/api/metrics/surveys/${surveyId}`);
        
        if (!response.ok) {
          throw new Error('Failed to fetch survey details');
        }
        
        const data = await response.json();
        setSurveyData(data);
      } catch (err) {
        setError(err.message);
      } finally {
        setIsLoading(false);
      }
    }
    
    fetchSurveyDetails();
  }, [surveyId]);
  
  if (isLoading) return <div>Loading survey details...</div>;
  if (error) return <div>Error: {error}</div>;
  if (surveyData.length === 0) return <div>No data available</div>;
  
  return (
    <div className="survey-analysis">
      <h1>Survey Analysis</h1>
      
      {surveyData.map(question => (
        <div key={question.pergunta_id} className="question-card">
          <h2>{question.texto_pergunta}</h2>
          <p className="profession">Profession: {question.profissao}</p>
          
          <table className="responses-table">
            <thead>
              <tr>
                <th>Answer Option</th>
                <th>Responses</th>
                <th>% of Total</th>
                <th>Sales</th>
                <th>Conversion %</th>
                <th>% of Sales</th>
              </tr>
            </thead>
            <tbody>
              {question.respostas.map((answer, index) => (
                <tr key={index} className={index === 0 ? 'top-answer' : ''}>
                  <td>{answer.texto_opcao}</td>
                  <td>{answer.num_respostas.toLocaleString()}</td>
                  <td>{answer.percentual_participacao}%</td>
                  <td>{answer.num_vendas.toLocaleString()}</td>
                  <td>{answer.taxa_conversao_percentual.toFixed(2)}%</td>
                  <td>{answer.percentual_vendas.toFixed(2)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </div>
  );
}
```

### Create a Visualization Chart

```jsx
import { Bar } from 'react-chartjs-2';

function ConversionRateChart({ question }) {
  const labels = question.respostas.map(r => r.texto_opcao);
  const conversionRates = question.respostas.map(r => r.taxa_conversao_percentual);
  const responseCount = question.respostas.map(r => r.num_respostas);
  
  const data = {
    labels,
    datasets: [
      {
        label: 'Conversion Rate (%)',
        data: conversionRates,
        backgroundColor: 'rgba(53, 162, 235, 0.5)',
        borderColor: 'rgba(53, 162, 235, 1)',
        borderWidth: 1
      }
    ]
  };
  
  const options = {
    indexAxis: 'y',
    plugins: {
      title: {
        display: true,
        text: question.texto_pergunta,
        font: { size: 16 }
      },
      tooltip: {
        callbacks: {
          afterLabel: function(context) {
            const index = context.dataIndex;
            return `Responses: ${responseCount[index].toLocaleString()}`;
          }
        }
      }
    }
  };
  
  return (
    <div className="chart-container">
      <Bar data={data} options={options} />
    </div>
  );
}
```

## Date Range Filtering Component

```jsx
function DateRangeFilter({ onApplyFilters }) {
  const [startDate, setStartDate] = useState(null);
  const [endDate, setEndDate] = useState(null);
  const [filterType, setFilterType] = useState('all'); // 'all', 'survey', 'sale'
  
  const handleApplyFilters = () => {
    if (!startDate || !endDate) return;
    
    // Format dates to ISO8601 with Brasilia timezone
    const formatDate = (date) => {
      const isoDate = date.toISOString();
      return isoDate.replace('Z', '-03:00');
    };
    
    const filters = {};
    
    if (filterType === 'all') {
      filters.data_inicio = formatDate(startDate);
      filters.data_fim = formatDate(endDate);
    } else if (filterType === 'survey') {
      filters.pesquisa_inicio = formatDate(startDate);
      filters.pesquisa_fim = formatDate(endDate);
    } else if (filterType === 'sale') {
      filters.venda_inicio = formatDate(startDate);
      filters.venda_fim = formatDate(endDate);
    }
    
    onApplyFilters(filters);
  };
  
  return (
    <div className="date-filter">
      <div className="filter-type">
        <label>
          <input 
            type="radio" 
            value="all" 
            checked={filterType === 'all'} 
            onChange={() => setFilterType('all')} 
          />
          All Events
        </label>
        <label>
          <input 
            type="radio" 
            value="survey" 
            checked={filterType === 'survey'} 
            onChange={() => setFilterType('survey')} 
          />
          Survey Responses Only
        </label>
        <label>
          <input 
            type="radio" 
            value="sale" 
            checked={filterType === 'sale'} 
            onChange={() => setFilterType('sale')} 
          />
          Sales Only
        </label>
      </div>
      
      <div className="date-inputs">
        <div>
          <label>Start Date</label>
          <input 
            type="date" 
            onChange={(e) => setStartDate(new Date(e.target.value))} 
          />
        </div>
        <div>
          <label>End Date</label>
          <input 
            type="date" 
            onChange={(e) => setEndDate(new Date(e.target.value))} 
          />
        </div>
      </div>
      
      <button onClick={handleApplyFilters}>Apply Filters</button>
    </div>
  );
}
```

## Advanced Analysis Use Cases

1. **Identify High-Converting Answers**
   - Look for options with high `taxa_conversao_percentual` to understand what resonates with buyers

2. **Optimize Lead Scoring**
   - Compare `score_peso` with actual conversion rates to calibrate your scoring algorithm

3. **Segment Analysis by Profession**
   - Filter or group by `profissao` to understand different audience behaviors

4. **Identify Effective Questions**
   - Questions where answers have widely varying conversion rates are more predictive of buying behavior

5. **Time-based Trend Analysis**
   - Use date filters to compare periods before and after marketing changes

## Best Practices

1. **Cache Results** - The survey details endpoint returns comprehensive data that can be cached on the frontend for better performance

2. **Progressive Loading** - If displaying many questions, consider implementing progressive loading or pagination

3. **Highlight Top Performers** - Visually highlight answer options with highest conversion rates or response counts

4. **Use Relative Comparison** - When displaying metrics, provide context (e.g., "2x higher than average")

5. **Filter Combinations** - Allow users to combine multiple filter types for deeper analysis 