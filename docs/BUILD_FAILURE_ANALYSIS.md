# Integrated development failure log

## External access (2026-11-08)

- `gh pr view 12 --json ...` and PR API calls: proxy/API 403 for `https://api.github.com/graphql`. Git HTTPS fetch/push works. Use the authorized Git path and provide a PR creation URL. Do not extract proxy credentials or request secret values in chat.
- Crossref metadata request: proxy CONNECT 403 for `api.crossref.org`. No Glorius SI-derived preset has been registered without verified title/DOI/page/settings. Existing general presets remain general presets.
- Required follow-up: allow the required domains in environment settings, then retry the specific read/API operation with existing authentication. Publisher/SI host requirements must come from the actual source, not guesses.

## Validation scope

Parser and calculation tests use explicitly synthetic fixtures and Mock ORCA. There is no installed licensed ORCA or Windows desktop on this Linux machine. GitHub Windows Actions validates Go tests, WebView2 host compilation and ZIP packaging; it does not establish scientific correctness or validate real MPI/update restart behavior.

New compile/test failures and fixes are recorded below as they occur. Existing historical failure analysis is retained in the repository root.

## Windows run 42496832164

Code commit 76a2f0b: Windows workflow failed after 53 seconds (exit 1). The public page did not expose test logs; authenticated API requests remain blocked by 403. No cause is claimed from the exit code alone. Added CI capture of verbose Go test output and a bounded public error annotation to obtain the actual failing test without bypassing access controls. Local Linux 73 test/subtest cases and race suite passed. Follow-up run and fix recorded below.
