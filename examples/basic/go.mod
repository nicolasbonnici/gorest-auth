module example

go 1.25.1

require (
	github.com/nicolasbonnici/gorest v0.4.0
	github.com/nicolasbonnici/gorest-auth v1.0.0
)

replace github.com/nicolasbonnici/gorest-auth => ../..
