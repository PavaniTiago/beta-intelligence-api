# Beta Intelligence API

API em Go para o projeto Beta Intelligence.

## Requisitos

- Docker
- Docker Compose (opcional, para desenvolvimento local)
- Conta no Google Cloud Platform (para deploy)

## Desenvolvimento Local

### Usando Docker Compose

1. Clone o repositório:
   ```bash
   git clone https://github.com/PavaniTiago/beta-intelligence-api.git
   cd beta-intelligence-api
   ```

2. Certifique-se de que o arquivo `.env` está configurado corretamente:
   ```
   DATABASE_URL=postgresql://usuario:senha@host:porta/banco
   GORM_DSN=postgresql://usuario:senha@host:porta/banco
   PORT=8080
   FRONTEND_URL=http://localhost:3000
   ```

3. Execute a aplicação com Docker Compose:
   ```bash
   docker-compose up --build
   ```

4. A API estará disponível em `http://localhost:8080`

### Usando Docker diretamente

1. Construa a imagem Docker:
   ```bash
   docker build -t beta-intelligence-api .
   ```

2. Execute o container:
   ```bash
   docker run -p 8080:8080 --env-file .env beta-intelligence-api
   ```

## Deploy no Google Cloud Platform (GCP)

### Preparação

1. Instale e configure a Google Cloud CLI:
   ```bash
   # Instale a gcloud CLI seguindo as instruções em:
   # https://cloud.google.com/sdk/docs/install
   
   # Faça login na sua conta Google
   gcloud auth login
   
   # Configure o projeto
   gcloud config set project SEU_ID_DO_PROJETO
   ```

### Opção 1: Deploy usando Artifact Registry e Compute Engine

1. Crie um repositório no Artifact Registry:
   ```bash
   gcloud artifacts repositories create beta-intelligence-repo \
     --repository-format=docker \
     --location=us-central1 \
     --description="Repositório para a API Beta Intelligence"
   ```

2. Configure o Docker para usar o Artifact Registry:
   ```bash
   gcloud auth configure-docker us-central1-docker.pkg.dev
   ```

3. Construa e envie a imagem para o Artifact Registry:
   ```bash
   docker build -t us-central1-docker.pkg.dev/SEU_ID_DO_PROJETO/beta-intelligence-repo/beta-intelligence-api:latest .
   docker push us-central1-docker.pkg.dev/SEU_ID_DO_PROJETO/beta-intelligence-repo/beta-intelligence-api:latest
   ```

4. Crie uma VM no Compute Engine:
   ```bash
   gcloud compute instances create-with-container beta-intelligence-vm \
     --container-image=us-central1-docker.pkg.dev/SEU_ID_DO_PROJETO/beta-intelligence-repo/beta-intelligence-api:latest \
     --machine-type=e2-small \
     --zone=us-central1-a \
     --tags=http-server \
     --container-env=DATABASE_URL=postgresql://usuario:senha@host:porta/banco,GORM_DSN=postgresql://usuario:senha@host:porta/banco,PORT=8080,FRONTEND_URL=https://seu-frontend.com
   ```

5. Configure o firewall para permitir tráfego HTTP:
   ```bash
   gcloud compute firewall-rules create allow-http \
     --allow=tcp:8080 \
     --target-tags=http-server \
     --description="Permitir tráfego HTTP na porta 8080"
   ```

### Opção 2: Deploy usando Cloud Run (Serverless)

1. Construa e envie a imagem para o Artifact Registry:
   ```bash
   docker build -t us-central1-docker.pkg.dev/SEU_ID_DO_PROJETO/beta-intelligence-repo/beta-intelligence-api:latest .
   docker push us-central1-docker.pkg.dev/SEU_ID_DO_PROJETO/beta-intelligence-repo/beta-intelligence-api:latest
   ```

2. Deploy no Cloud Run:
   ```bash
   gcloud run deploy beta-intelligence-api \
     --image=us-central1-docker.pkg.dev/SEU_ID_DO_PROJETO/beta-intelligence-repo/beta-intelligence-api:latest \
     --platform=managed \
     --region=us-central1 \
     --allow-unauthenticated \
     --set-env-vars=DATABASE_URL=postgresql://usuario:senha@host:porta/banco,GORM_DSN=postgresql://usuario:senha@host:porta/banco,FRONTEND_URL=https://seu-frontend.com
   ```

## Notas Importantes

- Certifique-se de substituir os valores de exemplo (como `SEU_ID_DO_PROJETO`, `usuario`, `senha`, etc.) pelos valores reais.
- Para ambientes de produção, considere usar o Secret Manager do GCP para gerenciar variáveis de ambiente sensíveis.
- Considere configurar um balanceador de carga e HTTPS para ambientes de produção. 