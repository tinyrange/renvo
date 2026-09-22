module example.com/versionapp

go 1.26

require example.com/versionlib v1.0.0

replace example.com/versionlib => ./lib
replace example.com/versionvalue => ./value
