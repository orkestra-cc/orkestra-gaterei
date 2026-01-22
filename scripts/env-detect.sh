#!/bin/bash

# ============================================
# ORKESTRA Environment Detection Utility
# ============================================
#
# Purpose: Shared environment detection for all scripts
# Usage: source scripts/env-detect.sh
#        detect_environment
#        echo "Environment: $DETECTED_ENV"
#
# Priority:
#   1. OS environment variable (ENV)
#   2. Value from docker/.env file
#
# ============================================

# Determine paths relative to this script
SCRIPT_DIR_ENV="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT_ENV="$(dirname "$SCRIPT_DIR_ENV")"
DOCKER_DIR_ENV="$PROJECT_ROOT_ENV/docker"
ENV_FILE_PATH="$DOCKER_DIR_ENV/.env"

# Colors for output
_RED='\033[0;31m'
_GREEN='\033[0;32m'
_YELLOW='\033[1;33m'
_CYAN='\033[0;36m'
_NC='\033[0m'

# Exported variables (set by detect_environment)
DETECTED_ENV=""
COMPOSE_FILE=""

# Detects environment from OS env var or .env file
# Sets: DETECTED_ENV, COMPOSE_FILE
# Returns: 0 on success, 1 on failure
detect_environment() {
    local env_value=""
    local source_type=""

    # Priority 1: Check OS environment variable
    if [ -n "${ENV:-}" ]; then
        env_value="$ENV"
        source_type="OS environment variable"
    # Priority 2: Read from .env file
    elif [ -f "$ENV_FILE_PATH" ]; then
        # Extract ENV value from .env file (handles quoted and unquoted values)
        env_value=$(grep -E "^ENV=" "$ENV_FILE_PATH" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" | tr -d '[:space:]')
        source_type=".env file"
    else
        echo -e "${_RED}ERROR: No .env file found at $ENV_FILE_PATH${_NC}"
        echo -e "${_YELLOW}Create one by copying from .env.example or setting ENV variable:${_NC}"
        echo -e "  cp $DOCKER_DIR_ENV/.env.example $ENV_FILE_PATH"
        echo -e "  # or"
        echo -e "  ENV=development ./deploy.sh"
        return 1
    fi

    # Validate environment value
    case "$env_value" in
        development|staging|production)
            DETECTED_ENV="$env_value"
            ;;
        "")
            echo -e "${_RED}ERROR: ENV variable is empty or not set${_NC}"
            echo -e "${_YELLOW}Set ENV=development|staging|production in $ENV_FILE_PATH${_NC}"
            echo -e "${_YELLOW}Or pass as environment variable: ENV=development ./deploy.sh${_NC}"
            return 1
            ;;
        *)
            echo -e "${_RED}ERROR: Invalid ENV value: '$env_value'${_NC}"
            echo -e "${_YELLOW}Valid values: development, staging, production${_NC}"
            return 1
            ;;
    esac

    # Set compose file based on environment
    case "$DETECTED_ENV" in
        development)
            COMPOSE_FILE="$DOCKER_DIR_ENV/docker-compose.dev.yml"
            ;;
        staging)
            COMPOSE_FILE="$DOCKER_DIR_ENV/docker-compose.staging.yml"
            ;;
        production)
            COMPOSE_FILE="$DOCKER_DIR_ENV/docker-compose.prod.yml"
            ;;
    esac

    echo -e "${_GREEN}Environment: $DETECTED_ENV${_NC} (from $source_type)"
    return 0
}

# Prints environment status with color-coded icon
print_env_status() {
    if [ -z "${DETECTED_ENV:-}" ]; then
        detect_environment || return 1
    fi

    local env_color
    local env_icon
    case "$DETECTED_ENV" in
        development)
            env_color=$_GREEN
            env_icon="[DEV]"
            ;;
        staging)
            env_color=$_YELLOW
            env_icon="[STG]"
            ;;
        production)
            env_color=$_RED
            env_icon="[PROD]"
            ;;
    esac

    echo -e "${env_color}$env_icon $(echo $DETECTED_ENV | tr '[:lower:]' '[:upper:]')${_NC}"
}

# Get environment without printing (for use in conditionals)
get_environment_silent() {
    local env_value=""

    if [ -n "${ENV:-}" ]; then
        env_value="$ENV"
    elif [ -f "$ENV_FILE_PATH" ]; then
        env_value=$(grep -E "^ENV=" "$ENV_FILE_PATH" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'" | tr -d '[:space:]')
    fi

    case "$env_value" in
        development|staging|production)
            echo "$env_value"
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}
