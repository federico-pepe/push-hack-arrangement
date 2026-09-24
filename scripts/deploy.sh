#!/usr/bin/env bash
# Copy the built binary to the Push and run it in the foreground (Ctrl+C releases
# display + MIDI intercept). Usage: scripts/deploy.sh [host]   (default push.local)
set -euo pipefail
HOST="${1:-push.local}"
DIR=/data/push-hack/hacks/arrangement
cd "$(dirname "$0")/.."
make build
ssh "ableton@$HOST" "mkdir -p $DIR"
scp arrangement hack.json "ableton@$HOST:$DIR/"
RS="/data/Music/Ableton/User Library/Remote Scripts/PushHackArrangement"
ssh "ableton@$HOST" "mkdir -p '$RS'"
scp remote-script/*.py "ableton@$HOST:$RS/"
ssh "ableton@$HOST" "chmod +x $DIR/arrangement"
echo "Running on $HOST. Press Shift+Session on Push. Ctrl+C to stop."
ssh -t "ableton@$HOST" "$DIR/arrangement"
