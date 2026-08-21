# Compiler bugs

## Frontend linker rejects receiving from Context.Done

The std/context compiled regression reaches cancellation successfully, but adding `select { case <-ctx.Done(): ... }` where `ctx` has interface type `context.Context` fails with RENVO-LINK-003 (`linked unit is invalid`). Direct context cancellation and Err compile and run. The missing semantic is channel receive/select through an interface method result.

## errors.As cannot assign general typed targets

The Renvo errors.As implementation can traverse wrappers and invoke custom As methods, but cannot implement Go's required assignment into arbitrary pointers such as `**MyError`. The current reflect surface has no assignability test or Value.Set/interface assignment intrinsic. This is compiler/runtime reflection work rather than an errors package semantic exception.

