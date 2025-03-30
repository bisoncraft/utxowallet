#!/bin/bash

# Build JSON array manually
json='['
first=1
for arg in "$@"; do
    # Escape double quotes and backslashes
    escaped_arg=$(printf '%s' "$arg" | sed 's/\\/\\\\/g; s/"/\\"/g')
    if [ $first -eq 1 ]; then
        json+="\"$escaped_arg\""
        first=0
    else
        json+=",\"$escaped_arg\""
    fi
done
json+=']'

# Post to localhost:44825
response=$(curl -sS -w "%{http_code}" -o /tmp/body.json -X POST http://localhost:44825/submit \
  -H "Content-Type: application/json" \
  -d "$json")

status="${response: -3}"  # last 3 characters = status code

if [[ "$status" =~ ^2 ]]; then
    cat /tmp/body.json | python3 -m json.tool
else
    echo "Error ($status):" >&2
    cat /tmp/body.json >&2
fi