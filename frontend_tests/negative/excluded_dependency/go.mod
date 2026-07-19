module example.com/renvotests/negative/excluded_dependency

go 1.25

require example.com/excluded v1.0.0
exclude example.com/excluded v1.0.0
