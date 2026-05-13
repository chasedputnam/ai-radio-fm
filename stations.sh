#!/usr/bin/env bash
# stations.sh — Orchestration script for AI Radio FM multi-station deployment.
#
# Usage:
#   ./stations.sh start              Start TTS sidecar + all stations
#   ./stations.sh stop               Stop all stations + TTS sidecar
#   ./stations.sh status             Show running/stopped state of all processes
#   ./stations.sh restart <name>     Stop and restart a single station
#   ./stations.sh logs <name>        Tail the log for a station
#
# Station discovery: any directory under stations/ containing a schedule.yaml.
# Environment: stations/.env.shared is sourced first, then stations/<name>/.env
# per station. Per-station values override shared values.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATIONS_DIR="$SCRIPT_DIR/stations"
BINARY="$SCRIPT_DIR/airadio"
TTS_BINARY="$SCRIPT_DIR/go-kokoro-tts/tts-server"
TTS_PID_FILE="$STATIONS_DIR/tts-server.pid"
TTS_LOG_FILE="$STATIONS_DIR/tts-server.log"
SHARED_ENV="$STATIONS_DIR/.env.shared"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

log() { echo "[stations.sh] $*"; }
err() { echo "[stations.sh] ERROR: $*" >&2; }

# discover_stations prints one station name per line.
discover_stations() {
    find "$STATIONS_DIR" -mindepth 2 -maxdepth 2 -name "schedule.yaml" \
        | sed "s|$STATIONS_DIR/||" \
        | sed 's|/schedule.yaml||' \
        | sort
}

# load_env <name> sources shared env then per-station env into the current shell.
load_env() {
    local name="$1"
    if [[ -f "$SHARED_ENV" ]]; then
        set -a
        # shellcheck source=/dev/null
        source "$SHARED_ENV"
        set +a
    fi
    local station_env="$STATIONS_DIR/$name/.env"
    if [[ -f "$station_env" ]]; then
        set -a
        # shellcheck source=/dev/null
        source "$station_env"
        set +a
    fi
}

# get_api_addr <name> returns the API_ADDR for a station (from its env or default).
get_api_addr() {
    local name="$1"
    local addr=""
    # Source env in a subshell to avoid polluting current shell.
    addr=$(
        [[ -f "$SHARED_ENV" ]] && source "$SHARED_ENV" 2>/dev/null || true
        local station_env="$STATIONS_DIR/$name/.env"
        [[ -f "$station_env" ]] && source "$station_env" 2>/dev/null || true
        echo "${API_ADDR:-}"
    )
    echo "$addr"
}

# get_mount <name> returns the ICECAST_MOUNT for a station (from its env or default /<name>).
get_mount() {
    local name="$1"
    local mount=""
    mount=$(
        [[ -f "$SHARED_ENV" ]] && source "$SHARED_ENV" 2>/dev/null || true
        local station_env="$STATIONS_DIR/$name/.env"
        [[ -f "$station_env" ]] && source "$station_env" 2>/dev/null || true
        echo "${ICECAST_MOUNT:-/$name}"
    )
    echo "$mount"
}

# is_running <pid_file> returns 0 if the process is alive, 1 otherwise.
is_running() {
    local pid_file="$1"
    if [[ ! -f "$pid_file" ]]; then return 1; fi
    local pid
    pid=$(cat "$pid_file")
    if [[ -z "$pid" ]]; then return 1; fi
    kill -0 "$pid" 2>/dev/null
}

# stop_process <pid_file> <label> sends SIGTERM and waits up to 5 seconds.
stop_process() {
    local pid_file="$1"
    local label="$2"
    if ! is_running "$pid_file"; then
        log "$label: not running"
        return 0
    fi
    local pid
    pid=$(cat "$pid_file")
    log "Stopping $label (PID $pid)..."
    kill -TERM "$pid" 2>/dev/null || true
    local i=0
    while kill -0 "$pid" 2>/dev/null && [[ $i -lt 10 ]]; do
        sleep 0.5
        ((i++))
    done
    if kill -0 "$pid" 2>/dev/null; then
        err "$label did not stop within 5 seconds — sending SIGKILL"
        kill -KILL "$pid" 2>/dev/null || true
    fi
    rm -f "$pid_file"
    log "$label stopped."
}

