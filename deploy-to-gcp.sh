#!/bin/bash

# Script para fazer deploy da aplicação Beta Intelligence no GCP

# Configurações - Edite estas variáveis
PROJECT_ID=""  # Seu ID do projeto GCP
REGION="us-central1"
REPOSITORY_NAME="beta-intelligence-repo"
IMAGE_NAME="beta-intelligence-api"
VM_NAME="beta-intelligence-vm"
MACHINE_TYPE="e2-small"
ZONE="us-central1-a"

# Verificar se o PROJECT_ID foi configurado
if [ -z "$PROJECT_ID" ]; then
  echo "Erro: Você precisa configurar o PROJECT_ID no script."
  exit 1
fi

# Função para exibir mensagens de ajuda
show_help() {
  echo "Uso: $0 [opção]"
  echo "Opções:"
  echo "  --help                Exibe esta mensagem de ajuda"
  echo "  --setup-repo          Configura o repositório no Artifact Registry"
  echo "  --build-push          Constrói e envia a imagem para o Artifact Registry"
  echo "  --deploy-vm           Faz deploy em uma VM do Compute Engine"
  echo "  --deploy-cloud-run    Faz deploy no Cloud Run (serverless)"
  echo "  --all                 Executa todas as etapas (setup-repo, build-push, deploy-vm)"
}

# Configurar o repositório no Artifact Registry
setup_repo() {
  echo "Configurando repositório no Artifact Registry..."
  
  # Verificar se o repositório já existe
  REPO_EXISTS=$(gcloud artifacts repositories list --project=$PROJECT_ID --filter="name:$REPOSITORY_NAME" --format="value(name)")
  
  if [ -z "$REPO_EXISTS" ]; then
    gcloud artifacts repositories create $REPOSITORY_NAME \
      --repository-format=docker \
      --location=$REGION \
      --description="Repositório para a API Beta Intelligence" \
      --project=$PROJECT_ID
    
    echo "Repositório criado com sucesso!"
  else
    echo "Repositório já existe, pulando criação."
  fi
  
  # Configurar Docker para usar o Artifact Registry
  gcloud auth configure-docker $REGION-docker.pkg.dev
  
  echo "Configuração do repositório concluída."
}

# Construir e enviar a imagem para o Artifact Registry
build_push() {
  echo "Construindo e enviando a imagem para o Artifact Registry..."
  
  # Construir a imagem Docker
  docker build -t $REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/$IMAGE_NAME:latest .
  
  # Enviar a imagem para o Artifact Registry
  docker push $REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/$IMAGE_NAME:latest
  
  echo "Imagem construída e enviada com sucesso!"
}

# Fazer deploy em uma VM do Compute Engine
deploy_vm() {
  echo "Fazendo deploy em uma VM do Compute Engine..."
  
  # Verificar se a VM já existe
  VM_EXISTS=$(gcloud compute instances list --project=$PROJECT_ID --filter="name=$VM_NAME" --format="value(name)")
  
  if [ -z "$VM_EXISTS" ]; then
    # Criar a VM com o container
    gcloud compute instances create-with-container $VM_NAME \
      --container-image=$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/$IMAGE_NAME:latest \
      --machine-type=$MACHINE_TYPE \
      --zone=$ZONE \
      --tags=http-server \
      --project=$PROJECT_ID
    
    echo "VM criada com sucesso!"
  else
    # Atualizar a VM existente
    gcloud compute instances update-container $VM_NAME \
      --container-image=$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/$IMAGE_NAME:latest \
      --zone=$ZONE \
      --project=$PROJECT_ID
    
    echo "VM atualizada com sucesso!"
  fi
  
  # Verificar se a regra de firewall já existe
  FIREWALL_EXISTS=$(gcloud compute firewall-rules list --project=$PROJECT_ID --filter="name=allow-http-8080" --format="value(name)")
  
  if [ -z "$FIREWALL_EXISTS" ]; then
    # Criar regra de firewall para permitir tráfego HTTP
    gcloud compute firewall-rules create allow-http-8080 \
      --allow=tcp:8080 \
      --target-tags=http-server \
      --description="Permitir tráfego HTTP na porta 8080" \
      --project=$PROJECT_ID
    
    echo "Regra de firewall criada com sucesso!"
  else
    echo "Regra de firewall já existe, pulando criação."
  fi
  
  # Obter o IP externo da VM
  IP_ADDRESS=$(gcloud compute instances describe $VM_NAME --zone=$ZONE --project=$PROJECT_ID --format="value(networkInterfaces[0].accessConfigs[0].natIP)")
  
  echo "Deploy concluído! A aplicação está disponível em: http://$IP_ADDRESS:8080"
}

# Fazer deploy no Cloud Run
deploy_cloud_run() {
  echo "Fazendo deploy no Cloud Run..."
  
  # Deploy no Cloud Run
  gcloud run deploy $IMAGE_NAME \
    --image=$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/$IMAGE_NAME:latest \
    --platform=managed \
    --region=$REGION \
    --allow-unauthenticated \
    --project=$PROJECT_ID
  
  # Obter a URL do serviço
  SERVICE_URL=$(gcloud run services describe $IMAGE_NAME --platform=managed --region=$REGION --project=$PROJECT_ID --format="value(status.url)")
  
  echo "Deploy concluído! A aplicação está disponível em: $SERVICE_URL"
}

# Processar argumentos da linha de comando
if [ $# -eq 0 ]; then
  show_help
  exit 0
fi

case "$1" in
  --help)
    show_help
    ;;
  --setup-repo)
    setup_repo
    ;;
  --build-push)
    build_push
    ;;
  --deploy-vm)
    deploy_vm
    ;;
  --deploy-cloud-run)
    deploy_cloud_run
    ;;
  --all)
    setup_repo
    build_push
    deploy_vm
    ;;
  *)
    echo "Opção desconhecida: $1"
    show_help
    exit 1
    ;;
esac

exit 0 