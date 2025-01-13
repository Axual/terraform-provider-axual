module axual-debug-webclient

go 1.23.0

require axual-webclient v0.0.0

replace axual-webclient => ./../axual-webclient

require (
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/oauth2 v0.25.0 // indirect
)
