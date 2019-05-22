module skylight.example/fuzz

go 1.12

require (
	github.com/dvyukov/go-fuzz v0.0.0-20190516070045-5cc3605ccbb6
	skylight.example/rpn v0.0.0
)

replace skylight.example/rpn => ../../src
