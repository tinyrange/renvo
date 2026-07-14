module example.com/rtgtests/regressions/module_resolution

go 1.25

require (
	example.com/rtgtests/regressions/module_resolution/lib v0.0.0
	example.com/rtgtests/regressions/module_resolution/value v0.0.0 // indirect
)

replace example.com/rtgtests/regressions/module_resolution/lib => ./lib

replace example.com/rtgtests/regressions/module_resolution/value => ./value
