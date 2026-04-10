1. Run `git status` and `git diff` to understand all changes
2. For each changed module, check if its CLAUDE.md or README references
   affected behavior/APIs. If outdated, update the docs and include
   them in the commit.
3. Stage all changes with `git add -A`
4. Generate a conventional commit message:
   `<type>(<scope>): <subject>` (max 50 chars, imperative mood)
   Body: what and why, wrapped at 72 chars. Omit footer unless there are breaking changes or issue refs.
   Types: feat, fix, docs, style, refactor, perf, test, chore, ci
5. Show the proposed commit message and wait for confirmation before committing
6. After committing, remind about `git push` if on a feature branch
