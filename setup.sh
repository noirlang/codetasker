#!/usr/bin/env bash

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}"
echo "=========================================================="
echo "          CodeTasker Docker Setup & Installer             "
echo "=========================================================="
echo -e "${NC}"

# Helper function to generate secure random keys
generate_key() {
  local length=$1
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$((length / 2))"
  else
    tr -dc 'a-f0-9' < /dev/urandom | head -c "$length"
  fi
}

# 1. Check prerequisites
echo -e "${BLUE}[*] Checking prerequisites...${NC}"

DOCKER_CMD=""
if command -v docker >/dev/null 2>&1; then
  echo -e "  - Docker: ${GREEN}Installed${NC}"
else
  echo -e "  - Docker: ${RED}Not Found${NC}"
  echo -e "${YELLOW}Warning: Docker is required to run the containers. Please install Docker first.${NC}"
fi

if command -v docker-compose >/dev/null 2>&1; then
  DOCKER_CMD="docker-compose"
  echo -e "  - Docker Compose: ${GREEN}Installed (docker-compose)${NC}"
elif docker compose version >/dev/null 2>&1; then
  DOCKER_CMD="docker compose"
  echo -e "  - Docker Compose: ${GREEN}Installed (docker compose)${NC}"
else
  echo -e "  - Docker Compose: ${RED}Not Found${NC}"
  echo -e "${YELLOW}Warning: Docker Compose is required. Please install docker-compose or docker-compose-plugin.${NC}"
fi

echo ""
echo -e "${BLUE}[*] Configure Environment Variables${NC}"
echo "We will generate your .env file now. Press Enter to use the default values."
echo ""

# Helper to read with default value
prompt_default() {
  local var_name=$1
  local prompt_text=$2
  local default_value=$3
  local input_val

  read -p "$(echo -e "${CYAN}${prompt_text}${NC} [Default: ${default_value}]: ")" input_val
  if [ -z "$input_val" ]; then
    eval "$var_name=\"\$default_value\""
  else
    eval "$var_name=\"\$input_val\""
  fi
}

# Helper to read required field
prompt_required() {
  local var_name=$1
  local prompt_text=$2
  local input_val

  while true; do
    read -p "$(echo -e "${YELLOW}${prompt_text} (Required): ${NC}")" input_val
    if [ -n "$input_val" ]; then
      eval "$var_name=\"\$input_val\""
      break
    fi
  done
}

# Prompts
prompt_default PORT "Enter Backend API Port" "8080"
prompt_default MONGO_URI "Enter MongoDB URI" "mongodb://mongo:27017"
prompt_default DB_NAME "Enter MongoDB Database Name" "codetasker"

prompt_required GITHUB_CLIENT_ID "Enter GitHub OAuth Client ID"
prompt_required GITHUB_CLIENT_SECRET "Enter GitHub OAuth Client Secret"

prompt_default GITHUB_REDIRECT_URL "Enter GitHub OAuth Redirect URL" "http://localhost:8080/api/auth/github/callback"
prompt_default FRONTEND_URL "Enter Frontend URL" "http://localhost:5173"

# SMTP Prompts
echo ""
read -p "$(echo -e "${YELLOW}Do you want to enable email notifications (SMTP)? (y/n) ${NC}[Default: n]: ")" enable_smtp
if [ "$enable_smtp" = "y" ] || [ "$enable_smtp" = "Y" ]; then
  SMTP_ENABLED="true"
  prompt_required SMTP_HOST "Enter SMTP Host (e.g., mail.noirlang.tr)"
  prompt_default SMTP_PORT "Enter SMTP Port" "465"
  prompt_required SMTP_USERNAME "Enter SMTP Username"
  
  # Read password securely (without echoing)
  while true; do
    read -sp "$(echo -e "${YELLOW}Enter SMTP Password (Required): ${NC}")" SMTP_PASSWORD
    echo ""
    if [ -n "$SMTP_PASSWORD" ]; then
      break
    fi
  done

  prompt_default SMTP_FROM "Enter SMTP From Header" "CodeTasker <$SMTP_USERNAME>"
