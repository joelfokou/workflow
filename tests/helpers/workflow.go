// Package helpers - workflows used in tests.
package helpers

func SimpleWorkflow() string {
	return `
name = "simple"

[tasks.a]
cmd = "echo hello"
`
}

func RetryWorkflow() string {
	return `
name = "retry"

[tasks.a]
cmd = "false"
retries = 2
`
}

func ComplexWorkflow() string {
	return `
name = "complex"

[tasks.a]
cmd = "echo Task A"
[tasks.b]
cmd = "echo Task B"
depends_on = ["a"]
[tasks.c]
cmd = "echo Task C"
depends_on = ["a"]
[tasks.d]
cmd = "echo Task D"
depends_on = ["b", "c"]
`
}

func FailingWorkflow() string {
	return `
name = "failing"

[tasks.fail]
cmd = "exit 1"
retries = 0
`
}

func LongRunningWorkflow() string {
	return `
name = "long-running"

[tasks.sleep]
cmd = "sleep 30"
retries = 0
`
}

func MultiTaskWorkflow() string {
	return `
name = "multi"

[tasks.build]
cmd = "echo Building"
retries = 1

[tasks.test]
cmd = "echo Testing"
depends_on = ["build"]
retries = 1

[tasks.deploy]
cmd = "echo Deploying"
depends_on = ["test"]
retries = 0
`
}

func InvalidWorkflow() string {
	return `
name = "invalid"

[tasks.task1]
cmd = "echo test"
depends_on = ["nonexistent"]
`
}

func CycleWorkflow() string {
	return `
name = "cycle"

[tasks.task1]
cmd = "echo test"
depends_on = ["task2"]

[tasks.task2]
cmd = "echo test"
depends_on = ["task1"]
`
}

func ResumeWorkflow() string {
	return `
name = "resume"

[tasks.task1]
cmd = "exit 1"

[tasks.task2]
cmd = "echo success"
depends_on = ["task1"]
`
}

func ResumeWorkflowFixed() string {
	return `
name = "resume"

[tasks.task1]
cmd = "exit 0"

[tasks.task2]
cmd = "echo success"
depends_on = ["task1"]
`
}
