#!/bin/bash

# Database Seeding Script
# Wrapper for Go seed tool with friendly interface

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Shared helpers (safe .env loading, staleness-aware builds)
# shellcheck source=scripts/common.sh
. "$(dirname "$0")/scripts/common.sh"

load_env_file

# Configuration
SEED_BINARY="./bin/seed"
GO_FILE="./cmd/seed/main.go"

# Build the seed tool unless the existing binary is already up to date
check_seed_binary() {
  build_if_stale "$SEED_BINARY" "$GO_FILE" || exit 1
  [ "$BUILD_REBUILT" = "1" ] && echo -e "${GREEN}✓ Seed tool built${NC}"
  return 0
}

# Show usage
show_usage() {
  echo -e "${CYAN}==========================================${NC}"
  echo -e "${CYAN}  Database Seeding Tool${NC}"
  echo -e "${CYAN}==========================================${NC}"
  echo ""
  echo "Usage: $0 <command> [options]"
  echo ""
  echo -e "${GREEN}Commands:${NC}"
  echo "  all                  Run all seeders"
  echo "  <seeder_name>        Run specific seeder"
  echo "  list                 List available seeders"
  echo "  rebuild              Rebuild seed tool"
  echo "  help, -h, --help     Show this help message"
  echo ""
  echo -e "${GREEN}Available Seeders:${NC}"
  echo "  address_reference    Seed countries, provinces, cities, areas"
  echo "  users                Seed the default admin and user accounts"
  echo ""
  echo -e "${BLUE}Examples:${NC}"
  echo "  $0 all                             Run all seeders"
  echo "  $0 address_reference               Run address_reference seeder only"
  echo "  $0 users                           Run users seeder only"
  echo "  $0 list                            List available seeders"
  echo ""
  echo -e "${YELLOW}Environment Variables:${NC}"
  echo "  DB_HOST         Database host (default: localhost)"
  echo "  DB_PORT         Database port (default: 5432)"
  echo "  DB_USER         Database user (default: deployer)"
  echo "  DB_PASSWORD     Database password"
  echo "  DB_NAME         Database name (default: gatra_budget_db)"
  echo ""
}

# Run all seeders
run_all_seeders() {
  check_seed_binary
  echo -e "${BLUE}Running all seeders...${NC}"
  echo ""
  "$SEED_BINARY"
}

# Run specific seeder
run_specific_seeder() {
  check_seed_binary
  seeder_name="$1"
  echo -e "${BLUE}Running seeder: $seeder_name${NC}"
  echo ""
  "$SEED_BINARY" "$seeder_name"
}

# List available seeders
list_seeders() {
  echo -e "${CYAN}==========================================${NC}"
  echo -e "${CYAN}  Available Seeders${NC}"
  echo -e "${CYAN}==========================================${NC}"
  echo ""
  echo -e "${GREEN}▶ address_reference${NC} - Seed countries, provinces, cities, and areas"
  echo -e "${GREEN}▶ users${NC} - Seed the default admin and user accounts"
  echo ""
  echo -e "${YELLOW}To add more seeders:${NC}"
  echo "  1. Create a new file in seeds/ directory"
  echo "  2. Implement the seeder function"
  echo "  3. Register it in cmd/seed/main.go"
  echo ""
}

# Rebuild seed tool
rebuild_tool() {
  echo -e "${BLUE}Rebuilding seed tool...${NC}"
  rm -f "$SEED_BINARY"
  mkdir -p bin
  go build -o "$SEED_BINARY" "$GO_FILE"
  echo -e "${GREEN}✓ Seed tool rebuilt${NC}"
}

# Main command handler
case "${1:-help}" in
all)
  run_all_seeders
  ;;

list | ls)
  list_seeders
  ;;

rebuild)
  rebuild_tool
  ;;

help | -h | --help)
  show_usage
  ;;

*)
  if [ -n "$1" ]; then
    # Try to run as specific seeder
    run_specific_seeder "$1"
  else
    show_usage
  fi
  ;;
esac
