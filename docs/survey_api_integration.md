# Survey API Integration Guide with Next.js App Router

This document explains how to integrate with the Survey API endpoints from a Next.js application using the App Router architecture, including available filtering options and practical examples.

## API Endpoints

The Survey API offers three main endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/surveys` | GET | List all surveys with optional filtering and pagination |
| `/metrics/surveys` | GET | Get aggregated metrics for surveys with various filtering options |
| `/metrics/surveys/:id` | GET | Get detailed analysis of a specific survey, including question-level metrics |

## Integration with Next.js API Routes

Using Next.js API routes provides several advantages:
- Keep API keys and credentials secure on the server
- Simplify error handling for the client
- Avoid potential CORS issues
- Transform and format data before sending to the client

### Creating Next.js API Routes with App Router

First, create API route handlers in your Next.js application using the App Router structure:

#### 1. List Surveys API Route

```javascript
// app/api/surveys/route.js
import { NextResponse } from 'next/server';

export async function GET(request) {
  // Get URL and query parameters
  const { searchParams } = new URL(request.url);
  
  const page = searchParams.get('page') || '1';
  const limit = searchParams.get('limit') || '10';
  const funnel_id = searchParams.get('funnel_id');
  const survey_id = searchParams.get('survey_id');
  const include_funnel = searchParams.get('include_funnel');
  
  try {
    // Build query parameters
    const queryParams = new URLSearchParams();
    if (page) queryParams.append('page', page);
    if (limit) queryParams.append('limit', limit);
    if (funnel_id) queryParams.append('funnel_id', funnel_id);
    if (survey_id) queryParams.append('survey_id', survey_id);
    if (include_funnel) queryParams.append('include_funnel', include_funnel);
    
    // Call the external API
    const response = await fetch(`${process.env.API_BASE_URL}/surveys?${queryParams.toString()}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        // Add any required API keys or authentication headers here
        'Authorization': `Bearer ${process.env.API_TOKEN}`
      }
    });
    
    if (!response.ok) {
      // Forward the error status from the external API
      return NextResponse.json(
        { error: 'Failed to fetch surveys' },
        { status: response.status }
      );
    }
    
    const data = await response.json();
    
    // Return the data to the client
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching surveys:', error);
    return NextResponse.json(
      { error: 'Failed to fetch surveys' },
      { status: 500 }
    );
  }
}
```

#### 2. Survey Metrics API Route

```javascript
// app/api/metrics/surveys/route.js
import { NextResponse } from 'next/server';

export async function GET(request) {
  // Get URL and query parameters
  const { searchParams } = new URL(request.url);
  
  // Create an object from searchParams
  const params = {};
  for (const [key, value] of searchParams.entries()) {
    params[key] = value;
  }
  
  try {
    // Build query parameters for the external API
    const queryParams = new URLSearchParams();
    
    // Add any provided filters to query params
    Object.entries(params).forEach(([key, value]) => {
      if (value) queryParams.append(key, value);
    });
    
    // Call the external API
    const response = await fetch(`${process.env.API_BASE_URL}/metrics/surveys?${queryParams.toString()}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${process.env.API_TOKEN}`
      }
    });
    
    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch survey metrics' },
        { status: response.status }
      );
    }
    
    const data = await response.json();
    
    // Return the data to the client
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching survey metrics:', error);
    return NextResponse.json(
      { error: 'Failed to fetch survey metrics' },
      { status: 500 }
    );
  }
}
```

#### 3. Survey Details API Route

```javascript
// app/api/metrics/surveys/[id]/route.js
import { NextResponse } from 'next/server';

export async function GET(request, { params }) {
  // Get the ID from the route parameters
  const id = params.id;
  
  // Get URL and query parameters
  const { searchParams } = new URL(request.url);
  
  // Create an object from searchParams
  const filterParams = {};
  for (const [key, value] of searchParams.entries()) {
    filterParams[key] = value;
  }
  
  try {
    // Build query parameters
    const queryParams = new URLSearchParams();
    
    // Add any provided filters to query params
    Object.entries(filterParams).forEach(([key, value]) => {
      if (value) queryParams.append(key, value);
    });
    
    // Call the external API
    const response = await fetch(`${process.env.API_BASE_URL}/metrics/surveys/${id}?${queryParams.toString()}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${process.env.API_TOKEN}`
      }
    });
    
    if (!response.ok) {
      return NextResponse.json(
        { error: `Failed to fetch survey details for ID ${id}` },
        { status: response.status }
      );
    }
    
    const data = await response.json();
    
    // Return the data to the client
    return NextResponse.json(data);
  } catch (error) {
    console.error(`Error fetching survey details for ID ${id}:`, error);
    return NextResponse.json(
      { error: `Failed to fetch survey details for ID ${id}` },
      { status: 500 }
    );
  }
}
```

### Calling Next.js API Routes from the Frontend

Now that you have created the API routes, you can call them from your frontend components:

#### Fetching Surveys from Next.js API

```javascript
async function fetchSurveys(page = 1, limit = 10) {
  try {
    const response = await fetch(`/api/surveys?page=${page}&limit=${limit}`);
    
    if (!response.ok) {
      throw new Error('Network response was not ok');
    }
    
    return await response.json();
  } catch (error) {
    console.error('Error fetching surveys:', error);
    throw error;
  }
}
```

#### Fetching Survey Metrics from Next.js API

```javascript
async function fetchSurveyMetrics(filters = {}) {
  // Build query parameters
  const queryParams = new URLSearchParams();
  
  // Add any provided filters to query params
  Object.entries(filters).forEach(([key, value]) => {
    if (value) queryParams.append(key, value);
  });
  
  try {
    const response = await fetch(`/api/metrics/surveys?${queryParams.toString()}`);
    
    if (!response.ok) {
      throw new Error('Network response was not ok');
    }
    
    return await response.json();
  } catch (error) {
    console.error('Error fetching survey metrics:', error);
    throw error;
  }
}
```

#### Fetching Survey Details from Next.js API

```javascript
async function fetchSurveyDetails(surveyId, filters = {}) {
  // Build query parameters
  const queryParams = new URLSearchParams();
  
  // Add any provided filters to query params
  Object.entries(filters).forEach(([key, value]) => {
    if (value) queryParams.append(key, value);
  });
  
  try {
    const response = await fetch(`/api/metrics/surveys/${surveyId}?${queryParams.toString()}`);
    
    if (!response.ok) {
      throw new Error('Network response was not ok');
    }
    
    return await response.json();
  } catch (error) {
    console.error('Error fetching survey details:', error);
    throw error;
  }
}
```

## Filter Parameters

### Survey Listing Filters

The `/api/surveys` endpoint supports the following filters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | integer | Page number for pagination (default: 1) |
| `limit` | integer | Number of items per page (default: 10) |
| `funnel_id` | integer | Filter by funnel ID |
| `survey_id` | integer | Filter by survey ID |
| `include_funnel` | boolean | Include funnel details in response (default: false) |

### Survey Metrics Filters

The `/api/metrics/surveys` endpoint supports rich filtering options:

| Parameter | Type | Description |
|-----------|------|-------------|
| `data_inicio` | string | General start date (ISO8601 with timezone) |
| `data_fim` | string | General end date (ISO8601 with timezone) |
| `lead_inicio` | string | Start date for leads (ISO8601 with timezone) |
| `lead_fim` | string | End date for leads (ISO8601 with timezone) |
| `pesquisa_inicio` | string | Start date for survey responses (ISO8601 with timezone) |
| `pesquisa_fim` | string | End date for survey responses (ISO8601 with timezone) |
| `venda_inicio` | string | Start date for sales (ISO8601 with timezone) |
| `venda_fim` | string | End date for sales (ISO8601 with timezone) |
| `profissao` | integer | Filter by profession ID |
| `funil` | integer | Filter by funnel ID |
| `pesquisa_id` | integer | Filter by survey ID |

### Survey Details Filters

The `/api/metrics/surveys/:id` endpoint supports the following filters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `data_inicio` | string | General start date (ISO8601 with timezone) |
| `data_fim` | string | General end date (ISO8601 with timezone) |
| `pesquisa_inicio` | string | Start date for survey responses (ISO8601 with timezone) |
| `pesquisa_fim` | string | End date for survey responses (ISO8601 with timezone) |
| `venda_inicio` | string | Start date for sales (ISO8601 with timezone) |
| `venda_fim` | string | End date for sales (ISO8601 with timezone) |

## Date Filtering

Proper date formatting is crucial for effective filtering. The API accepts the following date formats:

1. ISO8601 with timezone (recommended): `2025-04-15T00:00:00-03:00`
2. Simple date format: `2025-04-15` (automatically converts to 00:00:00 for start dates and 23:59:59 for end dates)
3. Date and time without timezone: `2025-04-15T00:00:00` (uses server timezone)

### Example: Filtering by Date Range in React

```jsx
'use client';

import React, { useState } from 'react';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';

function SurveyMetricsComponent() {
  const [startDate, setStartDate] = useState(new Date());
  const [endDate, setEndDate] = useState(new Date());
  const [metrics, setMetrics] = useState([]);
  const [loading, setLoading] = useState(false);

  const fetchMetrics = async () => {
    setLoading(true);
    try {
      // Format dates to ISO8601 with Brasilia timezone (-03:00)
      const startDateStr = startDate.toISOString().replace('Z', '-03:00');
      const endDateStr = endDate.toISOString().replace('Z', '-03:00');
      
      const filters = {
        data_inicio: startDateStr,
        data_fim: endDateStr,
      };
      
      const queryParams = new URLSearchParams();
      Object.entries(filters).forEach(([key, value]) => {
        queryParams.append(key, value);
      });
      
      const response = await fetch(`/api/metrics/surveys?${queryParams.toString()}`);
      const data = await response.json();
      setMetrics(data);
    } catch (error) {
      console.error('Error fetching metrics:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>Survey Metrics</h2>
      
      <div className="filters">
        <div>
          <label>Start Date:</label>
          <DatePicker
            selected={startDate}
            onChange={date => setStartDate(date)}
            selectsStart
            startDate={startDate}
            endDate={endDate}
          />
        </div>
        
        <div>
          <label>End Date:</label>
          <DatePicker
            selected={endDate}
            onChange={date => setEndDate(date)}
            selectsEnd
            startDate={startDate}
            endDate={endDate}
            minDate={startDate}
          />
        </div>
        
        <button onClick={fetchMetrics} disabled={loading}>
          {loading ? 'Loading...' : 'Apply Filters'}
        </button>
      </div>
      
      {metrics.length > 0 ? (
        <table>
          <thead>
            <tr>
              <th>Survey Name</th>
              <th>Funnel</th>
              <th>Total Leads</th>
              <th>Total Responses</th>
              <th>Total Sales</th>
              <th>Response Rate</th>
              <th>Conversion Rate</th>
            </tr>
          </thead>
          <tbody>
            {metrics.map((metric, index) => (
              <tr key={index}>
                <td>{metric.nome_pesquisa}</td>
                <td>{metric.funil}</td>
                <td>{metric.total_leads}</td>
                <td>{metric.total_respostas}</td>
                <td>{metric.total_vendas}</td>
                <td>{(metric.taxa_resposta * 100).toFixed(2)}%</td>
                <td>{(metric.conversao_vendas * 100).toFixed(2)}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p>No metrics available. Apply filters to see data.</p>
      )}
    </div>
  );
}

export default SurveyMetricsComponent;
```

## Advanced Filtering Scenarios

### 1. Monitoring Campaign Performance

To monitor leads and responses from a specific marketing campaign that started on April 15, 2025:

```javascript
const filters = {
  lead_inicio: '2025-04-15T00:00:00-03:00',
  lead_fim: '2025-04-30T23:59:59-03:00'
};

const campaignMetrics = await fetchSurveyMetrics(filters);
```

### 2. Conversion Analysis by Period

To identify if recent changes in the sales script improved conversion after April 22, 2025:

```javascript
const filters = {
  pesquisa_inicio: '2025-04-01T00:00:00-03:00',
  venda_inicio: '2025-04-22T00:00:00-03:00'
};

const conversionMetrics = await fetchSurveyMetrics(filters);
```

### 3. Comparing Before/After Product Update

To compare survey performance before and after a product update on April 20, 2025:

```javascript
// Before update
const beforeFilters = {
  venda_fim: '2025-04-19T23:59:59-03:00'
};

// After update
const afterFilters = {
  venda_inicio: '2025-04-20T00:00:00-03:00'
};

const beforeMetrics = await fetchSurveyDetails(1, beforeFilters);
const afterMetrics = await fetchSurveyDetails(1, afterFilters);

// Now you can compare the metrics
```

### 4. Funnel-specific Analysis

To analyze metrics for a specific funnel:

```javascript
const filters = {
  funil: 9,
  data_inicio: '2025-04-01T00:00:00-03:00',
  data_fim: '2025-04-30T23:59:59-03:00'
};

const funnelMetrics = await fetchSurveyMetrics(filters);
```

## Rendering Survey Details Analysis

The `/api/metrics/surveys/:id` endpoint returns detailed data about questions and answers with conversion metrics. Here's how to display this in a React component:

```jsx
'use client';

import React, { useState, useEffect } from 'react';

function SurveyDetailsComponent({ surveyId }) {
  const [surveyDetails, setSurveyDetails] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetchDetails = async () => {
      setLoading(true);
      try {
        const response = await fetch(`/api/metrics/surveys/${surveyId}`);
        const data = await response.json();
        setSurveyDetails(data);
      } catch (error) {
        console.error('Error fetching survey details:', error);
      } finally {
        setLoading(false);
      }
    };
    
    fetchDetails();
  }, [surveyId]);

  if (loading) return <div>Loading survey details...</div>;
  
  if (surveyDetails.length === 0) return <div>No survey details available</div>;

  return (
    <div>
      <h2>Survey Analysis</h2>
      
      {surveyDetails.map((question) => (
        <div key={question.pergunta_id} className="question-analysis">
          <h3>{question.texto_pergunta}</h3>
          
          <table>
            <thead>
              <tr>
                <th>Option</th>
                <th>Responses</th>
                <th>% of Total</th>
                <th>Sales</th>
                <th>Conversion Rate</th>
                <th>% of Sales</th>
              </tr>
            </thead>
            <tbody>
              {question.respostas.map((answer, index) => (
                <tr key={index}>
                  <td>{answer.texto_opcao}</td>
                  <td>{answer.num_respostas}</td>
                  <td>{answer.percentual_participacao}%</td>
                  <td>{answer.num_vendas}</td>
                  <td>{answer.taxa_conversao_percentual}%</td>
                  <td>{answer.percentual_vendas}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </div>
  );
}

export default SurveyDetailsComponent;
```

## Environment Variables Setup

Store your API tokens and URLs securely using Next.js environment variables:

1. Create a `.env.local` file in your project root:
```
API_BASE_URL=https://api.example.com
API_TOKEN=your_secret_token_here
```

2. Access these variables in your API routes:
```javascript
// app/api/surveys/route.js
const apiBaseUrl = process.env.API_BASE_URL;
const apiToken = process.env.API_TOKEN;

// Use these variables when making requests
```

## Timezone Considerations

All date parameters accept ISO8601 format with timezone information. For consistency, use the Brasilia timezone (UTC-3) by adding the suffix `-03:00` to your dates.

Example: `2025-04-15T00:00:00-03:00`

When working with dates in JavaScript, be sure to format them properly before sending to the API:

```javascript
// Convert a JavaScript Date to ISO8601 with Brasilia timezone
function formatDateWithBrasiliaTimezone(date) {
  const isoDate = date.toISOString();
  return isoDate.replace('Z', '-03:00');
}

const today = new Date();
const formattedDate = formatDateWithBrasiliaTimezone(today);
```

## Error Handling

Implement proper error handling in both your API routes and frontend components:

### API Route Error Handling

```javascript
// app/api/surveys/route.js
import { NextResponse } from 'next/server';

export async function GET(request) {
  try {
    // API call code here...
    
    if (!response.ok) {
      // Forward the status code from the external API
      return NextResponse.json(
        { error: `External API error: ${response.statusText}` },
        { status: response.status }
      );
    }
    
    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Server error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
```

### Frontend Error Handling

```javascript
async function fetchSurveyData() {
  try {
    const response = await fetch('/api/metrics/surveys');
    
    if (!response.ok) {
      const errorData = await response.json();
      console.error('API error:', errorData);
      
      // Handle specific error cases
      if (response.status === 404) {
        return { error: 'No surveys found' };
      }
      
      return { error: errorData.error || 'An error occurred' };
    }
    
    return await response.json();
  } catch (error) {
    console.error('Fetch error:', error);
    return { error: 'Network error, please try again' };
  }
}
```

## Summary of API Enhancements and Integration Notes

This documentation covers all three API endpoints that were implemented and enhanced:

1. **List Surveys** (`/surveys`): Basic survey listing with pagination and optional funnel details
   - Supports filtering by funnel_id and survey_id
   - Returns survey metadata including names, descriptions, and configuration

2. **Survey Metrics** (`/metrics/surveys`): Aggregated survey performance metrics 
   - Enhanced with profession information via JOIN with products and professions tables
   - Corrected GROUP BY clause from "b.funil" to "b.funnel_name" to match the actual column names
   - Fixed SQL template handling issues by replacing Go template syntax with string concatenation

3. **Survey Details** (`/metrics/surveys/:id`): Detailed question-level analysis
   - Includes profession-specific data for more targeted analysis
   - Shows conversion rates and sales metrics per answer option
   - Allows filtering by various date ranges

All three endpoints have been thoroughly tested and confirmed to be working correctly. The SQL improvements ensure proper data retrieval without the previous template syntax errors.

When integrating with this API:
- Remember that date parameters should use ISO8601 format with timezone information (e.g., `2025-04-15T00:00:00-03:00`)
- For Next.js applications, use the App Router architecture as described in this guide
- Leverage the profession data now available in the metrics to perform more granular audience analysis
- Use proper error handling both in your API routes and frontend components

These API enhancements provide a solid foundation for building comprehensive survey analytics dashboards and integrating survey data with your marketing and sales systems.

## Conclusion

This guide provides the foundation for integrating with the Survey API in your Next.js application using the App Router architecture. By using Next.js API routes as intermediaries, you can keep your API credentials secure, simplify client-side code, and ensure a more robust implementation.

Remember to properly format date parameters and implement appropriate error handling for a robust integration.

## API Response Examples

Below are examples of the JSON responses returned by each API endpoint:

### 1. List Surveys Response (`/surveys`)

```json
{
  "surveys": [
    {
      "id": 1,
      "title": "Pesquisa de satisfação do cliente",
      "description": "Uma pesquisa para entender o nível de satisfação dos clientes com nossos produtos",
      "funnel_id": 3,
      "funnel_name": "Funil de Vendas Principal",
      "created_at": "2025-03-15T10:30:00-03:00",
      "updated_at": "2025-03-20T14:45:00-03:00",
      "status": "active",
      "questions_count": 8
    },
    {
      "id": 2,
      "title": "Perfil do advogado moderno",
      "description": "Pesquisa para identificar características e necessidades dos advogados",
      "funnel_id": 4,
      "funnel_name": "Funil Jurídico",
      "created_at": "2025-04-01T09:15:00-03:00",
      "updated_at": "2025-04-05T11:20:00-03:00",
      "status": "active",
      "questions_count": 12
    }
  ],
  "pagination": {
    "current_page": 1,
    "total_pages": 3,
    "total_items": 28,
    "items_per_page": 10
  }
}
```

### 2. Survey Metrics Response (`/metrics/surveys`)

```json
[
  {
    "survey_id": 1,
    "nome_pesquisa": "Pesquisa de satisfação do cliente",
    "funil": "Funil de Vendas Principal",
    "profissao": "Advogado",
    "total_leads": 8250,
    "total_respostas": 6453,
    "total_vendas": 82,
    "taxa_resposta": 0.7822,
    "conversao_vendas": 0.0127
  },
  {
    "survey_id": 2,
    "nome_pesquisa": "Perfil do advogado moderno",
    "funil": "Funil Jurídico",
    "profissao": "Advogado",
    "total_leads": 3450,
    "total_respostas": 2876,
    "total_vendas": 63,
    "taxa_resposta": 0.8336,
    "conversao_vendas": 0.0219
  },
  {
    "survey_id": 2,
    "nome_pesquisa": "Perfil do advogado moderno",
    "funil": "Funil Jurídico",
    "profissao": "Contador",
    "total_leads": 1240,
    "total_respostas": 987,
    "total_vendas": 18,
    "taxa_resposta": 0.7960,
    "conversao_vendas": 0.0182
  }
]
```

### 3. Survey Details Response (`/metrics/surveys/:id`)

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
]
```

These response examples show the format and type of data you can expect from each endpoint, which can be useful when designing your frontend components and data processing logic. 