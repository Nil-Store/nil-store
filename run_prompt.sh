while true; do 
	npx --yes @openai/codex@latest --dangerously-bypass-approvals-and-sandbox -m 'gpt-5.2-codex' exec "$(cat mainnet_parity_prompt.md)"
done
