#!/bin/bash

# Quick setup script for using existing PostgreSQL
# Usage: ./setup.sh

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}   Nimbus Development Setup (Native)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if PostgreSQL is running
if ! pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ PostgreSQL doesn't seem to be running on port 5432${NC}"
    echo "Please start PostgreSQL first or use Docker: ./dev.sh start"
    exit 1
fi

echo -e "${GREEN}✓${NC} PostgreSQL is running"

# Copy environment files
echo ""
echo -e "${YELLOW}Setting up environment files...${NC}"

if [ ! -f backend/.env ]; then
    cp backend/.env.example backend/.env
    echo -e "${GREEN}✓${NC} Created backend/.env"
else
    echo -e "${BLUE}ℹ${NC} backend/.env already exists"
fi

if [ ! -f frontend/.env.local ]; then
    cp frontend/.env.local.example frontend/.env.local
    echo -e "${GREEN}✓${NC} Created frontend/.env.local"
else
    echo -e "${BLUE}ℹ${NC} frontend/.env.local already exists"
fi

# Install backend dependencies
echo ""
echo -e "${YELLOW}Installing backend dependencies...${NC}"
cd backend && go mod download && cd ..
echo -e "${GREEN}✓${NC} Backend dependencies installed"

# Install frontend dependencies
echo ""
echo -e "${YELLOW}Installing frontend dependencies...${NC}"
cd frontend && npm install && cd ..
echo -e "${GREEN}✓${NC} Frontend dependencies installed"

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Setup Complete! 🎉${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}⚠ IMPORTANT: Create the 'nimbus' database${NC}"
echo ""
echo "Option 1 - Via pgAdmin (easiest):"
echo "  1. Open pgAdmin 4"
echo "  2. Right-click Databases → Create → Database"
echo "  3. Name: nimbus"
echo "  4. Click Save"
echo ""
echo "Option 2 - Via psql:"
echo "  psql -U postgres"
echo "  CREATE DATABASE nimbus;"
echo "  \\q"
echo ""
echo -e "${YELLOW}Then update backend/.env with your postgres password!${NC}"
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Start development:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Terminal 1:"
echo "  cd backend && make dev"
echo ""
echo "Terminal 2:"
echo "  cd frontend && npm run dev"
echo ""
echo -e "${GREEN}Access your app:${NC}"
echo "  Frontend: http://localhost:3000"
echo "  Backend:  http://localhost:8080/api/v1/health"
echo ""