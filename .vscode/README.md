# MatchMate VS Code Workspace

The committed workspace configuration keeps editor behavior consistent without overriding personal themes, keyboard shortcuts, font choices, or other user preferences.

- `extensions.json` recommends the project-supported language, container, database, CI, and documentation extensions.
- `settings.json` enables format-on-save, explicit ESLint fixes, Go import organization, LF line endings, and monorepo search/watcher exclusions.

Do not add machine-specific paths, credentials, database passwords, personal extension settings, or secrets here.

Add shared tasks or launch configurations only after the matching repository commands exist. A task must call the same checked-in command used by CI rather than duplicating build logic in VS Code.

See [`../docs/development/README.md`](../docs/development/README.md) for complete setup instructions.

