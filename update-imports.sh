#!/bin/bash

# Script para atualizar os caminhos de importação no projeto

OLD_PATH="github.com/PavaniTiago/beta-intelligence"
NEW_PATH="github.com/PavaniTiago/beta-intelligence-api"

# Encontrar todos os arquivos .go e substituir os caminhos
find . -name "*.go" -type f -exec sed -i "s|$OLD_PATH|$NEW_PATH|g" {} \;

echo "Caminhos de importação atualizados com sucesso!"

# Verifique se ainda existem referências ao caminho antigo
grep -r "github.com/PavaniTiago/beta-intelligence/" --include="*.go" . 

# Primeiro, vamos garantir que o script tenha permissão de execução
chmod +x update-imports.sh

# Agora, execute o script
./update-imports.sh 