# ---------------------------------------------------------------------------
# Conflict detection
# ---------------------------------------------------------------------------

check_conflicts() {
    local stations=("$@")
    local -A seen_ports seen_mounts
    local conflict=0

    for name in "${stations[@]}"; do
        local addr
        addr=$(get_api_addr "$name")
        if [[ -n "$addr" ]]; then
            if [[ -n "${seen_ports[$addr]+x}" ]]; then
                err "API_ADDR conflict: stations '${seen_ports[$addr]}' and '$name' both use $addr"
                conflict=1
            else
                seen_ports[$addr]="$name"
            fi
        fi

        local mount
        mount=$(get_mount "$name")
        if [[ -n "${seen_mounts[$mount]+x}" ]]; then
            err "ICECAST_MOUNT conflict: stations '${seen_mounts[$mount]}' and '$name' both use $mount"
            conflict=1
        else
            seen_mounts[$mount]="$name"
        fi
    done

    if [[ $conflict -ne 0 ]]; then
        err "Resolve conflicts before starting. Exiting."
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

cmd_start() {
    mapfile -t stations < <(discover_stations)
    if [[ ${#stations[@]} -eq 0 ]]; then
        err "No stations found under $STATIONS_DIR (each needs a schedule.yaml)"
        exit 1
    fi

    log "Found stations: ${stations[*]}"
    check_conflicts "${stations[@]}"

    if [[ ! -f "$BINARY" ]]; then
        err "Binary not found at $BINARY — run 'go build -o airadio .' first"
        exit 1
    fi

    # Start TTS sidecar.
    if is_running "$TTS_PID_FILE"; then
        log "TTS sidecar already running (PID $(cat "$TTS_PID_FILE"))"
    else
        if [[ ! -f "$TTS_BINARY" ]]; then
            log "Warning: tts-server binary not found at $TTS_BINARY — TTS disabled for all stations"
        else
            # Load shared env to get sidecar flags.
            local tts_addr lib_path model_path voice_dir coreml_flag
            tts_addr=$(
                [[ -f "$SHARED_ENV" ]] && source "$SHARED_ENV" 2>/dev/null || true
                echo "${TTS_GRPC_ADDR:-:50051}"
            )
            lib_path=$(
                [[ -f "$SHARED_ENV" ]] && source "$SHARED_ENV" 2>/dev/null || true
                echo "${KOKORO_LIB_PATH:-/opt/homebrew/lib/libonnxruntime.dylib}"
            )
            model_path=$(
                [[ -f "$SHARED_ENV" ]] && source "$SHARED_ENV" 2>/dev/null || true
                echo "${KOKORO_MODEL_PATH:-./go-kokoro-tts/kokoro-v0_19.onnx}"
            )
            voice_dir=$(
                [[ -f "$SHARED_ENV" ]] && source "$SHARED_ENV" 2>/dev/null || true
                echo "${KOKORO_VOICE_DIR:-./go-kokoro-tts/voices}"
            )
            coreml_flag=""
            local coreml_val
            coreml_val=$(
                [[ -f "$SHARED_ENV" ]] && source "$SHARED_ENV" 2>/dev/null || true
                echo "${KOKORO_COREML:-true}"
            )
            if [[ "$coreml_val" == "true" || "$coreml_val" == "1" ]]; then
                coreml_flag="--coreml"
            fi

            mkdir -p "$STATIONS_DIR"
            log "Starting TTS sidecar on $tts_addr..."
            "$TTS_BINARY" \
                --addr "$tts_addr" \
                --lib "$lib_path" \
                --model "$model_path" \
                --voice-dir "$voice_dir" \
                $coreml_flag \
                >> "$TTS_LOG_FILE" 2>&1 &
            echo $! > "$TTS_PID_FILE"
            log "TTS sidecar started (PID $(cat "$TTS_PID_FILE"))"
            # Brief pause to let the sidecar initialize before stations connect.
            sleep 2
        fi
    fi

    # Start each station.
    for name in "${stations[@]}"; do
        local pid_file="$STATIONS_DIR/$name/station.pid"
        local log_file="$STATIONS_DIR/$name/station.log"

        if is_running "$pid_file"; then
            log "Station '$name' already running (PID $(cat "$pid_file"))"
            continue
        fi

        mkdir -p "$STATIONS_DIR/$name"
        log "Starting station '$name'..."

        # Build env for this station process.
        (
            load_env "$name"
            "$BINARY" start --station "$name" >> "$log_file" 2>&1
        ) &
        local station_pid=$!
        echo $station_pid > "$pid_file"
        log "Station '$name' started (PID $station_pid)"
    done

    log "All stations started. Use './stations.sh status' to check."
}

cmd_stop() {
    mapfile -t stations < <(discover_stations)

    for name in "${stations[@]}"; do
        local pid_file="$STATIONS_DIR/$name/station.pid"
        stop_process "$pid_file" "station '$name'"
    done

    stop_process "$TTS_PID_FILE" "TTS sidecar"
    log "All processes stopped."
}

cmd_status() {
    mapfile -t stations < <(discover_stations)

    echo ""
    printf "%-30s %-10s %s\n" "COMPONENT" "STATUS" "PID"
    printf "%-30s %-10s %s\n" "---------" "------" "---"

    # TTS sidecar
    if is_running "$TTS_PID_FILE"; then
        printf "%-30s %-10s %s\n" "tts-server" "running" "$(cat "$TTS_PID_FILE")"
    else
        printf "%-30s %-10s\n" "tts-server" "stopped"
    fi

    for name in "${stations[@]}"; do
        local pid_file="$STATIONS_DIR/$name/station.pid"
        if is_running "$pid_file"; then
            printf "%-30s %-10s %s\n" "station/$name" "running" "$(cat "$pid_file")"
        else
            printf "%-30s %-10s\n" "station/$name" "stopped"
        fi
    done
    echo ""
}

cmd_restart() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        err "Usage: $0 restart <station_name>"
        exit 1
    fi

    local station_dir="$STATIONS_DIR/$name"
    if [[ ! -d "$station_dir" ]]; then
        err "Station '$name' not found at $station_dir"
        exit 1
    fi

    local pid_file="$STATIONS_DIR/$name/station.pid"
    local log_file="$STATIONS_DIR/$name/station.log"

    stop_process "$pid_file" "station '$name'"

    log "Restarting station '$name'..."
    (
        load_env "$name"
        "$BINARY" start --station "$name" >> "$log_file" 2>&1
    ) &
    local station_pid=$!
    echo $station_pid > "$pid_file"
    log "Station '$name' restarted (PID $station_pid)"
}

cmd_logs() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        err "Usage: $0 logs <station_name>"
        exit 1
    fi

    local log_file="$STATIONS_DIR/$name/station.log"
    if [[ ! -f "$log_file" ]]; then
        err "No log file found at $log_file"
        exit 1
    fi

    tail -f "$log_file"
}

# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

COMMAND="${1:-}"
shift || true

case "$COMMAND" in
    start)   cmd_start ;;
    stop)    cmd_stop ;;
    status)  cmd_status ;;
    restart) cmd_restart "${1:-}" ;;
    logs)    cmd_logs "${1:-}" ;;
    *)
        echo "Usage: $0 {start|stop|status|restart <name>|logs <name>}"
        exit 1
        ;;
esac