else
  SMTP_ENABLED="false"
  SMTP_HOST=""
  SMTP_PORT=""
  SMTP_USERNAME=""
  SMTP_PASSWORD=""
  SMTP_FROM=""
fi

# Secret generation or prompt
echo ""
read -p "$(echo -e "${CYAN}Enter JWT Secret (leave blank to auto-generate secure key)${NC}: ")" JWT_SECRET
if [ -z "$JWT_SECRET" ]; then
  JWT_SECRET=$(generate_key 64)
  echo -e "  ${GREEN}-> Generated secure JWT Secret.${NC}"
fi

read -p "$(echo -e "${CYAN}Enter GitHub Webhook Secret (leave blank to auto-generate secure key)${NC}: ")" WEBHOOK_SECRET
if [ -z "$WEBHOOK_SECRET" ]; then
  WEBHOOK_SECRET=$(generate_key 32)
  echo -e "  ${GREEN}-> Generated secure Webhook Secret.${NC}"
fi

read -p "$(echo -e "${CYAN}Enter AES-256 Token Encryption Key (must be exactly 32 hex chars, leave blank to auto-generate)${NC}: ")" TOKEN_ENCRYPT_KEY
if [ -z "$TOKEN_ENCRYPT_KEY" ]; then
  TOKEN_ENCRYPT_KEY=$(generate_key 32)
  echo -e "  ${GREEN}-> Generated secure Token Encryption Key.${NC}"
fi

# Create .env content
echo ""
echo -e "${BLUE}[*] Writing configuration to .env file...${NC}"

cat << EOF > .env
# ============================================================
# CodeTasker Environment Configuration (Generated via setup.sh)
# ============================================================

PORT=$PORT
MONGO_URI=$MONGO_URI
DB_NAME=$DB_NAME

GITHUB_CLIENT_ID=$GITHUB_CLIENT_ID
GITHUB_CLIENT_SECRET=$GITHUB_CLIENT_SECRET
GITHUB_REDIRECT_URL=$GITHUB_REDIRECT_URL

JWT_SECRET=$JWT_SECRET
WEBHOOK_SECRET=$WEBHOOK_SECRET
TOKEN_ENCRYPT_KEY=$TOKEN_ENCRYPT_KEY

FRONTEND_URL=$FRONTEND_URL

# SMTP configuration
SMTP_ENABLED=$SMTP_ENABLED
SMTP_HOST=$SMTP_HOST
SMTP_PORT=$SMTP_PORT
SMTP_USERNAME=$SMTP_USERNAME
SMTP_PASSWORD=$SMTP_PASSWORD
SMTP_FROM=$SMTP_FROM
EOF

echo -e "${GREEN}[+] Configuration saved to .env successfully!${NC}"
echo ""

# Ask to start
if [ -n "$DOCKER_CMD" ]; then
  read -p "$(echo -e "${YELLOW}Do you want to build and start the Docker containers now? (y/n) ${NC}[Default: y]: ")" run_docker
  if [ -z "$run_docker" ] || [ "$run_docker" = "y" ] || [ "$run_docker" = "Y" ]; then
    echo -e "${BLUE}[*] Running: $DOCKER_CMD up -d --build${NC}"
    $DOCKER_CMD up -d --build
    echo ""
    echo -e "${GREEN}=========================================================="
    echo "  CodeTasker is starting up!"
    echo "  - Frontend: $FRONTEND_URL"
    echo "  - Backend API: http://localhost:$PORT"
    echo "==========================================================${NC}"
  else
    echo -e "${YELLOW}Setup complete. You can run CodeTasker manually with: $DOCKER_CMD up -d --build${NC}"
  fi
else
  echo -e "${YELLOW}Setup complete. Please install Docker and run: docker compose up -d --build${NC}"
fi
