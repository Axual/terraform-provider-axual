// Testing!
package webclient_test

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Write test setup code here.
	// If I recall correctly, this is only run once.
	// Maybe you could do things like retrieving Axual credentials here?
	// Or do some platform setup? I have no idea.
	log.Println("Setting up test infrastructure")

	exitCode := m.Run()

	// Write test teardown code here.
	log.Println("Tearing down test infrastructure")

	os.Exit(exitCode)
}
