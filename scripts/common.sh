#!/bin/bash
# Shared helpers for db.sh, seed.sh and migrate.sh.
#
# These three scripts used to carry their own copies of the .env loader and the
# "build the binary if it is missing" check. The copies had already drifted
# apart, and the build check was the dangerous one: a binary left behind in
# bin/ is silently preferred over the current source, so a checkout that was
# copied rather than cloned runs whatever was compiled last, from wherever it
# was compiled. Sharing one implementation keeps them consistent.

# load_env_file exports KEY=VALUE pairs from .env, if it exists.
#
# Variables already present in the environment win, so `DB_NAME=other ./db.sh ...`
# works. This matches godotenv.Load() in the Go tools, which also does not
# override. The original `export $(grep -v '^#' .env | xargs)` did override,
# and also broke on any value containing spaces or a '#'.
load_env_file() {
    [ -f .env ] || return 0

    echo -e "${BLUE}Loading configuration from .env file...${NC}"

    local line key value
    while IFS= read -r line || [ -n "$line" ]; do
        # Strip carriage returns, then skip blanks and comments.
        line="${line%$'\r'}"
        case "$line" in
            ''|\#*) continue ;;
            *=*) ;;
            *) continue ;;
        esac

        key="${line%%=*}"
        value="${line#*=}"

        # Strip matching surrounding quotes.
        case "$value" in
            \"*\") value="${value#\"}"; value="${value%\"}" ;;
            \'*\') value="${value#\'}"; value="${value%\'}" ;;
        esac

        # Only set what the caller has not already provided.
        if [ -z "${!key+x}" ]; then
            export "$key=$value"
        fi
    done < .env
}

# build_if_stale builds a Go binary unless it is already newer than every source
# file in the project. It sets BUILD_REBUILT to 1 when it actually built, so
# callers can report that without duplicating the staleness logic.
#
# `go build` is incremental, so this costs almost nothing when nothing has
# changed, while removing the failure mode where a stale binary shadows edited
# source. Checking only for the file's existence is not enough: the binary can
# be older than the code and still be used, and a bin/ directory carried over
# from another checkout is exactly that case.
BUILD_REBUILT=0

build_if_stale() {
    local binary="$1"
    shift

    BUILD_REBUILT=0

    if [ -f "$binary" ] && [ -z "$(find . -name '*.go' -newer "$binary" -not -path './bin/*' -print -quit 2>/dev/null)" ]; then
        return 0
    fi

    mkdir -p "$(dirname "$binary")"
    go build -o "$binary" "$@" || return 1

    BUILD_REBUILT=1
}
