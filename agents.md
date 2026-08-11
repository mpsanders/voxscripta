# Agent Instructions for this repository

## Go code style

- Follow standard Go idioms and keep implementations simple, clear, and idiomatic.
- Format all Go source files with gofmt before considering a change complete.
- Prefer small, focused functions; use descriptive names; keep package-level API minimal and intentional.
- Use standard library patterns for error handling, file I/O, and concurrency.
- Avoid unnecessary abstractions, duplication, or premature optimization.
- Add GoDoc headers to all methods and functions to provide comprehensive documentation. For non-test method & functions be sure to detail the goal/purpose of the method, and be sure to document any parameters.
- Prefer well known, maintained 3rd party libraries where appropriate to achieve commmon tasks rather than reinventing the wheel.

## Testing expectations

- Write tests as table-driven tests with subtests whenever practical.
- Every new or modified test should include at least 5 cases.
- Include a nil or null-style case when applicable, along with a zero-value case.
- Cover both expected and unexpected inputs, including negative values and out-of-range values where relevant.
- Make test names and table case names explicit so failures are easy to understand.

## Change guidance

- Preserve existing behavior unless the task explicitly requires a change.
- Prefer minimal, targeted edits that solve the stated problem without introducing unrelated churn.

## Documentation

Ensure that there is up-to-date documentation representing the current state of the project.

- docs/uml: Contains plantuml files (class/package/sequence diagrams) to help get an understanding of the code and help build mental models.
- docs/planning: Contains overall project guidance. IDEAS.md for things that could be useful functionality in the future. TODO.md for tasks that need to be done to work through the roadmap, and to keep the project in good shape. ROADMAP.md to show the upcoming path for the project. GOAL.md to provide the project's 'North Star' to ensure we're always working toward the ultimate goal.

These files should be reviewed and maintained as the code evolves.