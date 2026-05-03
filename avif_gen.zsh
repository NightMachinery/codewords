#!/usr/bin/env zsh
export PS4='> '
setopt LOCAL_OPTIONS PIPE_FAIL PRINT_EXIT_VALUE ERR_RETURN SOURCE_TRACE XTRACE
setopt TYPESET_SILENT NO_CASE_GLOB multios re_match_pcre extendedglob pipefail interactivecomments hash_executables_only
setopt NO_BANG_HIST 2>/dev/null || true
##

export CODEWORDS_IMAGE_DIR="${CODEWORDS_IMAGE_DIR:-$HOME/Pictures/SurrealPictures/chosen_2}"
export CODEWORDS_IMAGE_CACHE_DIR="${CODEWORDS_IMAGE_CACHE_DIR:-$HOME/.cache/talespin/cards}"

./bin/codewords avif-cache gen
