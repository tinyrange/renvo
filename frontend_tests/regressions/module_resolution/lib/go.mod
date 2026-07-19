module example.com/renvotests/regressions/module_resolution/lib

go 1.25

require example.com/renvotests/regressions/module_resolution/value v0.0.0

replace example.com/renvotests/regressions/module_resolution/value => ../value
