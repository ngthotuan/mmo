Run the full project verification suite and report any errors.

Steps:
1. Run `cd /home/tuannt92/dev/projects/mmo/backend && go build ./...` — report any compile errors
2. Run `cd /home/tuannt92/dev/projects/mmo/backend && go vet ./...` — report any vet warnings
3. Run `cd /home/tuannt92/dev/projects/mmo/frontend && npx tsc --noEmit` — report any TypeScript errors

If all three pass with zero output, report "All checks pass ✓". If any fail, show the exact error output and identify which files need fixing.
