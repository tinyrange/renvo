package semicolon; import "example.com/genericnestedcallbacks/callback"; func Apply(value int) int { return callback.Run(func(other int) int { return value }) }
