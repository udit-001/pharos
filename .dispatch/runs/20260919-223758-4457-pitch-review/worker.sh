#!/usr/bin/env bash
set -o pipefail
echo "dispatch pitch-review | model closedrouter/glm-5.3-flash | tools read,grep,find,ls"
rc=0
cd /home/udit/Dev/personal/learn-tool || rc=97
if [ "$rc" -eq 0 ]; then
  timeout 900 pi --no-extensions --no-skills --no-prompt-templates --no-session --model closedrouter/glm-5.3-flash --tools read\,grep\,find\,ls Complete\ the\ task\ described\ in\ the\ brief\ below. --mode json < /home/udit/Dev/personal/learn-tool/.dispatch/runs/20260919-223758-4457-pitch-review/brief.md 2> >(tee /home/udit/Dev/personal/learn-tool/.dispatch/runs/20260919-223758-4457-pitch-review/log >&2) | tee /home/udit/Dev/personal/learn-tool/.dispatch/runs/20260919-223758-4457-pitch-review/events.jsonl | jq -r \(select\(.type==\"tool_execution_start\"\)\ \|\ \"→\ \"+.toolName+\"\ \"+\(\[.args.path\,\ .args.command\,\ .args.pattern\,\ .args.file_path\,\ .args.url\]\ \|\ map\(select\(.\!=null\)\)\ \|\ join\(\"\ \"\)\ \|\ .\[0:140\]\)\)\,\ \(select\(.type==\"tool_execution_end\"\ and\ .isError\)\ \|\ \"✗\ \"+.toolName+\"\ failed\"\) >&2
  rc=${PIPESTATUS[0]}
  wait
fi
jq -rs map\(select\(.type==\"message_end\"\ and\ .message.role==\"assistant\"\)\ \|\ .message.content\ \|\ map\(select\(.type==\"text\"\)\ \|\ .text\)\ \|\ join\(\"\"\)\)\ \|\ last\ //\ empty /home/udit/Dev/personal/learn-tool/.dispatch/runs/20260919-223758-4457-pitch-review/events.jsonl > /home/udit/Dev/personal/learn-tool/.dispatch/runs/20260919-223758-4457-pitch-review/output.md 2>>/home/udit/Dev/personal/learn-tool/.dispatch/runs/20260919-223758-4457-pitch-review/log || true
[ "$rc" -ne 0 ] && echo "dispatch: worker failed (exit $rc) — see /home/udit/Dev/personal/learn-tool/.dispatch/runs/20260919-223758-4457-pitch-review/log"
echo "dispatch run 20260919-223758-4457-pitch-review finished (exit $rc)"
