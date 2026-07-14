module example.com/rtgtests/negative/module_cycle

go 1.25

require example.com/rtgtests/negative/module_cycle/lib v0.0.0

replace example.com/rtgtests/negative/module_cycle/lib => ./lib
