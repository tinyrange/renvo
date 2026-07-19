module example.com/renvotests/negative/module_cycle

go 1.25

require example.com/renvotests/negative/module_cycle/lib v0.0.0

replace example.com/renvotests/negative/module_cycle/lib => ./lib
