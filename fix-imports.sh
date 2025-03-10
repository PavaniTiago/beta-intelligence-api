#!/bin/bash

# Script para corrigir os caminhos de importação no projeto

# Primeiro, vamos verificar se há algum caminho corrompido
echo "Verificando caminhos corrompidos..."
grep -r "github.com/PavaniTiago/beta-intelligence-api-api" --include="*.go" .

# Corrigir caminhos corrompidos
echo "Corrigindo caminhos corrompidos..."
find . -name "*.go" -type f -exec sed -i 's|github.com/PavaniTiago/beta-intelligence-api-api.*internal|github.com/PavaniTiago/beta-intelligence-api/internal|g' {} \;

# Garantir que todos os caminhos estejam corretos
echo "Garantindo que todos os caminhos estejam corretos..."
find . -name "*.go" -type f -exec sed -i 's|github.com/PavaniTiago/beta-intelligence/|github.com/PavaniTiago/beta-intelligence-api/|g' {} \;

echo "Caminhos de importação corrigidos com sucesso!"

# Verificar se ainda há problemas
echo "Verificando se ainda há problemas..."
grep -r "github.com/PavaniTiago/beta-intelligence/" --include="*.go" .
grep -r "github.com/PavaniTiago/beta-intelligence-api-api" --include="*.go" . 