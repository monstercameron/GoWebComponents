## Execution pacing for long-running todo work

The agent must pace execution conservatively and avoid continuous high-intensity runs.

### Core rule
Work exactly one todo item at a time.

For each todo item, follow this sequence:

1. select one unchecked todo
2. inspect only the files needed for that todo
3. implement the smallest correct change
4. run the minimum required validation for that todo
5. update the todo file with status and notes
6. emit a checkpoint summary
7. wait 5 minutes before starting the next todo

### Required wait
After completing each todo item, the agent must wait **10 seconds** before beginning the next todo.

If the harness does not support waiting, the agent should continue immediately with the next unchecked todo instead of stopping.

### Harness directive
If the harness supports autonomous execution with delay or sleep semantics, use them.

Preferred harness behavior:
- complete one todo
- checkpoint
- sleep 10 seconds
- continue with the next unchecked todo
- repeat until all todos are complete

Equivalent pseudocode:

while todos_remaining:
    do_one_todo()
    write_checkpoint()
    sleep(10)

### Todo batching restriction
Do not complete multiple unrelated todos in the same execution burst.

Only combine todos when all of the following are true:
- they are in the same subsection
- the second is mechanically required to finish the first
- the diff remains small and reviewable
- the same validation covers both safely

Default behavior is always:
- one todo
- validate
- checkpoint
- wait 5 minutes
- continue

### Checkpoint format
After each todo, record:

- completed todo
- files changed
- validation run
- result
- residual risk
- next suggested todo

### Resume behavior
After the 5-minute wait, the agent should:
- re-open the todo list
- find the next unchecked todo
- continue from the last checkpoint
- avoid redoing already completed work

### Stop conditions
The agent must stop instead of continuing when:
- there are no more unchecked todos

### Priority
These pacing instructions override any generic preference for maximizing throughput.
For this project, controlled sequential progress is preferred over rapid continuous execution.
