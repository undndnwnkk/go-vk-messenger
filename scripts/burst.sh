#!/usr/bin/env bash
# Sends 24 concurrent messages from one fresh user, alternating between two
# replicas, and prints each HTTP status. With a shared Redis limiter
# (10 messages per 5 s per user) the expected result is 10 x 201 and 14 x 429.
#
# Usage: APP_A_URL=http://localhost:8081 APP_B_URL=http://localhost:8082 ./scripts/burst.sh
# Creates users, a chat and messages: run it against a test database.
set -euo pipefail

A=${APP_A_URL:-http://localhost:8081}
B=${APP_B_URL:-http://localhost:8082}
N=${BURST_SIZE:-24}
JSON='Content-Type: application/json'

field() { sed -E "s/.*\"$1\":\"([^\"]+)\".*/\1/"; }
register() {
	curl -sf -X POST "$1/api/v1/auth/register" -H "$JSON" \
		-d "{\"username\":\"burst_${RANDOM}${RANDOM}\",\"password\":\"burst-password-1\"}" | field access_token
}

sender=$(register "$A")
peer=$(register "$B")
peer_id=$(curl -sf "$B/api/v1/me" -H "Authorization: Bearer $peer" | field id)
chat_id=$(curl -sf -X POST "$A/api/v1/chats/direct" -H "Authorization: Bearer $sender" -H "$JSON" \
	-d "{\"user2_id\":\"$peer_id\"}" | field id)

out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
for i in $(seq 1 "$N"); do
	if ((i % 2)); then base=$A name=replica-a; else base=$B name=replica-b; fi
	(
		code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$base/api/v1/chats/$chat_id/messages" \
			-H "Authorization: Bearer $sender" -H "$JSON" -d '{"content":"burst"}')
		echo "$i $name $code" >"$out/$i"
	) &
done
wait

cat "$out"/* | sort -n | awk '
	{ text = ($3 == 201) ? "Created" : ($3 == 429) ? "Too Many Requests" : "unexpected"
	  printf "#%02d  %s  ->  %s %s\n", $1, $2, $3, text; count[$3]++ }
	END { printf "\nHTTP 201: %d   HTTP 429: %d   (total %d)\n", count[201], count[429], NR }'
