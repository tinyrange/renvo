# Compiler bugs

## Frontend linker rejects receiving from Context.Done

The std/context compiled regression reaches cancellation successfully, but adding `select { case <-ctx.Done(): ... }` where `ctx` has interface type `context.Context` fails with RENVO-LINK-003 (`linked unit is invalid`). Direct context cancellation and Err compile and run. The missing semantic is channel receive/select through an interface method result.

