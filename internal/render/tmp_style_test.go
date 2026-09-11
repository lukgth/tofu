package render

import (
	"fmt"
	"testing"

	"github.com/alecthomas/chroma/v2/styles"
)

func TestTmpStyle(t *testing.T) {
	dark := styles.Get("tofu-dark")
	if dark == nil {
		t.Fatal("tofu-dark not registered")
	}
	fmt.Println("dark name:", dark.Name)
	light := styles.Get("tofu")
	fmt.Println("light name:", light.Name)
}
